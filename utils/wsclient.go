package utils

import (
	"fmt"
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
	url  string
	conn *websocket.Conn

	// 事件处理（open/close/error/reconnect）
	handlers map[string][]EventHandler
	mu       sync.RWMutex

	// 数据通道（高频消息）
	DataChan  chan []byte
	onMessage func([]byte)

	// 重连控制
	quit      chan struct{}
	reconnect int32 // atomic
	pingEvery time.Duration
}

/*
NewWebSocketClient 创建客户端
@param url 服务器地址
@param pingEvery 心跳间隔
*/
func NewWebSocketClient(url string, pingEvery time.Duration) *WebSocketClient {
	return &WebSocketClient{
		url:       url,
		handlers:  make(map[string][]EventHandler),
		DataChan:  make(chan []byte, 4096),
		quit:      make(chan struct{}),
		reconnect: 1,
		pingEvery: pingEvery,
	}
}

/*
On 注册事件处理函数
@param event 事件名称
@param handler 事件处理函数
*/
func (c *WebSocketClient) On(event WSEventType, handler EventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[string(event)] = append(c.handlers[string(event)], handler)
}

/*
emit 触发事件
@param event 事件名称
@param data 事件数据
*/
func (c *WebSocketClient) emit(event WSEventType, data any) {
	c.mu.RLock()
	handlers := c.handlers[string(event)]
	c.mu.RUnlock()

	for _, h := range handlers {
		go h(data)
	}
}

/*
connect 连接服务器
*/
func (c *WebSocketClient) connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return err
	}
	c.conn = conn
	c.conn.SetCloseHandler(func(code int, text string) error {
		c.emit(WSEventClose, code)
		return fmt.Errorf("ws close %s", text)
	})
	c.emit(WSEventOpen, nil)
	return nil
}

/*
startPing 启动心跳
*/
func (c *WebSocketClient) startPing() {
	ticker := time.NewTicker(c.pingEvery)
	go func() {
		for {
			select {
			case <-ticker.C:
				if c.conn != nil {
					c.conn.WriteMessage(websocket.PingMessage, []byte("ping"))
				}
			case <-c.quit:
				ticker.Stop()
				return
			}
		}
	}()
}

/*
readLoop 读取消息循环
*/
func (c *WebSocketClient) readLoop() {
	for {
		msgType, msg, err := c.conn.ReadMessage()
		if err != nil {
			c.emit(WSEventError, err)
			c.emit(WSEventClose, err)
			if atomic.LoadInt32(&c.reconnect) == 1 {
				c.reconnectLoop()
				continue
			} else {
				return
			}
		}
		if msgType == websocket.CloseMessage {
			c.emit(WSEventClose, websocket.CloseMessage)
			if atomic.LoadInt32(&c.reconnect) == 1 {
				c.reconnectLoop()
				continue
			} else {
				return
			}
		}

		// 高频数据 → 数据通道
		select {
		case c.DataChan <- msg:
		case <-c.quit:
			return
		default:
			// channel 满了 → 丢掉 1 条旧消息
			select {
			case <-c.DataChan:
				// 忽略旧消息
			default:
			}
			// 再写入最新消息
			c.DataChan <- msg
		}
	}
}

/*
reconnectLoop 重连循环
*/
func (c *WebSocketClient) reconnectLoop() {
	delay := time.Second

	for {
		log.Println("[WS] Reconnecting...")
		// 重连事件
		c.emit(WSEventReconnect, nil)
		err := c.connect()
		if err == nil {
			log.Println("[WS] Reconnected ✓")
			break
		}

		log.Println("[WS] retry failed:", err)
		time.Sleep(delay)

		if delay < 30*time.Second {
			delay *= 2
		}
	}
}

/*
Start 启动客户端
*/
func (c *WebSocketClient) Start() {
	if err := c.connect(); err != nil {
		c.emit(WSEventError, err)
		if atomic.LoadInt32(&c.reconnect) == 1 {
			c.reconnectLoop()
		}
	}

	go c.readLoop()
	c.startPing()
	c.startMessageWorker()
}

/*
Send 发送消息
@param msg 消息内容
*/
func (c *WebSocketClient) Send(msg []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return websocket.ErrCloseSent
	}
	return c.conn.WriteMessage(websocket.TextMessage, msg)
}

/*
Close 关闭客户端
*/
func (c *WebSocketClient) Close() {
	atomic.StoreInt32(&c.reconnect, 0)
	close(c.quit)

	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
	}
	c.mu.Unlock()
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
