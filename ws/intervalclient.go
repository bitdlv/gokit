package ws

import (
	"context"
	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
	"time"
)

type GoroutineHandler func(ctx context.Context, in <-chan []byte, out chan<- []byte)

func NewGoroutineClient(conn *websocket.Conn, manager *Manager, goroutine GoroutineHandler) *GoroutineClient {
	return &GoroutineClient{
		conn:      conn,
		manager:   manager,
		goroutine: goroutine,
		out:       make(chan []byte),
		in:        make(chan []byte),
	}
}

type GoroutineClient struct {
	conn      *websocket.Conn
	manager   *Manager
	goroutine GoroutineHandler
	ctx       context.Context
	cancel    func()
	in        chan []byte
	out       chan []byte
}

func (w *GoroutineClient) StartProcess() {
	w.ctx, w.cancel = context.WithCancel(context.Background())

	go w.startRead()
	go w.startWrite()
}

func (w *GoroutineClient) Close() {
	w.conn.Close()
	w.cancel()
	close(w.in)
	close(w.out)
	w.manager.Unregister(w)
}

func (w *GoroutineClient) startRead() {
	w.manager.Register(w)
	go w.goroutine(w.ctx, w.in, w.out)
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			_, content, err := w.conn.ReadMessage()
			if err != nil {
				return
			}
			println("client read:", string(content))

			w.in <- content
		}
	}
}

func (w *GoroutineClient) startWrite() {
	defer w.Close()

	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case msg, ok := <-w.out:
			if !ok {
				return
			}
			err := w.conn.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				logx.Error(err)
				return
			}
		case <-ticker.C:
			err := w.conn.WriteControl(websocket.PingMessage, []byte{}, time.Time{})
			if err != nil {
				logx.Error(err)
				return
			}
		}
	}
}
