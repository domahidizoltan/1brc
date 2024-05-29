# 1BRC: One Billion Row Challenge

WIP Learning high performance Go by doing the One Billion Row Challenge

Find more details about:
- the One Billion Row Challenge at https://1brc.dev/  
- the leaderboard and how to generate or run the solutions https://github.com/gunnarmorling/1brc  


My system information:  
```sh
$ lscpu | sed -n '1p;5p;8p;11p;20,23p'
Architecture:                       x86_64
CPU(s):                             4
Model name:                         Intel(R) Core(TM) i7-7500U CPU @ 2.70GHz
Thread(s) per core:                 2
L1d cache:                          64 KiB (2 instances)
L1i cache:                          64 KiB (2 instances)
L2 cache:                           512 KiB (2 instances)
L3 cache:                           4 MiB (1 instance)

$ free -g -h -t | grep Total | cut -b 17-20
17Gi
``` 

---

Reference implementations (on my machine):  

Execution time for 1B row
```sh
$ time ./calculate_average_baseline.sh > /dev/null  
378,93s user 12,82s system 99% cpu 6:32,96 total

$ time ./calculate_average_thomaswue.sh > /dev/null  
0,23s user 0,17s system 0% cpu 1:02,58 total

$ time ./calculate_average_AlexanderYastrebov.sh > /dev/null  
167,48s user 15,02s system 307% cpu 59,309 total

```

Execution time for 1M row
```sh
$ time ./calculate_average_baseline.sh  
1,39s user 0,30s system 149% cpu 1,132 total

$ time ./calculate_average_thomaswue.sh > /dev/null  
1,16s user 0,22s system 200% cpu 0,689 total

$ time ./calculate_average_AlexanderYastrebov.sh  
0,24s user 0,10s system 171% cpu 0,194 total

```
---

For reference:
- 1BRC Baseline: 392.96s
- Thomas Wuerthinger: 62.58s
- Alexander Yastrebov (Go): 59.30s

My solution steps:

| Step | Description                     | Exec. time | Improvement | Baseline imp. | Commit                                                                                                  |
|-----:|---------------------------------|-----------:|------------:|--------------:|:--------------------------------------------------------------------------------------------------------|
| 1    | Naive approach                  | 286s       | -           | -             | [6bc5f94](https://github.com/domahidizoltan/1brc/blob/6bc5f9461f976b00b7b5dd02277c7196521d7c31/main.go) |
| 2    | Parallel measurement processors | 243s       | 1.177x      | 1.177x        | [b652f32](https://github.com/domahidizoltan/1brc/blob/b652f3292ec34aabdddaea0ba60a6bd29502ea2e/main.go) |
| 3    | Batch read file lines           | 167s       | 1.455x      | 1.712x        |                                                                                                         |

Comments for the steps:  
  1. Naive approach: Sequential file read and processing using 1 CPU core.  
  2. Parallel measurement processors: Sequential file read with multiple parallel measurement processors. The processors are sharded and one stations measurement will always be processed by the same processor. The results are merged and printed at the end. No concurrency optimizations were made at this point.  
  3. Batch read file lines: After doing a trace analysis (using file with 1M lines) we could see that `processMeasurements` function takes ~48% of the total time and more than half of it's time it is waiting for `chanrecv` (also sharding is suboptimal, less processing is done as expected). On the other hand `readFileLines` takes ~22% of the total time but it does a lot of IO reads with waits between them. The next step was optimizing only(!) the file read to read lines in batches and ignore splitting and converting them to strings. After this `readFileLines` took 1% of the total time (~20% waiting and half of it in syscall) and `processMeasurements` took 77% (~20% waiting). `processMeasurements` changed mostly the hashing, and now a station could land on multiple worker so at the end we need to merge them. With these changes the CPU cores are more busy and `readFileLines` does more syscalls in bigger chunks.
  Benchmark before and after for `readFileLines`:
```sh
❯ benchstat stats.old.txt stats.txt
goos: linux
goarch: amd64
pkg: github.com/domahidizoltan/1brc
cpu: Intel(R) Core(TM) i7-7500U CPU @ 2.70GHz
                │ stats.old.txt │             stats.txt              │
                │    sec/op     │   sec/op     vs base               │
ReadFileLines-4    3960.8µ ± 9%   130.9µ ± 2%  -96.70% (p=0.002 n=6)

                │ stats.old.txt  │              stats.txt               │
                │      B/op      │     B/op       vs base               │
ReadFileLines-4   15636.2Ki ± 0%   1008.1Ki ± 0%  -93.55% (p=0.002 n=6)

                │ stats.old.txt │           stats.txt           │
                │   allocs/op   │ allocs/op   vs base           │
ReadFileLines-4      5.000 ± 0%   5.000 ± 0%  ~ (p=1.000 n=6) ¹
¹ all samples are equal
```
  4. TODO

---

TODO:
- Compare implementation created by GitHub Copilot

- Extra task: implement measurements.txt file generator  
Reference execution time: create_measurements3.sh -> `Wrote 1,000,000,000 measurements in 267,667 ms`
