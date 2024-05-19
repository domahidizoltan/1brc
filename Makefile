run:
	go build -o 1brc main.go
	./1brc > tmp.txt
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
