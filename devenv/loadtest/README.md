# Image rendering load test

Runs load tests and checks using [k6](https://k6.io/).

## Prerequisites

Docker

## Run

Run load test for 15 minutes using 2 virtual users and targeting http://localhost:3000.

```bash
$ ./run.sh
```

Run load test for custom duration:

```bash
$ ./run.sh -d 10s
```

Run only 1 iteration of the load test (useful for testing):

```bash
$ ./run.sh -i 1

Run load test for custom target url:

```bash
$ ./run.sh -u http://grafana.loc
```

Run load test for 10 virtual users:

```bash
$ ./run.sh -v 10
```

Example output:

```bash
> ./run.sh -d 15m -v 10 -u http://grafana.loc

          /\      |‾‾|  /‾‾/  /‾/
     /\  /  \     |  |_/  /  / /
    /  \/    \    |      |  /  ‾‾\
   /          \   |  |‾\  \ | (_) |
  / __________ \  |__|  \__\ \___/ .io

  execution: local
     output: -
     script: src/render_test.js

    duration: 15m0s, iterations: -
         vus: 10,    max: 10

    done [==========================================================] 15m0s / 15m0s

    █ render test

      █ user authenticates thru ui with username and password

        ✓ response status is 200

      █ render graph panel

        ✓ response status is 200

    checks.....................: 100.00% ✓ 726  ✗ 0
    data_received..............: 94 MB   104 kB/s
    data_sent..................: 170 kB  188 B/s
    group_duration.............: avg=12.32s  min=78.5ms  med=10.17s max=30.26s  p(90)=22.21s  p(95)=24.46s
    http_req_blocked...........: avg=34.73µs min=2.16µs  med=5.45µs max=4.04ms  p(90)=8.78µs  p(95)=15.34µs
    http_req_connecting........: avg=9.99µs  min=0s      med=0s     max=1.26ms  p(90)=0s      p(95)=0s
    http_req_duration..........: avg=12.32s  min=76.75ms med=10.17s max=30.26s  p(90)=22.21s  p(95)=24.46s
    http_req_receiving.........: avg=2.38ms  min=44.7µs  med=2.35ms max=16.37ms p(90)=4.64ms  p(95)=5.98ms
    http_req_sending...........: avg=51.41µs min=10.48µs med=21.5µs max=4.29ms  p(90)=38.26µs p(95)=72.63µs
    http_req_tls_handshaking...: avg=0s      min=0s      med=0s     max=0s      p(90)=0s      p(95)=0s
    http_req_waiting...........: avg=12.32s  min=76.59ms med=10.17s max=30.24s  p(90)=22.21s  p(95)=24.46s
    http_reqs..................: 726     0.806667/s
    iteration_duration.........: avg=12.31s  min=5.28µs  med=10.17s max=30.26s  p(90)=22.21s  p(95)=24.46s
    iterations.................: 726     0.806667/s
    vus........................: 10      min=10 max=10
    vus_max....................: 10      min=10 max=10
```
