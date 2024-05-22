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
	minS, maxS string
	min, max   float32
	sum, count int32
}

const resFormat = ", %s=%.1f/%.1f/%.1f"

func main() {
	var (
		stations     []string
		measurements = map[string]measurement{}
	)

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

		t, err := strconv.ParseFloat(tokens[1], 32)
		if err != nil {
			panic(err)
		}

		temp := int32(t * 10)
		if m, ok := measurements[tokens[0]]; !ok {
			stations = append(stations, tokens[0])
			mes := measurement{
				min:   float32(t),
				minS:  tokens[1],
				max:   float32(t),
				maxS:  tokens[1],
				sum:   temp,
				count: 1,
			}
			measurements[tokens[0]] = mes
		} else {
			m.min = float32(math.Min(float64(m.min), float64(t)))
			m.minS = tokens[1]
			m.max = float32(math.Max(float64(m.max), float64(t)))
			m.maxS = tokens[1]
			m.sum += temp
			m.count++
			measurements[tokens[0]] = m
		}

	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	res := getResult2(stations, measurements)
	fmt.Println(res)
	fmt.Println(time.Since(start))
}

func avg(mes measurement) float64 {
	x := float64(mes.sum) / 10.0 / float64(mes.count)
	avg := math.Round(x*10) / 10
	return avg
}

// TODO optimise measurements struct to avoid false sharing
// TODO benchstat and comment that getResult was reduced by 8.1x
// TODO fix cmp errors
// TODO make it parallel and document the loss (probably it will be slower on unit level)
func getResult2(stations []string, measurements map[string]measurement) string {
	slices.Sort(stations)

	var builder strings.Builder
	builder.Grow(2 + len(stations)*120)
	builder.WriteString("{")

	mes := measurements[stations[0]]
	formatMeasurement(&builder, stations[0], mes)
	for _, station := range stations[1:] {
		mes := measurements[string(station)]
		builder.WriteString(", ")
		formatMeasurement(&builder, station, mes)
	}

	builder.WriteString("}")
	return builder.String()
}

func formatMeasurement(builder *strings.Builder, station string, mes measurement) {
	builder.WriteString(station)
	builder.WriteString("=")
	builder.WriteString(mes.minS)
	builder.WriteString("/")
	formatFloat(builder, avg(mes))
	builder.WriteString("/")
	builder.WriteString(mes.maxS)
}

func formatFloat(builder *strings.Builder, f float64) {
	i := int(f * 10.0)
	builder.WriteString(strconv.Itoa(i / 10))
	builder.WriteString(".")
	builder.WriteString(strconv.Itoa(i % 10))
}

func getResult(stations []string, measurements map[string]measurement) string {
	slices.Sort(stations)
	mes := measurements[stations[0]]
	res := fmt.Sprintf("{%s=%.1f/%.1f/%.1f", stations[0], mes.min, avg(mes), mes.max)
	for _, station := range stations[1:] {
		mes := measurements[string(station)]
		res += fmt.Sprintf(resFormat, station, mes.min, avg(mes), mes.max)
	}
	res += "}"

	return res
}
