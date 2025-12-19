package polymarket

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
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

	// 订阅列表（重连恢复用）
	subsMu        sync.RWMutex
	subscriptions [][]byte
}

/*
NewWebSocketClient 创建客户端
@param url 服务器地址
@param pingEvery 心跳间隔
*/
func NewWebSocketClient(url string, pingEvery time.Duration) *WebSocketClient {
	return &WebSocketClient{
		url:           url,
		handlers:      make(map[string][]EventHandler),
		DataChan:      make(chan []byte, 4096),
		quit:          make(chan struct{}),
		reconnect:     1,
		pingEvery:     pingEvery,
		subscriptions: make([][]byte, 0),
	}
}

/*
On 注册事件处理函数
@param event 事件名称
@param handler 事件处理函数
*/
func (c *WebSocketClient) On(event string, handler EventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[event] = append(c.handlers[event], handler)
}

/*
emit 触发事件
@param event 事件名称
@param data 事件数据
*/
func (c *WebSocketClient) emit(event string, data any) {
	c.mu.RLock()
	handlers := c.handlers[event]
	c.mu.RUnlock()

	for _, h := range handlers {
		go h(data)
	}
}

/*
AddSubscription 添加订阅
@param msg 订阅消息
*/
func (c *WebSocketClient) AddSubscription(msg []byte) {
	c.subsMu.Lock()
	defer c.subsMu.Unlock()
	c.subscriptions = append(c.subscriptions, msg)
}

func (c *WebSocketClient) restoreSubscriptions() {
	c.subsMu.RLock()
	defer c.subsMu.RUnlock()
	for _, s := range c.subscriptions {
		c.Send(s)
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
	c.emit("open", nil)
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
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			c.emit("error", err)
			c.emit("close", nil)

			if atomic.LoadInt32(&c.reconnect) == 1 {
				c.reconnectLoop()
			}
			return
		}

		// 高频数据 → 数据通道
		select {
		case c.DataChan <- msg:
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
		err := c.connect()
		if err == nil {
			log.Println("[WS] Reconnected ✓")

			// 重连事件
			c.emit("reconnect", nil)

			// 恢复订阅
			c.restoreSubscriptions()

			go c.readLoop()
			return
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
func (c *WebSocketClient) Start() error {
	if err := c.connect(); err != nil {
		return err
	}

	go c.readLoop()
	c.startPing()
	c.startMessageWorker()
	return nil
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
Subscribe 订阅
@param msg 订阅消息
*/
func (c *WebSocketClient) Subscribe(msg []byte) error {
	c.AddSubscription(msg)
	return c.Send(msg)
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
		for msg := range c.DataChan {
			if c.onMessage != nil {
				c.onMessage(msg)
			}
		}
	}()
}
