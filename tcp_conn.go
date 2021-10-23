package network

import (
	"bufio"
	"net"
	"sync"
	"time"
)

type TCPConn struct {
	conn    net.Conn
	session Session
	writeCh chan []byte
	once    sync.Once
}

func (tcp *TCPConn) Close() {
	tcp.once.Do(func() {
		tcp.session.OnDisConnect()
		tcp.conn.Close()
	})
}

func (tcp *TCPConn) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		tcp.Close()
	}()
loop:
	for {
		select {
		case data := <-tcp.writeCh:
			tcp.conn.SetWriteDeadline(time.Now().Add(writeWait))
			_, err := tcp.conn.Write(data)
			if err != nil {
				log.Errorf("write err:%v", err)
				break loop
			}
		case <-ticker.C:
			tcp.conn.SetWriteDeadline(time.Now().Add(writeWait))
		}
	}
}

func (tcp *TCPConn) readPump() {
	scanner := bufio.NewScanner(tcp.conn)
	scanner.Buffer(make([]byte, 512), 512*2)
	scanner.Split(ScanSplit)
	for scanner.Scan() {
		tcp.session.OnMessage(scanner.Bytes())
	}
	if err := scanner.Err(); err != nil {
		log.Infof("tcp scan err:%v", err)
	}
}

func (tcp *TCPConn) ServerIO() {
	tcp.session.OnConnect(tcp)
	go tcp.readPump()
	tcp.writePump()
}

func NewTCPConn(conn net.Conn, sessionCreator func() Session) *TCPConn {
	tcpConn := &TCPConn{conn: conn, session: sessionCreator()}
	return tcpConn
}

type TCPServer struct {
	sessionCreator func() Session
}

func (server *TCPServer) Start(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Errorf("tcp accept is err:%v", err)
			continue
		}
		go func() {
			tcpConn := NewTCPConn(conn, server.sessionCreator)
			tcpConn.ServerIO()
		}()
	}
}

func NewTCPServer(sessionCreator func() Session) *TCPServer {
	return &TCPServer{sessionCreator: sessionCreator}
}
