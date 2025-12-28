package utils

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type WSEventType string

const (
	WSEventOpen      WSEventType = "open"
	WSEventClose     WSEventType = "close"
	WSEventError     WSEventType = "error"
	WSEventReconnect WSEventType = "reconnect"
)

type EventHandler func(data any)

type WebSocketClient struct {
	url string

	connMu sync.RWMutex
	conn   *websocket.Conn

	writeMu sync.Mutex // 串行化写入（gorilla 强制要求）

	handlers   map[string][]EventHandler
	handlersMu sync.RWMutex

	DataChan  chan []byte
	onMessage func([]byte)

	quit chan struct{}

	reconnectEnabled atomic.Bool
	alive            atomic.Bool

	pingEvery time.Duration
}

func NewWebSocketClient(url string, pingEvery time.Duration) *WebSocketClient {
	c := &WebSocketClient{
		url:       url,
		handlers:  make(map[string][]EventHandler),
		DataChan:  make(chan []byte, 4096),
		quit:      make(chan struct{}),
		pingEvery: pingEvery,
	}
	c.reconnectEnabled.Store(true)
	return c
}

func (c *WebSocketClient) On(event WSEventType, handler EventHandler) {
	c.handlersMu.Lock()
	defer c.handlersMu.Unlock()
	c.handlers[string(event)] = append(c.handlers[string(event)], handler)
}

func (c *WebSocketClient) emit(event WSEventType, data any) {
	c.handlersMu.RLock()
	handlers := append([]EventHandler{}, c.handlers[string(event)]...)
	c.handlersMu.RUnlock()

	for _, h := range handlers {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					log.Println("[WS] handler panic:", r)
				}
			}()
			h(data)
		}(h)
	}
}

func (c *WebSocketClient) connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return err
	}

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	// 标记存活
	c.alive.Store(true)

	// read deadline 初始设置
	_ = conn.SetReadDeadline(time.Now().Add(2 * c.pingEvery))

	// pong 刷新 deadline
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(2 * c.pingEvery))
	})

	// close handler -> 触发事件但不强行返回 error
	conn.SetCloseHandler(func(code int, text string) error {
		c.emit(WSEventClose, code)
		return nil
	})

	c.emit(WSEventOpen, nil)
	return nil
}

// ping loop 绑定当前 conn 生命周期
func (c *WebSocketClient) startPing(conn *websocket.Conn) {
	ticker := time.NewTicker(c.pingEvery)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.writeMu.Lock()
				err := conn.WriteMessage(websocket.PingMessage, nil)
				c.writeMu.Unlock()

				if err != nil {
					return
				}
			case <-c.quit:
				return
			}
		}
	}()
}

func (c *WebSocketClient) readLoop() {
	for {
		c.connMu.RLock()
		conn := c.conn
		c.connMu.RUnlock()

		if conn == nil {
			return
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			c.alive.Store(false)
			c.emit(WSEventError, err)
			c.emit(WSEventClose, err)

			// 自动重连
			if c.reconnectEnabled.Load() {
				c.reconnectLoop()
				continue
			}
			return
		}

		select {
		case c.DataChan <- msg:
		case <-c.quit:
			return
		default:
			// 丢旧保新
			select {
			case <-c.DataChan:
			default:
			}
			c.DataChan <- msg
		}
	}
}

func (c *WebSocketClient) reconnectLoop() {
	delay := time.Second

	for {
		if !c.reconnectEnabled.Load() {
			return
		}

		log.Println("[WS] reconnecting...")
		c.emit(WSEventReconnect, nil)

		err := c.connect()
		if err == nil {
			log.Println("[WS] reconnected ✓")

			c.connMu.RLock()
			conn := c.conn
			c.connMu.RUnlock()

			c.startPing(conn)
			return
		}

		log.Println("[WS] reconnect failed:", err)
		time.Sleep(delay)

		if delay < 30*time.Second {
			delay *= 2
		}
	}
}

func (c *WebSocketClient) Start() {
	if err := c.connect(); err != nil {
		c.emit(WSEventError, err)
		if c.reconnectEnabled.Load() {
			c.reconnectLoop()
		}
	}

	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	c.startPing(conn)
	go c.readLoop()
	c.startMessageWorker()
}

func (c *WebSocketClient) Send(msg []byte) error {
	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return websocket.ErrCloseSent
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	return conn.WriteMessage(websocket.TextMessage, msg)
}

func (c *WebSocketClient) Close() {
	c.reconnectEnabled.Store(false)
	close(c.quit)

	c.connMu.Lock()
	if c.conn != nil {
		_ = c.conn.Close()
	}
	c.conn = nil
	c.connMu.Unlock()

	c.alive.Store(false)
}

func (c *WebSocketClient) OnMessage(handler func([]byte)) {
	c.onMessage = handler
}

func (c *WebSocketClient) startMessageWorker() {
	go func() {
		for {
			select {
			case msg := <-c.DataChan:
				if c.onMessage != nil {
					c.onMessage(msg)
				}
			case <-c.quit:
				return
			}
		}
	}()
}

func (c *WebSocketClient) IsAlive() bool {
	return c.alive.Load()
}
