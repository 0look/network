package network

import (
	"bufio"
	"log"
	"net"
	"sync"
	"time"
)

type TCPConn struct {
	conn    net.Conn
	session Session
	once    sync.Once
	server  *TCPServer
	done    chan struct{}
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
	for {
		select {
		case <-ticker.C:
			tcp.conn.SetWriteDeadline(time.Now().Add(writeWait))
		case <-tcp.done:
			return
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
		log.Printf("network tcp scan err:%v", err)
	}
}

func (tcp *TCPConn) ServerIO() {
	tcp.session.OnConnect(tcp)
	go tcp.readPump()
	tcp.writePump()
}

func (tcp *TCPConn) Write(b []byte) error {
	tcp.conn.SetWriteDeadline(time.Now().Add(writeWait))
	_, err := tcp.conn.Write(b)
	if err != nil {
		log.Printf("network write err:%v", err)
	}
	return err
}

func (tcp *TCPConn) LocalAddr() net.Addr {
	return tcp.LocalAddr()
}

func (tcp *TCPConn) RemoteAddr() net.Addr {
	return tcp.RemoteAddr()
}

func (tcp *TCPConn) GetSession() Session {
	return tcp.session
}

func NewTCPConn(conn net.Conn, server *TCPServer) *TCPConn {
	tcpConn := &TCPConn{conn: conn, session: server.sessionCreator(), server: server, done: make(chan struct{})}
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
			log.Printf("network tcp accept is err:%v", err)
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
