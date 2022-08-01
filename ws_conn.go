package network

import (
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type WsConn struct {
	conn    *websocket.Conn
	session Session
	once    sync.Once
	done    chan struct{}
}

func (ws *WsConn) ServerIO() {
	ws.session.OnConnect(ws)
	ws.readPump()
}

func (ws *WsConn) Close() {
	ws.once.Do(func() {
		ws.session.OnDisConnect()
		ws.conn.Close()
	})
}

func (ws *WsConn) readPump() {
	for {
		select {
		case <-ws.done:
			return
		default:
			_, message, err := ws.conn.ReadMessage()
			if err != nil {
				log.Printf("network read is err:%v", err)
				break
			}
			ws.session.OnMessage(message)
		}
	}
}

func (ws *WsConn) Write(b []byte) error {
	return ws.conn.WriteMessage(websocket.BinaryMessage, b)
}

func (ws *WsConn) LocalAddr() net.Addr {
	return ws.conn.UnderlyingConn().LocalAddr()
}

func (ws *WsConn) RemoteAddr() net.Addr {
	return ws.conn.UnderlyingConn().RemoteAddr()
}

func (ws *WsConn) GetSession() Session {
	return ws.session
}

func NewWsConn(conn *websocket.Conn, sessionCreator func() Session) *WsConn {
	wsConn := &WsConn{conn: conn, session: sessionCreator(), done: make(chan struct{})}
	return wsConn
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WsServer struct {
	sessionCreator func() Session
}

func (server *WsServer) Start(addr string) error {
	return http.ListenAndServe(addr, server)
}

func (server *WsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("network upgrade:%v", err)
		return
	}
	defer c.Close()
	wsConn := NewWsConn(c, server.sessionCreator)
	wsConn.ServerIO()
}

func NewWsServer(sessionCreator func() Session) Server {
	return &WsServer{sessionCreator: sessionCreator}
}
