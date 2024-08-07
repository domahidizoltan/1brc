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
	maxStations = 10000
)

var (
	maxMeasurementWorkers = runtime.NumCPU()
	fileLinesBufferSize   = runtime.NumCPU() * maxStations
	chunksBufferSize      = 200

	l3CacheSize = 4 * 1024 * 1024
	args        = os.Args
)

func init() {
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	l3CacheSize = int(mem.HeapSys)
}

func main() {
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

	info, err := f.Stat()
	if err != nil {
		panic(err)
	}
	fileSize := info.Size()

	measurements := getMeasurements(fileSize)

	res := getResult(measurements)

	if noOutput {
		return
	}
	fmt.Println(res)

	if printTime {
		fmt.Println(time.Since(start))
	}
}

func getMeasurements(fileSize int64) map[string]measurement {
	width := fileSize / int64(maxMeasurementWorkers)

	workerMeasurementsCh := make(chan map[string]measurement, maxMeasurementWorkers)
	leftovers := make([][]byte, 2*maxMeasurementWorkers)
	var wg sync.WaitGroup
	wg.Add(maxMeasurementWorkers)
	for i := range maxMeasurementWorkers {
		go readAndProcessFile(i, width, leftovers, workerMeasurementsCh, &wg)
	}

	doneCh := make(chan struct{})
	measurements := make(map[string]measurement, maxStations)
	go func(workerMeasurementsCh chan map[string]measurement) {
		for workerResults := range workerMeasurementsCh {
			for station, mes := range workerResults {
				if v, ok := measurements[station]; !ok {
					measurements[station] = mes
				} else {
					v.min = min(v.min, mes.min)
					v.max = max(v.max, mes.max)
					v.sum += mes.sum
					v.count += mes.count
					measurements[station] = v
				}
			}
		}
		doneCh <- struct{}{}
	}(workerMeasurementsCh)

	wg.Wait()
	close(workerMeasurementsCh)
	<-doneCh

	leftoverLines := make([]string, maxMeasurementWorkers-1)
	for i, j := 1, 0; i < 2*maxMeasurementWorkers-1; i, j = i+2, j+1 {
		line := []byte(nil)
		if len(leftovers[i]) > 0 {
			line = append(line, leftovers[i]...)
		}
		if len(leftovers[i+1]) > 0 {
			line = append(line, leftovers[i+1]...)
		}
		leftoverLines[j] = string(line)
	}

	for station, m := range processMeasurements(strings.Join(leftoverLines, "\n")) {
		if v, ok := measurements[station]; !ok {
			measurements[station] = m
		} else {
			v.min = min(v.min, m.min)
			v.max = max(v.max, m.max)
			v.sum += m.sum
			v.count += m.count
			measurements[station] = v
		}
	}

	return measurements
}

func readAndProcessFile(idx int, width int64, leftovers [][]byte, workerMeasurementsCh chan map[string]measurement, wg *sync.WaitGroup) {
	f, err := os.Open("measurements.txt")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	from := width * int64(idx)
	if _, err := f.Seek(from, 0); err != nil {
		panic(err)
	}

	measurements := make(map[string]measurement, maxStations)
	tmpLine := []byte(nil)
	buf := make([]byte, l3CacheSize-1024)

	var totalRead int64
	getFirstLeftover, getLastLeftover := true, false
	for {
		read, err := f.Read(buf)
		n := int64(read)
		if err != nil {
			if err != io.EOF {
				panic(err)
			}
			break
		}

		if totalRead+n >= width {
			n = width - totalRead
			getLastLeftover = true
		}

		totalRead += n

		lastIdx := bytes.LastIndex(buf[:n], []byte("\n"))
		chunk := append([]byte(nil), tmpLine...)
		if lastIdx > -1 {
			chunk = append(chunk, buf[:lastIdx]...)
			tmpLine = append([]byte(nil), buf[lastIdx+1:n]...)
		} else {
			chunk = append(chunk, buf[:n]...)
			tmpLine = []byte(nil)
		}

		if getFirstLeftover && idx > 0 {
			firstIdx := bytes.Index(chunk, []byte("\n"))
			leftovers[2*idx] = append([]byte(nil), chunk[:firstIdx]...)
			chunk = chunk[firstIdx+1:]
			getFirstLeftover = false
		}

		if getLastLeftover {
			leftovers[2*idx+1] = append([]byte(nil), tmpLine...)
		}

		for station, m := range processMeasurements(string(chunk)) {
			if v, ok := measurements[station]; !ok {
				measurements[station] = m
			} else {
				v.min = min(v.min, m.min)
				v.max = max(v.max, m.max)
				v.sum += m.sum
				v.count += m.count
				measurements[station] = v
			}
		}

		if getLastLeftover {
			break
		}
	}

	workerMeasurementsCh <- measurements
	wg.Done()
}

func processMeasurements(chunk string) map[string]measurement {
	lines := strings.Split(chunk, "\n")
	allocSize := min(len(lines), maxStations)
	measurements := make(map[string]measurement, allocSize)

	for _, line := range lines {
		station, tempStr, _ := strings.Cut(line, ";")
		t, err := strconv.ParseFloat(tempStr, 64)
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

	formatFloat := func(builder *strings.Builder, f float64) {
		if f < 0 {
			builder.WriteString("-")
			f = -f
		}

		i := int(f * 10.0)
		builder.WriteString(strconv.Itoa(i / 10))
		builder.WriteString(".")
		builder.WriteString(strconv.Itoa(i % 10))
	}

	var builder strings.Builder
	builder.Grow(2 + len(stations)*122)
	builder.WriteString("{")

	lastIdx := len(stations) - 1
	for i, station := range stations {
		mes := measurements[station]

		builder.WriteString(station)
		builder.WriteString("=")
		formatFloat(&builder, mes.min)
		builder.WriteString("/")
		formatFloat(&builder, avg(mes))
		builder.WriteString("/")
		formatFloat(&builder, mes.max)

		if i != lastIdx {
			builder.WriteString(", ")
		}
	}

	builder.WriteString("}")
	return builder.String()
}

func avg(mes measurement) float64 {
	x := float64(mes.sum) / 10.0 / float64(mes.count)
	avg := math.Round(x*10) / 10
	return avg
}
