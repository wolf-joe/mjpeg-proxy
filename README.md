# mjpeg-proxy

- receive one or many mjpeg streaming
- send mjpeg streaming or snapshot

## usage

```shell
go install github.com/wolf-joe/mjpeg-proxy@latest
# set host & port, then run
HOST_PORT=":8091" $GOBIN/mjpeg-proxy
```

## arch

```mermaid
flowchart LR
    writer_1 -- push mjpeg to /stream1 --> p((mjpeg-proxy))
    writer_2 -- push mjpeg to /stream2 --> p

    p -- pull stream from /stream1 --> reader_1
    p -- pull snapshot from /stream1.jpg --> reader_2
```

## usecase

```mermaid
flowchart LR
    camera>camera] -- pull rtsp stream --> ffmpeg
    ffmpeg -- write to file --> disk[(disk)]
    ffmpeg -- convert to mjpeg & push to /camera --> p((mjpeg-proxy))
    p -- pull /camera & show --> octoprint
    p -- pull /camera.jpg & store --> octoprint
```