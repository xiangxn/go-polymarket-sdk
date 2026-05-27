package utils

import (
	"context"
	"errors"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var ErrMsgOverflow = errors.New("ws message channel overflow")

// WSHandler websocket事件处理器
// 注意：OnMessage 现在可能被多个 worker 并发调用。
type WSHandler interface {
	OnOpen()
	OnReconnect()
	OnError(err error)
	OnClose()
	// 如果 handler 内部有共享状态，请自行保证线程安全。
	OnMessage(msg []byte)
}

type WSClient interface {
	Run(ctx context.Context) error
	Send(msg []byte) error
	Messages() <-chan []byte
	IsAlive() bool
	Close() error
	Reset() error
}

type WSConfig struct {
	URL string

	HandshakeTimeout time.Duration

	// websocket消息缓冲区
	MsgBufferSize int

	// worker数量
	WorkerNum int

	// websocket ping frame间隔
	PingInterval time.Duration

	// pong超时时间
	PongWait time.Duration

	// 是否启用文本心跳（Polymarket需要）
	TextHeartbeat bool

	// 文本心跳内容
	TextHeartbeatMsg []byte

	Reconnect      bool
	ReconnectDelay time.Duration
	MaxReconnect   int
}

// overflow时是否丢弃旧消息
type wsClient struct {
	cfg     WSConfig
	handler WSHandler

	conn   *websocket.Conn
	dialer *websocket.Dialer

	// 写队列
	sendCh chan []byte

	// 原始消息队列
	msgCh chan []byte

	ctrlCh chan wsControl

	alive atomic.Bool

	mu      sync.Mutex
	writeMu sync.Mutex

	closed atomic.Bool
}

type wsControl int

const (
	ctrlReconnect wsControl = iota
	ctrlClose
)

func NewWSClient(cfg WSConfig, handler WSHandler) WSClient {
	if cfg.MsgBufferSize == 0 {
		cfg.MsgBufferSize = 8192
	}

	if cfg.WorkerNum <= 0 {
		cfg.WorkerNum = runtime.NumCPU()
	}

	if cfg.PingInterval == 0 {
		cfg.PingInterval = 10 * time.Second
	}

	if cfg.PongWait == 0 {
		cfg.PongWait = 30 * time.Second
	}

	if cfg.HandshakeTimeout == 0 {
		cfg.HandshakeTimeout = 20 * time.Second
	}

	if cfg.ReconnectDelay == 0 {
		cfg.ReconnectDelay = 5 * time.Second
	}

	if cfg.Reconnect && cfg.MaxReconnect == 0 {
		cfg.MaxReconnect = 3
	}

	// 默认开启Polymarket兼容文本心跳
	if cfg.TextHeartbeatMsg == nil {
		cfg.TextHeartbeatMsg = []byte("PING")
	}

	return &wsClient{
		cfg:     cfg,
		handler: handler,

		msgCh:  make(chan []byte, cfg.MsgBufferSize),
		sendCh: make(chan []byte, cfg.MsgBufferSize),
		ctrlCh: make(chan wsControl, 1),
	}
}

func (c *wsClient) Run(ctx context.Context) error {
	retry := 0
	first := true

	for {
		select {
		case <-ctx.Done():
			c.callOnClose()
			return ctx.Err()
		default:
		}

		if err := c.connect(); err != nil {
			if !c.cfg.Reconnect || retry >= c.cfg.MaxReconnect {
				return err
			}

			retry++

			log.Printf("[WSClient] reconnect attempt=%d", retry)

			if !SleepWithCtx(ctx, c.cfg.ReconnectDelay) {
				c.callOnClose()
				return ctx.Err()
			}

			continue
		}

		func() {
			myCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			c.alive.Store(true)
			c.closed.Store(false)

			retry = 0

			if first {
				c.callOnOpen()
				first = false
			} else {
				c.callOnReconnect()
			}

			errCh := make(chan error, 1)

			go c.readLoop(myCtx, errCh)
			go c.writeLoop(myCtx, errCh)
			go c.pingLoop(myCtx, errCh)

			// worker pool
			for i := 0; i < c.cfg.WorkerNum; i++ {
				go c.messageLoop(myCtx)
			}

			select {
			case <-ctx.Done():
				_ = c.Close()
				c.callOnClose()
				return

			case ctrl := <-c.ctrlCh:
				switch ctrl {
				case ctrlReconnect:
					c.callOnError(errors.New("manual reconnect"))
					_ = c.Close()
					return

				case ctrlClose:
					_ = c.Close()
					c.callOnClose()
					return
				}

			case err := <-errCh:
				c.callOnError(err)
				_ = c.Close()

				if !c.cfg.Reconnect {
					c.callOnClose()
					return
				}

				if !SleepWithCtx(ctx, c.cfg.ReconnectDelay) {
					c.callOnClose()
					return
				}
			}
		}()
	}
}

func (c *wsClient) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dialer := c.dialer
	if dialer == nil {
		dialer = websocket.DefaultDialer
	}

	dialer.HandshakeTimeout = c.cfg.HandshakeTimeout

	conn, _, err := dialer.Dial(c.cfg.URL, nil)
	if err != nil {
		return err
	}

	// read限制
	conn.SetReadLimit(8 << 20)

	// 初始deadline
	_ = conn.SetReadDeadline(time.Now().Add(c.cfg.PongWait))

	// websocket pong handler
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(c.cfg.PongWait))
	})

	c.conn = conn

	return nil
}

