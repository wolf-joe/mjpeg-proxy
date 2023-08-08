package main

import (
	"fmt"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func handleSnapshot(w http.ResponseWriter, req *http.Request, stream string) {
	reqID := RandStringRunes(10)
	log.Println(reqID, "new snapshot request:", req.RemoteAddr, req.URL.Path)

	manager, ok := GetFrameManager(stream)
	if !ok {
		log.Println(reqID, "get frame manager err:", stream)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	frame, err := manager.GetLatestFrame()
	if err != nil {
		log.Println(reqID, "get latest frame err:", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Length", strconv.Itoa(len(frame)))
	w.Write(frame)
	log.Println(reqID, "snapshot request done, sent", len(frame), "bytes")
}

func handleGet(w http.ResponseWriter, req *http.Request, stream string) {
	reqID := RandStringRunes(10)
	log.Println(reqID, "new get request:", req.RemoteAddr, req.URL.Path)

	cnt := 0
	manager, ok := GetFrameManager(stream)
	if !ok {
		log.Println(reqID, "get frame manager err:", stream)
		return
	}

	mimeWriter := multipart.NewWriter(w)
	contentType := fmt.Sprintf("multipart/x-mixed-replace;boundary=%s", mimeWriter.Boundary())
	w.Header().Add("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		frame, err := manager.GetLatestFrame()
		if err != nil {
			log.Println(reqID, "get latest frame err:", err)
			break
		}

		partHeader := make(textproto.MIMEHeader)
		partHeader.Add("Content-Type", "image/jpeg")
		partHeader.Add("Content-Length", strconv.Itoa(len(frame)))
		partWriter, err := mimeWriter.CreatePart(partHeader)
		if err != nil {
			log.Println(reqID, "create part err:", err)
			break
		}
		_, err = partWriter.Write(frame)
		if err != nil {
			log.Println(reqID, "write part err:", err)
			break
		}
		cnt++
		time.Sleep(66 * time.Millisecond) // 15fps
	}
	log.Println(reqID, "get request done, sent", cnt, "frames")
}

func handlePost(w http.ResponseWriter, req *http.Request, stream string) {
	reqID := RandStringRunes(10)
	log.Println(reqID, "new post request:", req.RemoteAddr, req.URL.Path)

	dispatcher := NewFrameManager(stream)
	reader := NewFrameReader(req.Body)
	cnt := 0
	for {
		frame, err := reader.ReadMJPEG()
		if err != nil {
			log.Println(reqID, "read mjpeg err:", err)
			break
		}
		dispatcher.AddFrame(frame)
		cnt++
	}
	log.Println(reqID, "post request done, received", cnt, "frames")
}

type Handler struct {
}

var (
	streamNameRegex = regexp.MustCompile(`^([a-zA-Z0-9_]+)$`)
)

func (h Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	retCode := func(code int) {
		log.Println("request err:", code, req.RemoteAddr, req.URL.Path)
		w.WriteHeader(code)
		w.Write([]byte(http.StatusText(code)))
	}

	stream := req.URL.Path[1:]
	if stream == "" {
		retCode(http.StatusNotFound)
		return
	}

	isSnapshot := false
	if strings.HasSuffix(stream, ".jpg") {
		stream = stream[:len(stream)-4]
		isSnapshot = true
	}

	if !streamNameRegex.MatchString(stream) {
		retCode(http.StatusBadRequest)
		return
	}

	switch req.Method {
	case "GET":
		if isSnapshot {
			handleSnapshot(w, req, stream)
		} else {
			handleGet(w, req, stream)
		}
	case "POST":
		handlePost(w, req, stream)
	default:
		retCode(http.StatusMethodNotAllowed)
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	hp := os.Getenv("HOST_PORT")
	if hp == "" {
		hp = ":8090"
	}
	log.Println("listening on", hp)
	if err := http.ListenAndServe(hp, Handler{}); err != nil {
		log.Fatal(err)
	}
}
