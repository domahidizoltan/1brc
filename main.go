package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

type measurement struct {
	min, max   float64
	sum, count int
}

var (
	stations     []string
	measurements = map[string]measurement{}
)

const resFormat = ", %s=%.1f/%.1f/%.1f"

func main() {
	start := time.Now()
	f, err := os.Open("measurements.txt")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Split(line, ";")

		t, err := strconv.ParseFloat(tokens[1], 64)
		if err != nil {
			panic(err)
		}

		temp := int(t * 10)
		if m, ok := measurements[tokens[0]]; !ok {
			stations = append(stations, tokens[0])
			measurements[tokens[0]] = measurement{t, t, temp, 1}
		} else {
			m.min = math.Min(m.min, t)
			m.max = math.Max(m.max, t)
			m.sum += temp
			m.count++
			measurements[tokens[0]] = m
		}

	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	slices.Sort(stations)
	mes := measurements[stations[0]]
	res := fmt.Sprintf("{%s=%.1f/%.1f/%.1f", stations[0], mes.min, avg(mes), mes.max)
	for _, station := range stations[1:] {
		mes := measurements[string(station)]
		res += fmt.Sprintf(resFormat, station, mes.min, avg(mes), mes.max)
	}
	res += "}"

	fmt.Println(res)
	fmt.Println(time.Since(start))
}

func avg(mes measurement) float64 {
	x := float64(mes.sum) / 10.0 / float64(mes.count)
	avg := math.Round(x*10) / 10
	return avg
}
