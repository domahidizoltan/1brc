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
```

Execution time for 1M row
```sh
$ time ./calculate_average_baseline.sh  
1,39s user 0,30s system 149% cpu 1,132 total

$ time ./calculate_average_thomaswue.sh > /dev/null  
1,16s user 0,22s system 200% cpu 0,689 total
```
---

My solution steps:

| Step | Description          | Exec. time | Improvement | Baseline imp. | Commit |
|-----:|----------------------|-----------:|------------:|--------------:|:-------|
| 1    | Naive implementation | ?s         |             |               |        |

For reference:
- 1BRC Baseline: 392.96s
- Thomas Wuerthinger: 62.58s
---

TODO:
- Compare implementation created by GitHub Copilot

- Extra task: implement measurements.txt file generator  
Reference execution time: create_measurements3.sh -> `Wrote 1,000,000,000 measurements in 267,667 ms`
