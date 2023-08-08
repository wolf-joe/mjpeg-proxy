package main

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

const (
	bufSize      = 4096
	maxPool      = 10 * 1024 * 1024 // 10MB
	frameTimeout = 5 * time.Second
)

var (
	ErrMaxPool = errors.New("max pool size exceeded")
)

type FrameReader struct {
	reader     io.Reader
	pool       []byte
	start, end int
}

func NewFrameReader(r io.Reader) *FrameReader {
	buf := make([]byte, 0, bufSize)
	return &FrameReader{reader: r, pool: buf, start: -1, end: -1}
}

func (fr *FrameReader) ReadMJPEG() ([]byte, error) {
	findStart := func(buf []byte) int {
		for i := 0; i < len(buf)-1; i++ {
			if buf[i] == 0xFF && buf[i+1] == 0xD8 {
				return i
			}
		}
		return -1
	}
	findEnd := func(buf []byte) int {
		for i := 0; i < len(buf)-1; i++ {
			if buf[i] == 0xFF && buf[i+1] == 0xD9 {
				return i
			}
		}
		return -1
	}
	buf := make([]byte, 4096)
	for fr.start == -1 || fr.end == -1 {
		n, err := fr.reader.Read(buf)
		if err != nil {
			return nil, err
		}
		fr.pool = append(fr.pool, buf[:n]...)
		if len(fr.pool) > maxPool {
			return nil, ErrMaxPool
		}

		if fr.start == -1 {
			fr.start = findStart(fr.pool)
		}
		if fr.end == -1 {
			fr.end = findEnd(fr.pool)
		}
	}
	if fr.end < fr.start {
		// should never happen
		fr.pool = fr.pool[fr.end+2:]
		return nil, errors.New("invalid frame")
	}
	frame := fr.pool[fr.start : fr.end+2]
	fr.pool = fr.pool[fr.end+2:]
	fr.start = -1
	fr.end = -1
	return frame, nil
}

var (
	managerMap = new(sync.Map)
)

func NewFrameManager(name string) *FrameManager {
	res, _ := managerMap.LoadOrStore(name, &FrameManager{})
	return res.(*FrameManager)
}

func GetFrameManager(name string) (*FrameManager, bool) {
	res, ok := managerMap.Load(name)
	if !ok {
		return nil, false
	}
	return res.(*FrameManager), true
}

type FrameManager struct {
	latestFrame     atomic.Value
	latestTimestamp atomic.Value
}

func (fd *FrameManager) AddFrame(frame []byte) {
	fd.latestFrame.Store(frame)
	fd.latestTimestamp.Store(time.Now())
}

func (fd *FrameManager) GetLatestFrame() ([]byte, error) {
	frame := fd.latestFrame.Load()
	if frame == nil {
		return nil, errors.New("no frame")
	}
	timestamp := fd.latestTimestamp.Load().(time.Time)
	if time.Since(timestamp) > frameTimeout {
		return nil, errors.New("frame timeout")
	}
	return frame.([]byte), nil
}
