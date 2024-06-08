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

type measurement2 struct {
	min, max int16
	count    uint32
	sum      int64
}

const (
	resFormat   = "%s=%.1f/%.1f/%.1f"
	l1CacheSize = 64 * 1024
)

var (
	maxMeasurementWorkers = runtime.NumCPU() - 1
	fileLinesBufferSize   = runtime.NumCPU() * 10000
	chunksBufferSize      = 1000
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

	linesCh := readFileLines(f)

	// 19.47
	var wg sync.WaitGroup
	wg.Add(maxMeasurementWorkers)
	measurements := getMeasurements2(linesCh, &wg)
	wg.Wait()

	res := getResult(measurements)
	// _ = res
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

var (
	processMeasurementsFunc  func(chan string) map[string]measurement       = processMeasurements
	processMeasurements2Func func(chan string) chan map[string]measurement2 = processMeasurements2
)

// TODO remove
func getMeasurements(linesCh chan []byte, wg *sync.WaitGroup) map[string]measurement {
	workerCh := make([]chan string, maxMeasurementWorkers)
	resCh := make(chan map[string]measurement, maxMeasurementWorkers)
	for i := 0; i < maxMeasurementWorkers; i++ {
		workerCh[i] = make(chan string, int(fileLinesBufferSize/maxMeasurementWorkers))
		go func(wCh chan string, idx int) {
			defer wg.Done()
			resCh <- processMeasurementsFunc(wCh)
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

func getMeasurements2(chunksCh chan []byte, wg *sync.WaitGroup) map[string]measurement2 {
	workerChans := make([]chan string, maxMeasurementWorkers)
	resCh := make(chan map[string]measurement2, maxMeasurementWorkers*chunksBufferSize)
	for i := 0; i < maxMeasurementWorkers; i++ {
		workerChans[i] = make(chan string, int(chunksBufferSize))
		go func(wCh chan string, wg *sync.WaitGroup) {
			defer wg.Done()
			for res := range processMeasurements2Func(wCh) {
				resCh <- res
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

	measurements := map[string]measurement2{}
	var resWg sync.WaitGroup
	resWg.Add(1)
	go func(resCh chan map[string]measurement2) {
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

type workerResult struct {
	result map[string]measurement2
	m      sync.RWMutex
}

func mergeResults(existing, latest map[string]measurement2) {
	for k, v := range latest {
		if m, ok := existing[k]; !ok {
			existing[k] = v
		} else {
			m.min = min(m.min, v.min)
			m.max = max(m.max, v.max)
			m.sum += v.sum
			m.count += v.count
			existing[k] = m
		}
	}
}

// TODO remove
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

func processMeasurementCore(chunk string) map[string]measurement2 {
	measurements := make(map[string]measurement2, 100)
	lines := strings.Split(chunk, "\n")
	for _, line := range lines {
		idx := strings.Index(line, ";")
		tokens := [2]string{line[:idx], line[idx+1:]}

		station := tokens[0]
		// temp := tempToInt(tokens[1])
		t, err := strconv.ParseFloat(tokens[1], 64)
		if err != nil {
			panic(err)
		}
		temp := int16(t * 10)

		if m, ok := measurements[station]; !ok {
			measurements[station] = measurement2{temp, temp, 1, int64(temp)}
		} else {
			m.min = min(m.min, temp)
			m.max = max(m.max, temp)
			m.sum += int64(temp)
			m.count++
			measurements[station] = m
		}
	}

	return measurements
}

func processMeasurements2(chunksCh chan string) chan map[string]measurement2 {
	measurements := make(chan map[string]measurement2, 100)

	go func(chunksCh chan string, measurements chan map[string]measurement2) {
		defer close(measurements)
		for chunk := range chunksCh {
			measurements <- processMeasurementCore(chunk)
		}
	}(chunksCh, measurements)

	return measurements
}

// func strSplit(s, delim string) []string {
// 	count := strings.Count(s, delim)
// 	l := len(delim)
// 	res := make([]string, count+1)
// 	for i := range count {
// 		idx := strings.Index(s, delim)
// 		tmp := s[:idx-1]
// 		res[i] = string(tmp)
// 		s = s[idx+l:]
// 	}
// 	res[count] = s
// 	return res
// }

// TODO do we need this, check diff without this and measurement2
func tempToInt(tempStr string) int16 {
	var temp int16
	var isNegative bool
	for i := range tempStr {
		var t int16
		switch tempStr[i] {
		case '0':
			t = 0
		case '1':
			t = 1
		case '2':
			t = 2
		case '3':
			t = 3
		case '4':
			t = 4
		case '5':
			t = 5
		case '6':
			t = 6
		case '7':
			t = 7
		case '8':
			t = 8
		case '9':
			t = 9
		case '-':
			isNegative = true
			continue
		default:
			continue
		}
		temp = (temp * 10) + t

	}
	if isNegative {
		temp = -temp
	}
	return temp
}

func getResult(measurements map[string]measurement2) string {
	stations := make([]string, 0, len(measurements))
	for station := range measurements {
		stations = append(stations, station)
	}
	slices.Sort(stations)

	res := make([]string, 0, len(stations))
	for _, station := range stations {
		mes := measurements[string(station)]
		res = append(res, fmt.Sprintf(resFormat, station, float32(mes.min)/10.0, avg(mes), float32(mes.max)/10.0))
	}
	return fmt.Sprintf("{%s}", strings.Join(res, ", "))
}

func avg(mes measurement2) float64 {
	x := float64(mes.sum) / 10.0 / float64(mes.count)
	avg := math.Round(x*10) / 10
	return avg
}
