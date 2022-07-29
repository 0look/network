package network

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type WsConn struct {
	conn    *websocket.Conn
	session Session
	writeCh chan []byte
	once    sync.Once
}

func (ws *WsConn) ServerIO() {
	ws.session.OnConnect(ws)
	go ws.readPump()
	ws.writePump()
}

func (ws *WsConn) Close() {
	ws.once.Do(func() {
		ws.session.OnDisConnect()
		ws.conn.Close()
	})
}

func (ws *WsConn) readPump() {
	for {
		mt, message, err := ws.conn.ReadMessage()
		if err != nil {
			log.Println("read is err:%v", err)
			break
		}
		if mt == websocket.BinaryMessage {
			ws.session.OnMessage(message)
		}
	}
}

func (ws *WsConn) writePump() {
	for {
		select {
		case b := <-ws.writeCh:
			ws.conn.WriteMessage(websocket.BinaryMessage, b)
		}
	}
}

func NewWsConn(conn *websocket.Conn, sessionCreator func() Session) *WsConn {
	wsConn := &WsConn{conn: conn, session: sessionCreator(), writeCh: make(chan []byte)}
	return wsConn
}

var upgrader = websocket.Upgrader{}

type WsServer struct {
	sessionCreator func() Session
}

func (server *WsServer) Start(addr string) error {
	return http.ListenAndServe(addr, server)
}

func (server *WsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer c.Close()
	wsConn := NewWsConn(c, server.sessionCreator)
	wsConn.ServerIO()
}

func NewWsServer(sessionCreator func() Session) Server {
	return &WsServer{sessionCreator: sessionCreator}
}
