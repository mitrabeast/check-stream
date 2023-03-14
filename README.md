# Check Streams

Install deps:
```
# apt install -y libavformat-dev libswscale-dev gcc pkg-config
``` 

Build:
```
$ go build -o check-stream *.go
```

Run:
```
$ ./check-stream rtsp://url/0
```
