package network

import (
	"encoding/binary"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
)

var log = logrus.WithFields(logrus.Fields{
	"network": "network",
})

const (
	writeWait  = 10 * time.Second
	pongWait   = 7 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

var (
	ErrFrameFormat = errors.New("frame is err")
)

type Conn interface {
	Close()
	Write([]byte)
}

type Session interface {
	OnConnect(Conn)
	OnMessage([]byte)
	OnDisConnect()
}

type Server interface {
	Start(addr string) error
}

type Frame struct {
	Body   []byte
	Extend uint64
	Len    uint64
}

func (frame *Frame) UnMarshal(data []byte) error {
	var fb = FrameByte(data)
	var dataLen int = len(fb)
	if dataLen <= 8+8 {
		return errors.New("bad data")
	}
	frame.Body = make([]byte, 0)

	frame.Len = fb.getFrameLenValue()
	frame.Extend = binary.BigEndian.Uint64(fb[dataLen-8-8 : dataLen-8])
	frame.Body = fb[:dataLen-8-2]
	return nil
}

type FrameByte []byte

func (fb FrameByte) getFrameLenValue() uint64 {
	return binary.BigEndian.Uint64(fb[len(fb)-8:])
}

func ScanSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	var fb = FrameByte(data)
	frameLen := fb.getFrameLenValue()
	if frameLen == 0 {
		return 0, nil, ErrFrameFormat
	}
	if len(data) < int(frameLen) {
		return 0, nil, nil
	}
	return int(frameLen), data[0:frameLen], nil
}
