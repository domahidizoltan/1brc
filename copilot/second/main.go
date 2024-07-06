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

const (
	// The maximum number of unique station names.
	maxStationNames = 10000
	// The maximum length of a station name.
	maxStationNameLength = 100
	// The maximum length of a line.
	maxLineLength = 100 + 3 // 100 bytes for station name + 3 bytes for temperature value + 1 byte for '\n'
	// The maximum number of goroutines to use.
	maxGoroutines = 2 * maxStationNames
)

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

	// Read the file.
	file, err := os.Open("measurements.txt")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Read the file line by line.
	scanner := bufio.NewScanner(file)

	// Create a map to store the station names and their temperature values.
	stations := make(map[string][]float64)

	// Create a mutex to synchronize access to the stations map.
	var mutex sync.Mutex

	// Create a wait group to wait for all goroutines to finish.
	var wg sync.WaitGroup

	// Create a channel to limit the number of goroutines.
	semaphore := make(chan struct{}, maxGoroutines)

	// Read each line of the file.
	for scanner.Scan() {
		// Get the line.
		line := scanner.Text()

		// Split the line by ';'.
		parts := strings.Split(line, ";")
		if len(parts) != 2 {
			continue
		}

		// Get the station name.
		station := parts[0]

		// Get the temperature value.
		temperature, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			continue
		}

		// Add the station name and temperature value to the map.
		semaphore <- struct{}{}
		wg.Add(1)
		go func(station string, temperature float64) {
			defer wg.Done()
			defer func() { <-semaphore }()

			// Lock the mutex.
			mutex.Lock()
			defer mutex.Unlock()

			// Add the temperature value to the station.
			stations[station] = append(stations[station], temperature)
		}(station, temperature)
	}

	// Wait for all goroutines to finish.
	wg.Wait()

	// Calculate the min, mean, and max temperature values for each station.
	results := make(map[string]string)
	for station, temperatures := range stations {
		min, mean, max := calculateTemperatureStats(temperatures)
		result := fmt.Sprintf("%.1f/%.1f/%.1f", min, mean, max)
		results[station] = result
	}

	// Sort the stations alphabetically.
	sortedStations := make([]string, 0, len(results))
	for station := range results {
		sortedStations = append(sortedStations, station)
	}
	sort.Strings(sortedStations)

	// Print the results.
	fmt.Print("{")
	for i, station := range sortedStations {
		result := results[station]
		fmt.Printf("%s=%s", station, result)
		if i < len(sortedStations)-1 {
			fmt.Print(", ")
		}
	}
	fmt.Println("}")

	if printTime {
		fmt.Println(time.Since(start))
	}
}

// Function to calculate the min, mean, and max temperature values.
func calculateTemperatureStats(temperatures []float64) (float64, float64, float64) {
	min := temperatures[0]
	max := temperatures[0]
	sum := 0

	for _, temperature := range temperatures {
		if temperature < min {
			min = temperature
		}
		if temperature > max {
			max = temperature
		}
		sum += int(temperature * 10)
	}

	x := float64(sum) / 10.0 / float64(len(temperatures))
	mean := math.Round(x*10) / 10

	return min, mean, max
}
