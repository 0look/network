package network

import (
	"errors"
	"net"
	"time"
)

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
	Write([]byte) error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	GetSession() Session
}

type Session interface {
	OnConnect(Conn)
	OnMessage([]byte)
	OnDisConnect()
}

type Server interface {
	Start(addr string) error
}
