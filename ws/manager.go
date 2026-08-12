// Package ws, 文档稍后写
package ws

import (
	"errors"
	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
	"net/http"
	"sync"
	"time"
)

var (
	ErrCloseWs   = errors.New("close ws")
	pingInterval = 10 * time.Second
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type IClient interface {
	StartProcess()
}

type clientBuilder func(conn *websocket.Conn, manager *Manager) (IClient, error)

func NewManager() *Manager {
	return &Manager{
		clients: make([]IClient, 0, 30),
	}
}

// Manager 管理器
type Manager struct {
	clients []IClient
	lock    sync.Mutex
}

func (m *Manager) Serve(w http.ResponseWriter, r *http.Request, builder clientBuilder) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logx.Error(err)
		return
	}
	client, err := builder(conn, m)
	if err != nil {
		logx.Error(err)
		conn.Close()
		return
	}
	client.StartProcess()
	println("client start")
}

func (m *Manager) Register(w IClient) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.clients = append(m.clients, w)
	println("client registered")
}

func (m *Manager) Unregister(w IClient) {
	m.lock.Lock()
	defer m.lock.Unlock()
	for i := 0; i < len(m.clients); i++ {
		if w == m.clients[i] {
			m.clients = append(m.clients[:i], m.clients[i+1:]...)
			break
		}
	}
	println("client unregistered")
}
