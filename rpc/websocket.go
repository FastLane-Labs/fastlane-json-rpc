package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/FastLane-Labs/fastlane-json-rpc/log"
	rpcContext "github.com/FastLane-Labs/fastlane-json-rpc/rpc/context"
	"github.com/FastLane-Labs/fastlane-json-rpc/rpc/jsonrpc"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type ConnContextKeyType string

const (
	pongWait   = 60 * time.Second
	pingPeriod = (60 * time.Second * 9) / 10
	writeWait  = 2 * time.Second

	ConnContextKey = ConnContextKeyType("ws-conn")
)

type Conn struct {
	*websocket.Conn
	IP       string
	sendChan chan []byte
}

func NewConn(conn *websocket.Conn) *Conn {
	return &Conn{
		Conn:     conn,
		IP:       conn.RemoteAddr().String(),
		sendChan: make(chan []byte, 256),
	}
}

func (c *Conn) send(msg *jsonrpc.JsonRpcResponse) {
	c.sendChan <- msg.Marshal()
}

func (c *Conn) SendRaw(data []byte) {
	c.sendChan <- data
}

func (s *Server) websocketHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	s.wg.Add(1)
	defer s.wg.Done()

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(ctx, "failed upgrading connection", "err", err)
		return
	}

	conn := NewConn(c)
	doneChan := make(chan struct{})

	go s.websocketWriteLoop(conn, doneChan)
	go s.websocketReadLoop(conn, doneChan)

	if s.metrics.enabled {
		s.metrics.WebsocketConnections.Inc()
	}
}

func (s *Server) websocketReadLoop(conn *Conn, doneChan chan struct{}) {
	defer func() {
		conn.Close()
		close(doneChan)

		if s.metrics.enabled {
			s.metrics.WebsocketConnections.Dec()
		}
	}()

	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure,
				websocket.CloseAbnormalClosure,
				websocket.CloseNoStatusReceived,
			) {
				log.Error(context.Background(), "websocketReadLoop: unexpected close error", "ip", conn.IP, "err", err)
			}
			return
		}

		// Handle the request in a separate goroutine
		go func() {
			ctx := rpcContext.NewContextWithTraceId(context.Background(), uuid.New().String())
			ctx = context.WithValue(ctx, ConnContextKey, conn)

			defer func() {
				if r := recover(); r != nil {
					conn.send(jsonrpc.NewJsonRpcErrorResponse(jsonrpc.InternalError, "internal error", nil, nil))
					log.Error(ctx, "websocket server execution error", "error", r, "stack", string(debug.Stack()))
				}
			}()

			s.wg.Add(1)
			defer s.wg.Done()

			if s.metrics.enabled {
				s.metrics.RequestWebsocket.Inc()
			}

			var request jsonrpc.JsonRpcRequest
			if err := json.Unmarshal(message, &request); err != nil {
				conn.send(jsonrpc.NewJsonRpcErrorResponse(jsonrpc.ParseError, "invalid request", err.Error(), nil))
				return
			}

			conn.send(s.handleJsonRpcRequest(ctx, &request))
		}()
	}
}

func (s *Server) websocketWriteLoop(conn *Conn, doneChan <-chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-s.shutdownChan:
			closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Server closing connection")
			if err := conn.WriteMessage(websocket.CloseMessage, closeMsg); err != nil {
				log.Error(context.Background(), "websocketWriteLoop: failed to write close message", "ip", conn.IP, "err", err)
			}
			return

		case <-doneChan:
			return

		case <-ticker.C:
			deadline := time.Now().Add(writeWait)
			if err := conn.WriteControl(websocket.PingMessage, []byte{}, deadline); err != nil {
				log.Error(context.Background(), "websocketWriteLoop: failed to write ping message", "ip", conn.IP, "err", err)
				return
			}

		case msg := <-conn.sendChan:
			deadline := time.Now().Add(writeWait)
			conn.SetWriteDeadline(deadline)

			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Error(context.Background(), "websocketWriteLoop: failed to write message", "ip", conn.IP, "err", err)
				return
			}
		}
	}
}