func (c *wsClient) readLoop(ctx context.Context, errCh chan<- error) {
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			select {
			case errCh <- err:
			default:
			}
			return
		}

		select {
		case <-ctx.Done():
			return

		case c.msgCh <- msg:

		default:
			// 高频市场数据优化：
			// overflow时丢弃旧消息，而不是重连。
			// 最新book比旧book更重要。

			select {
			case <-c.msgCh:
			default:
			}

			select {
			case c.msgCh <- msg:
			default:
			}

			log.Printf("[WSClient] msgCh overflow, dropped oldest message")

		}
	}
}

func (c *wsClient) writeLoop(ctx context.Context, errCh chan<- error) {
	for {
		select {
		case <-ctx.Done():
			return

		case msg, ok := <-c.sendCh:
			if !ok {
				return
			}

			if err := c.writeMessage(websocket.TextMessage, msg); err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
		}
	}
}

func (c *wsClient) pingLoop(ctx context.Context, errCh chan<- error) {
	ticker := time.NewTicker(c.cfg.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:

			// websocket ping frame
			if err := c.writeMessage(websocket.PingMessage, nil); err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}

			// Polymarket兼容文本PING
			if c.cfg.TextHeartbeat {
				if err := c.writeMessage(websocket.TextMessage, c.cfg.TextHeartbeatMsg); err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}
			}
		}
	}
}

func (c *wsClient) messageLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-c.msgCh:
			c.callOnMessage(msg)
		}
	}
}

func (c *wsClient) writeMessage(mt int, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if c.conn == nil {
		return errors.New("conn is nil")
	}

	_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

	return c.conn.WriteMessage(mt, data)
}

func (c *wsClient) Send(msg []byte) error {
	if !c.IsAlive() {
		return errors.New("ws not alive")
	}

	select {
	case c.sendCh <- msg:
		return nil

	default:
		return errors.New("send buffer full")
	}
}

func (c *wsClient) Messages() <-chan []byte {
	return c.msgCh
}

func (c *wsClient) IsAlive() bool {
	return c.alive.Load()
}

func (c *wsClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed.Load() {
		return nil
	}

	c.closed.Store(true)
	c.alive.Store(false)

	if c.conn != nil {

		// 尝试正常close handshake
		_ = c.writeMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)

		err := c.conn.Close()
		c.conn = nil
		return err
	}

	return nil
}

func (c *wsClient) Reset() error {
	select {
	case c.ctrlCh <- ctrlReconnect:
		return nil
	default:
		return errors.New("ws control busy")
	}
}

/* ---------- handler safe call ---------- */

func (c *wsClient) callOnOpen() {
	if c.handler != nil {
		SafeCall(c.handler.OnOpen)
	}
}

func (c *wsClient) callOnReconnect() {
	if c.handler != nil {
		SafeCall(c.handler.OnReconnect)
	}
}

func (c *wsClient) callOnError(err error) {
	if c.handler != nil {
		SafeCall(func() {
			c.handler.OnError(err)
		})
	}
}

func (c *wsClient) callOnClose() {
	if c.handler != nil {
		SafeCall(c.handler.OnClose)
	}
}

func (c *wsClient) callOnMessage(msg []byte) {
	if c.handler != nil {
		SafeCall(func() {
			c.handler.OnMessage(msg)
		})
	}
}
