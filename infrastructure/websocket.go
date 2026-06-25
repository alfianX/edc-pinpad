package infrastructure

import (
	"edc-pinpad/model"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

type WSManager struct {
	db          *gorm.DB
	connections map[string]*websocket.Conn
	mu          sync.RWMutex
	logger      *slog.Logger
	Upgrader    websocket.Upgrader
}

func NewWSManager(l *slog.Logger, db *gorm.DB) *WSManager {
	return &WSManager{
		db:          db,
		connections: make(map[string]*websocket.Conn),
		logger:      l,
		Upgrader:    websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
	}
}

func (m *WSManager) RegisterConn(tid string, conn *websocket.Conn) {
	m.mu.Lock()
	m.connections[tid] = conn
	m.db.Create(&model.WsConnection{TerminalId: tid, CreatedAt: time.Now()})
	m.mu.Unlock()
}

func (m *WSManager) UnregisterConn(tid string) {
	m.mu.Lock()
	if conn, ok := m.connections[tid]; ok {
		conn.Close()
		delete(m.connections, tid)
		m.db.Unscoped().Where("terminal_id = ?", tid).Delete(&model.WsConnection{})
	}
	m.mu.Unlock()
}

func (m *WSManager) SendJSON(tid string, data interface{}) error {
	m.mu.RLock()
	conn, ok := m.connections[tid]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("device offline")
	}
	return conn.WriteJSON(data)
}
