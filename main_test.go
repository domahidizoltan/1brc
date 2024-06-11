package main

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestAvg(t *testing.T) {
	for i, test := range []struct {
		temps    []float64
		expected string
	}{
		{
			temps: []float64{
				7.8, 0.8, 15.2, 21, 3, 16.1, 23.4, 13.9, 16.2, 16.9, 20.3,
				15.6, 18.7, 6, 13.9, 20, 6.7, 6.5, 17.6, 21.9, -0.8, 17.2,
				19.1, 7.7, 1.8, 15.7, 20.4, 7.7, 15.2, 19.3, 32.7, 15.6,
				13.7, 13.8, 22.4, 11.5, 20.3, 12.5, 10.6, 5.4, 18.2, 15.4,
				12.7, 12.7, 11.8, 14.2, 27.1, 17.9, 22.3, 23.9, 27, 6.8, 24,
				7.4, 5.3, -1.5, 18.6, 15.7, 11.1, 11.1, 14.6, 8.4, 29.1, 8.1,
				26.2, 9.7, 23.6, 9.6, 24.3, 10.1, 21.1, 17.8, 22.4, 13.6,
				12.8, 28.3, 15.8, 12.6, 11.7, 22.1, 11.9, 17.7, 7.3, 3.8,
				23.7, 7.5, 9.3, 11.5, 8.6, 9.8, 23.1, 4.5, 16, 20.7, 8.7,
				12.1, 13.3, 16.5, 6.4, 15.7,
			},
			expected: "14.6", // 14.55
		},
		{
			temps: []float64{
				11.9, 5, 28.3, 13.1, 22.4, 20.7, 17.9, 23.7, 15.7, 13.7, 9.6,
				9.3, 29.4, 23.2, 14.7, 26.7, 16.7, 20.5, 34.6, 25.5, 11.7,
				25.5, 20.4, 25, 13.1, 26.7, 15.2, 11.5, 14.6, 21.9, 19.8,
				20.5, 17.5, 19.6, 12.9, 24.5, 24.3, 18.1, 15.3, 25.9, 19.4,
				22.5, 22.5, 18.6, 12.5, 16.4, 12.8, 21.3, 22.7, 27.1, 5.1,
				19.5, 7.1, 10.2, 19.5, 16.6, 14.1, 29, 11.6, 19.9, 8.8, 26.7,
				28.6, 21.9, 16.9, 9.2, 16.3, 19.8, 9.6, 24.4, 11.9, 20.7, 17.6,
				31.1, 16.1, 15.9, 15, 20.1, 19.5, 17.5, 9, 14.1, 14.2, 25,
				16.9, 19.3, 24.5, 15.8, 2.9, 12.1, 19.3, 11.3, 17.6, 23.8,
				38.8, 20.1, 10.2, 24.1, 13.2, 16.9, 20, 16.1, 17.8, 22.2, 12.8,
				24.5,
			},
			expected: "18.3", // 18.35
		},
		{
			temps: []float64{
				14.5, 16.2, 9.4, 25.5, 21.4, 2, 9.5, 23.1, 26.4, 3.1, 31.6,
				1.6, 14.8, 19.8, 18.2, 19.1, 19.8, 20.2, 24.4, 13.8, 12,
				25.5, 12.2, 23.1, 19.1, 20.7, 1.8, 14.9, 7.4, 24.8, 28.4,
				26.4, 8.5, 14.2, 19.3, 6, 3.3, 1.7, 11.7, 14.6, 15.4, 34.5,
				6.5, 17.6, 19.3, 20.9, 26.7, 23.9, 15.4, 19.1, 28.2, 13.4,
				20.9, 11.5, 24.2, 15.2, 22.6, 20.1, 22.3, 10.8, 11.5, 26.5,
				8.5, 2, 19.9, 19.2, 24.9, 13.3, 12.9, 14.7, 26.5, 14.6, 11.2,
				4.3, 10.4, 4.3, 10.2, 19.3, 12.5, 28.1, 10.9, 23.2, 24.2, 15.9,
				12.6, 13.1, 12.7, 15.7, 18.4, 20.5,
			},
			expected: "16.5", // 16.45
		},
	} {
		t.Run(fmt.Sprintf("Test %d", i), func(t *testing.T) {
			var sum int
			for _, temp := range test.temps {
				sum += int(temp * 10)
			}

			actual := avg(measurement{sum: sum, count: len(test.temps)})
			actualString := fmt.Sprintf("%.1f", actual)
			if actualString != test.expected {
				t.Errorf("expected %s, got %s", test.expected, actualString)
			}
		})
	}
}

func BenchmarkMain(b *testing.B) {
	for i := 0; i < b.N; i++ {
		main()
	}
}

func BenchmarkReadFileLines(b *testing.B) {
	b.ReportAllocs()

	fileContentStr := `maH;6.9
Nerupperichc;13.3
gsheGuyuanRui’anKhulnaMuscatWenlingGaoz;9.2
PupriMajīthaWest DraytonDhama;10.8
niSu;14.8`

	fileContent := strings.NewReader(fileContentStr)
	res := []byte{}
	for i := 0; i < b.N; i++ {
		for line := range readFileLines(fileContent) {
			res = line
		}
	}
	_ = res
}

func BenchmarkProcessMeasurements(b *testing.B) {
	b.ReportAllocs()

	res := map[string]measurement{}

	for i := 0; i < b.N; i++ {
		lines := `gsheGuyuanRui’anKhulnaMuscatWenlingGaoz;9.2
			test;1.0
			gsheGuyuanRui’anKhulnaMuscatWenlingGaoz;9.0
			test;2.0
			gsheGuyuanRui’anKhulnaMuscatWenlingGaoz;9.1`
		res = processMeasurements(lines)
	}
	_ = res
}

func BenchmarkGetMeasurements(b *testing.B) {
	b.ReportAllocs()
	maxMeasurementWorkers = 1
	processMeasurementsFunc = func(string) map[string]measurement {
		return map[string]measurement{
			"gsheGuyuanRui’anKhulnaMuscatWenlingGaoz": {
				min: 90, max: 92, sum: 273, count: 3,
			},
			"test": {
				min: 10, max: 20, sum: 30, count: 2,
			},
		}
	}

	res := map[string]measurement{}
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		wg.Add(1)
		linesCh := make(chan []byte)
		go func(linesCh chan []byte) {
			linesCh <- []byte(`gsheGuyuanRui’anKhulnaMuscatWenlingGaoz;9.2

test;1.0
gsheGuyuanRui’anKhulnaMuscatWenlingGaoz;9.0
test;2.0
gsheGuyuanRui’anKhulnaMuscatWenlingGaoz;9.1`)

			close(linesCh)
		}(linesCh)
		res = getMeasurements(linesCh, &wg)
		wg.Wait()
	}
	_ = res
}
