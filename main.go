package main

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"

	// "runtime"
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
	resFormat   = "%s=%.1f/%.1f/%.1f"
	l1CacheSize = 64 * 1024
)

var (
	maxMeasurementWorkers = runtime.NumCPU()
	fileLinesBufferSize   = runtime.NumCPU() * 10000
)

func main() {
	var printTime bool
	if len(os.Args) > 1 {
		if os.Args[1] == "withTime" {
			printTime = true
		}
	}

	start := time.Now()

	f, err := os.Open("measurements.txt")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var wg sync.WaitGroup
	wg.Add(maxMeasurementWorkers)

	linesCh := readFileLines(f)

	measurements := getMeasurements(linesCh, &wg)

	wg.Wait()

	res := getResult(measurements)
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
		buf := make([]byte, l1CacheSize-1024)
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

func getMeasurements(linesCh chan []byte, wg *sync.WaitGroup) map[string]measurement {
	workerCh := make([]chan string, maxMeasurementWorkers)
	resCh := make(chan map[string]measurement, maxMeasurementWorkers)
	for i := 0; i < maxMeasurementWorkers; i++ {
		workerCh[i] = make(chan string, int(fileLinesBufferSize/maxMeasurementWorkers))
		go func(wCh chan string, idx int) {
			defer wg.Done()
			resCh <- processMeasurements(wCh)
		}(workerCh[i], i)
	}

	counter := 0
	for chunk := range linesCh {
		idx := counter % maxMeasurementWorkers
		for _, line := range strings.Split(string(chunk), "\n") {
			workerCh[idx] <- line
		}
		counter++
	}
	for i := 0; i < maxMeasurementWorkers; i++ {
		close(workerCh[i])
	}

	measurements := map[string]measurement{}
	for i := 0; i < maxMeasurementWorkers; i++ {
		m := <-resCh
		for k, v := range m {
			if m, ok := measurements[k]; !ok {
				measurements[k] = v
			} else {
				m.min = math.Min(m.min, v.min)
				m.max = math.Max(m.max, v.max)
				m.sum += v.sum
				m.count += v.count
				measurements[k] = m

			}
		}
	}
	return measurements
}

func processMeasurements(linesCh chan string) map[string]measurement {
	measurements := map[string]measurement{}
	var wg sync.WaitGroup
	wg.Add(1)

	go func(linesCh chan string) {
		for line := range linesCh {
			tokens := strings.Split(line, ";")
			t, err := strconv.ParseFloat(tokens[1], 64)
			if err != nil {
				panic(err)
			}

			temp := int(t * 10)
			if m, ok := measurements[tokens[0]]; !ok {
				measurements[tokens[0]] = measurement{t, t, temp, 1}
			} else {
				m.min = math.Min(m.min, t)
				m.max = math.Max(m.max, t)
				m.sum += temp
				m.count++
				measurements[tokens[0]] = m
			}
		}
		wg.Done()
	}(linesCh)

	wg.Wait()
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
