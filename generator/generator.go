package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	totalStations  = 10000
	tempFilePrefix = "temp_"
)

var (
	stations   = make([][]byte, totalStations)
	maxWorkers = runtime.NumCPU()

	totalRows    = 1_000_000
	l3CacheSize  = 4 * 1024 * 1024
	maxChunkSize = 32 * 1024

	args = os.Args
)

func init() {
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	l3CacheSize = int(mem.HeapSys) - 1024
	maxChunkSize = int(mem.MCacheSys) - 1024
}

func main() {
	if len(args) > 1 {
		var err error
		if totalRows, err = strconv.Atoi(args[1]); err != nil {
			panic(err)
		}
	}

	if totalRows <= 0 {
		panic("invalid row number")
	}

	started := time.Now()
	defer func(started time.Time) {
		fmt.Println("finished running in", time.Since(started))
	}(started)

	quitSig := make(chan os.Signal, 1)
	signal.Notify(quitSig, os.Interrupt)

	if err := populateStations(); err != nil {
		panic(err)
	}

	workerRows := totalRows / maxWorkers
	remainder := totalRows % maxWorkers

	var wg sync.WaitGroup
	wg.Add(maxWorkers)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-quitSig
		fmt.Println("\ninterrupted")
		cancel()
	}()

	chunksCh := make(chan string, maxWorkers*1000)

	for i := range maxWorkers {
		go generateChunks(chunksCh, i, workerRows+remainder, ctx, &wg)
		if i == 0 {
			remainder = 0
		}
	}

	doneCh, err := generateFile(chunksCh, ctx)
	if err != nil {
		panic(err)
	}

	wg.Wait()
	close(chunksCh)

	<-doneCh
}

func populateStations() error {
	f, err := os.Open("stations.txt")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}

	lines := bytes.Split(content, []byte("\n"))
	rand.Shuffle(len(lines), func(i, j int) {
		lines[i], lines[j] = lines[j], lines[i]
	})
	copy(stations, lines[:totalStations])

	return nil
}

func generateChunks(outCh chan string, workerIdx, totalRows int, ctx context.Context, wg *sync.WaitGroup) {
	builder := strings.Builder{}
	builder.Grow(maxChunkSize)

	for i := range totalRows {
		if ctx.Err() != nil {
			fmt.Printf("\nworker %d canceled", workerIdx)
			wg.Done()
			return
		}

		nextIdx := rand.Int31n(totalStations)

		temp := rand.Int31n(350)
		if i%1000 == 0 {
			temp += rand.Int31n(155)
		}
		if temp <= 250 && rand.Int31n(2) == 0 {
			temp = -temp
		}

		builder.Write(stations[nextIdx])
		builder.WriteByte(';')
		builder.Write(formatTemp(int(temp)))
		builder.WriteByte('\n')

		if builder.Len() > (maxChunkSize) || i == totalRows-1 {
			outCh <- builder.String()
			builder.Reset()
		}

	}

	wg.Done()
}

func formatTemp(temp int) []byte {
	f := make([]byte, 5)
	idx := 0

	if temp < 0 {
		f[idx] = '-'
		temp = -temp
		idx++
	}

	fraction := temp % 10
	temp /= 10

	formatDigit := func(digit int) byte {
		switch digit {
		case 1:
			return '1'
		case 2:
			return '2'
		case 3:
			return '3'
		case 4:
			return '4'
		case 5:
			return '5'
		case 6:
			return '6'
		case 7:
			return '7'
		case 8:
			return '8'
		case 9:
			return '9'
		default:
			return '0'
		}
	}

	if temp > 10 {
		f[idx] = formatDigit(temp / 10)
		idx++
	}

	f[idx] = formatDigit(temp % 10)
	idx++

	f[idx] = '.'
	idx++
	f[idx] = formatDigit(fraction)
	idx++

	return f[:idx]
}

func generateFile(outCh chan string, ctx context.Context) (chan struct{}, error) {
	f, err := os.Create("measurements.txt")
	if err != nil {
		return nil, err
	}

	writer := bufio.NewWriterSize(f, l3CacheSize)

	doneCh := make(chan struct{})
	go func(writer *bufio.Writer, f *os.File, doneCh chan struct{}) {
		builder := strings.Builder{}
		builder.Grow(l3CacheSize)

		writeAndFlush := func(builder *strings.Builder, writer *bufio.Writer) {
			if _, err := writer.WriteString(builder.String()); err != nil {
				panic(err)
			}
			builder.Reset()
			if err := writer.Flush(); err != nil {
				panic(err)
			}
		}

		for chunk := range outCh {
			if ctx.Err() != nil {
				fmt.Printf("\nfile generation canceled")
				doneCh <- struct{}{}
				return
			}

			if _, err := builder.WriteString(chunk); err != nil {
				panic(err)
			}

			if builder.Len() > maxChunkSize {
				writeAndFlush(&builder, writer)
			}
		}

		writeAndFlush(&builder, writer)

		f.Close()
		doneCh <- struct{}{}
	}(writer, f, doneCh)

	return doneCh, nil
}
