package main

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type measurement struct {
	min, max   float64
	sum, count int
}

const (
	resFormat = "%s=%.1f/%.1f/%.1f"
)

var (
	maxMeasurementWorkers = runtime.NumCPU()
	fileLinesBufferSize   = runtime.NumCPU() * 10000
	chunksBufferSize      = 200

	l3CacheSize = 4 * 1024 * 1024
	args        = os.Args
)

func main() {
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	l3CacheSize = int(mem.HeapSys)

	var printTime bool
	var noOutput bool
	if len(args) > 1 {
		switch args[1] {
		case "noOutput":
			noOutput = true
		case "withTime":
			printTime = true

		}
	}

	start := time.Now()

	f, err := os.Open("measurements.txt")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	linesCh := readFileLines(f)

	var wg sync.WaitGroup
	wg.Add(maxMeasurementWorkers)
	measurements := getMeasurements(linesCh, &wg)
	wg.Wait()

	res := getResult(measurements)

	if noOutput {
		return
	}
	fmt.Println(res)

	if printTime {
		fmt.Println(time.Since(start))
	}
}

func readFileLines(f io.Reader) chan []byte {
	ch := make(chan []byte, fileLinesBufferSize)

	go func(f io.Reader) {
		defer close(ch)
		tmpLine := []byte(nil)
		buf := make([]byte, l3CacheSize-1024)
		for {
			n, err := f.Read(buf)
			if err != nil {
				if err != io.EOF {
					panic(err)
				}
				break
			}

			lastIdx := bytes.LastIndex(buf[:n], []byte("\n"))
			chunk := append([]byte(nil), tmpLine...)
			if lastIdx > -1 {
				chunk = append(chunk, buf[:lastIdx]...)
				tmpLine = append([]byte(nil), buf[lastIdx+1:n]...)
			} else {
				chunk = append(chunk, buf[:n]...)
				tmpLine = []byte(nil)
			}

			ch <- chunk
		}
	}(f)
	return ch
}

var processMeasurementsFunc func(string) map[string]measurement = processMeasurements

func getMeasurements(chunksCh chan []byte, wg *sync.WaitGroup) map[string]measurement {
	workerChans := make([]chan string, maxMeasurementWorkers)
	resCh := make(chan map[string]measurement, maxMeasurementWorkers*chunksBufferSize)
	for i := 0; i < maxMeasurementWorkers; i++ {
		workerChans[i] = make(chan string, int(chunksBufferSize))
		go func(wCh chan string, wg *sync.WaitGroup) {
			defer wg.Done()
			for c := range wCh {
				resCh <- processMeasurementsFunc(c)
			}
		}(workerChans[i], wg)
	}

	go func(chunksCh chan []byte) {
		counter := 0
		for chunk := range chunksCh {
			idx := counter % maxMeasurementWorkers
			workerChans[idx] <- string(chunk)
			counter++
		}
		for i := 0; i < maxMeasurementWorkers; i++ {
			close(workerChans[i])
		}
	}(chunksCh)

	measurements := map[string]measurement{}
	var resWg sync.WaitGroup
	resWg.Add(1)
	go func(resCh chan map[string]measurement) {
		for m := range resCh {
			for k, v := range m {
				if m, ok := measurements[k]; !ok {
					measurements[k] = v
				} else {
					m.min = min(m.min, v.min)
					m.max = max(m.max, v.max)
					m.sum += v.sum
					m.count += v.count
					measurements[k] = m

				}
			}
		}
		resWg.Done()
	}(resCh)

	wg.Wait()
	close(resCh)
	resWg.Wait()

	return measurements
}

func processMeasurements(chunk string) map[string]measurement {
	measurements := make(map[string]measurement, 100)
	lines := strings.Split(chunk, "\n")
	for _, line := range lines {
		idx := strings.Index(line, ";")
		tokens := [2]string{line[:idx], line[idx+1:]}

		station := tokens[0]
		t, err := strconv.ParseFloat(tokens[1], 64)
		if err != nil {
			panic(err)
		}

		temp := int(t * 10)
		if m, ok := measurements[station]; !ok {
			measurements[station] = measurement{t, t, temp, 1}
		} else {
			m.min = min(m.min, t)
			m.max = max(m.max, t)
			m.sum += temp
			m.count++
			measurements[station] = m
		}
	}

	return measurements
}

func getResult(measurements map[string]measurement) string {
	stations := make([]string, 0, len(measurements))
	for station := range measurements {
		stations = append(stations, station)
	}
	slices.Sort(stations)

	res := make([]string, 0, len(stations))
	for _, station := range stations {
		mes := measurements[string(station)]
		res = append(res, fmt.Sprintf(resFormat, station, mes.min, avg(mes), mes.max))
	}
	return fmt.Sprintf("{%s}", strings.Join(res, ", "))
}

func avg(mes measurement) float64 {
	x := float64(mes.sum) / 10.0 / float64(mes.count)
	avg := math.Round(x*10) / 10
	return avg
}
