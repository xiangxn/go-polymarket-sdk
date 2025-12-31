package utils

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type WSClient interface {
	Run(ctx context.Context) error
	Send(msg []byte) error
	Messages() <-chan []byte
	IsAlive() bool
	Close() error
}

type WSConfig struct {
	URL string

	HandshakeTimeout time.Duration

	MsgBufferSize int

	PingInterval time.Duration
	PongWait     time.Duration

	Reconnect      bool
	ReconnectDelay time.Duration
	MaxReconnect   int
}

type wsClient struct {
	cfg WSConfig

	conn   *websocket.Conn
	dialer *websocket.Dialer

	sendCh chan []byte
	msgCh  chan []byte

	alive atomic.Bool
	mu    sync.Mutex
}

func NewWSClient(cfg WSConfig) WSClient {
	if cfg.MsgBufferSize == 0 {
		cfg.MsgBufferSize = 1024
	}
	if cfg.PingInterval == 0 {
		cfg.PingInterval = 10 * time.Second
	}
	if cfg.PongWait == 0 {
		cfg.PongWait = 30 * time.Second
	}
	if cfg.ReconnectDelay == 0 {
		cfg.ReconnectDelay = 5 * time.Second
	}

	return &wsClient{
		cfg:   cfg,
		msgCh: make(chan []byte, cfg.MsgBufferSize),
	}
}

func (c *wsClient) Run(ctx context.Context) error {
	retry := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := c.connect(); err != nil {
			if !c.cfg.Reconnect || retry >= c.cfg.MaxReconnect {
				return err
			}
			retry++
			if !SleepWithCtx(ctx, c.cfg.ReconnectDelay) {
				return ctx.Err()
			}
			continue
		}

		c.alive.Store(true)
		retry = 0

		errCh := make(chan error, 1)

		go c.readLoop(errCh)
		go c.writeLoop(ctx, errCh)
		go c.pingLoop(ctx)

		select {
		case <-ctx.Done():
			c.Close()
			return ctx.Err()

		case err := <-errCh:
			c.Close()
			if !c.cfg.Reconnect {
				return err
			}
			if !SleepWithCtx(ctx, c.cfg.ReconnectDelay) {
				return ctx.Err()
			}
		}
	}
}

func (c *wsClient) readLoop(errCh chan<- error) {
	c.conn.SetReadLimit(1 << 20)
	_ = c.conn.SetReadDeadline(time.Now().Add(c.cfg.PongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(c.cfg.PongWait))
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			select {
			case errCh <- err:
			default:
			}
			return
		}

		// 不阻塞 IO
		select {
		case c.msgCh <- msg:
		default:
			// backpressure：丢弃
		}
	}
}

func (c *wsClient) writeLoop(ctx context.Context, errCh chan<- error) {
	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-c.sendCh:
			c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
		}
	}
}

func (c *wsClient) pingLoop(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			_ = c.conn.WriteMessage(websocket.PingMessage, nil)
		}
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

	c.conn = conn
	c.sendCh = make(chan []byte, c.cfg.MsgBufferSize)

	return nil
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

	c.alive.Store(false)
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
