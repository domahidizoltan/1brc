run:
	go build -o 1brc main.go
	./1brc withTime > tmp.txt
	head -n1 tmp.txt > averages.txt
	tail -n1 tmp.txt
	rm 1brc tmp.txt

use-1b:
	ln -fs files/measurements_1B.txt measurements.txt

use-1m:
	ln -fs files/measurements_1M.txt measurements.txt

diff-1b:
	cmp -l averages.txt files/avg_baseline_1B.txt

diff-1m:
	cmp -l averages.txt files/avg_baseline_1M.txt

profile:
	rm -f /tmp/profile.* /tmp/trace.out
	go test -bench BenchmarkMain -count=1 -cpuprofile=/tmp/profile.cpu.out -memprofile=/tmp/profile.mem.out -blockprofile=/tmp/profile.block.out -v -trace=/tmp/trace.out

profile-test:
	rm -f /tmp/profile.* /tmp/trace.out
	go test -bench $(test) -count=1 -cpuprofile=/tmp/profile.cpu.out -memprofile=/tmp/profile.mem.out -blockprofile=/tmp/profile.block.out -v -trace=/tmp/trace.out

#mode is one of cpu, mem or block. Usage: mode=block make pprof
pprof:
	go tool pprof /tmp/profile.$(mode).out

flame:
	go tool pprof -http=:8080 /tmp/profile.$(mode).out

trace:
	go tool trace /tmp/trace.out

bench:
	go test -bench=$(test) -count=6 > stats.txt

#golang.org/x/perf/cmd/benchstat@latest
benchstat:
	benchstat stats.txt stats.old.txt
