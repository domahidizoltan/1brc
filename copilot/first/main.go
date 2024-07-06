package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type StationData struct {
	min, max         float64
	totalTemp, count int
}

var (
	stationTemperatures = make(map[string]*StationData)
	mu                  sync.Mutex
)

func processLine(line string) {
	parts := strings.Split(line, ";")
	if len(parts) != 2 {
		return // Skip invalid lines
	}
	station, tempStr := parts[0], parts[1]
	t, err := strconv.ParseFloat(tempStr, 64)
	if err != nil {
		return // Skip lines with invalid temperature values
	}
	temp := int(t * 10)

	mu.Lock()
	defer mu.Unlock()
	if data, exists := stationTemperatures[station]; exists {
		data.min = math.Min(data.min, t)
		data.max = math.Max(data.max, t)
		data.totalTemp += temp
		data.count++
	} else {
		stationTemperatures[station] = &StationData{min: t, max: t, totalTemp: temp, count: 1}
	}
}

func main() {
	var printTime bool
	args := os.Args
	if len(args) > 1 {
		switch args[1] {
		case "withTime":
			printTime = true
		}
	}

	start := time.Now()

	file, err := os.Open("measurements.txt")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		processLine(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	var stations []string
	for station := range stationTemperatures {
		stations = append(stations, station)
	}
	sort.Strings(stations)

	fmt.Print("{")
	for i, station := range stations {
		data := stationTemperatures[station]
		fmt.Printf("%s=%.1f/%.1f/%.1f", station, data.min, avg(data), data.max)
		if i < len(stations)-1 {
			fmt.Print(", ")
		}
	}
	fmt.Print("}\n")

	if printTime {
		fmt.Println(time.Since(start))
	}
}

func avg(data *StationData) float64 {
	x := float64(data.totalTemp) / 10.0 / float64(data.count)
	avg := math.Round(x*10) / 10
	return avg
}
