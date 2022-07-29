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
	server  *TCPServer
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
	scanner.Split(tcp.server.splitFunc)
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

func (tcp *TCPConn) Write(b []byte) {
	tcp.writeCh <- b
}

func NewTCPConn(conn net.Conn, server *TCPServer) *TCPConn {
	tcpConn := &TCPConn{conn: conn, session: server.sessionCreator()}
	return tcpConn
}

type TCPServer struct {
	sessionCreator func() Session
	splitFunc      bufio.SplitFunc
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
			tcpConn := NewTCPConn(conn, server)
			tcpConn.ServerIO()
		}()
	}
}

func NewTCPServer(sessionCreator func() Session, splitFunc bufio.SplitFunc) *TCPServer {
	return &TCPServer{sessionCreator: sessionCreator, splitFunc: splitFunc}
}
