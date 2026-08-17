package polymarket

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestGetRateLimitRetry 429 时按 Retry-After 头等待并重试,最终成功(无需网络)
func TestGetRateLimitRetry(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if count.Add(1) <= 2 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"openPrice":123.45,"closePrice":120.1}`))
	}))
	defer server.Close()

	client := NewClient(DefaultConfig())
	start := time.Now()
	result, err := client.Get(server.URL, nil, nil)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if result.Get("openPrice").Float() != 123.45 {
		t.Fatalf("openPrice mismatch: %f", result.Get("openPrice").Float())
	}
	if count.Load() != 3 {
		t.Fatalf("request count: %d, want 3", count.Load())
	}
	if elapsed < 2*time.Second {
		t.Fatalf("elapsed %s, Retry-After 未生效", elapsed)
	}
}

// TestGetRateLimitRetryBackoff 无 Retry-After 头时按指数退避重试(无需网络)
func TestGetRateLimitRetryBackoff(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if count.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.RateLimitBaseDelay = 20 * time.Millisecond
	client := NewClient(cfg)
	start := time.Now()
	_, err := client.Get(server.URL, nil, nil)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if count.Load() != 2 {
		t.Fatalf("request count: %d, want 2", count.Load())
	}
	if elapsed < 20*time.Millisecond {
		t.Fatalf("elapsed %s, 指数退避未生效", elapsed)
	}
}

// TestGetRateLimitRetryExhausted 超过最大重试次数仍 429 时返回错误(无需网络)
func TestGetRateLimitRetryExhausted(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.RateLimitMaxRetries = 2
	cfg.RateLimitBaseDelay = 10 * time.Millisecond
	client := NewClient(cfg)
	_, err := client.Get(server.URL, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("want 429 error, got %v", err)
	}
	if count.Load() != 3 { // 1 次原始请求 + 2 次重试
		t.Fatalf("request count: %d, want 3", count.Load())
	}
}

// TestGetRateLimitRetryBudgetTruncation Retry-After 远超剩余预算时被截断:
// Get 在预算耗尽后返回错误,不会按 Retry-After 睡满(无需网络)
func TestGetRateLimitRetryBudgetTruncation(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.RateLimitRetryBudget = 200 * time.Millisecond
	client := NewClient(cfg)
	start := time.Now()
	_, err := client.Get(server.URL, nil, nil)
	elapsed := time.Since(start)
	if err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("want 429 error, got %v", err)
	}
	if elapsed >= time.Second {
		t.Fatalf("elapsed %s, Retry-After 未按剩余预算截断", elapsed)
	}
	if count.Load() != 2 { // 首次 429 + 截断后最后一次重试
		t.Fatalf("request count: %d, want 2", count.Load())
	}
}

// TestGetContextCancelDuringRetrySleep ctx 在 429 重试等待期间到期时,
// GetContext 立即返回 DeadlineExceeded,不会睡满 Retry-After(无需网络)
func TestGetContextCancelDuringRetrySleep(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewClient(DefaultConfig())
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := client.GetContext(ctx, server.URL, nil, nil)
	elapsed := time.Since(start)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want context.DeadlineExceeded, got %v", err)
	}
	if elapsed >= time.Second {
		t.Fatalf("elapsed %s, ctx 到期未中断重试等待", elapsed)
	}
}

// TestGetRateLimitRetryBudgetTruncationSuccess 超出预算的 Retry-After 被截断后,
// 最后一次重试仍可在预算内成功拿到结果(无需网络)
func TestGetRateLimitRetryBudgetTruncationSuccess(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if count.Add(1) == 1 {
			w.Header().Set("Retry-After", "3600")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.RateLimitRetryBudget = 300 * time.Millisecond
	client := NewClient(cfg)
	start := time.Now()
	result, err := client.Get(server.URL, nil, nil)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if !result.Get("ok").Bool() {
		t.Fatalf("unexpected result: %s", result.Raw)
	}
	if elapsed >= time.Second {
		t.Fatalf("elapsed %s, Retry-After 未按剩余预算截断", elapsed)
	}
	if count.Load() != 2 { // 首次 429 + 截断后的最后一次重试
		t.Fatalf("request count: %d, want 2", count.Load())
	}
}

// TestFetchOpenPriceContextCancel 429 等待期间 ctx 到期,
// FetchOpenPriceContext 快速兜底返回 (0, 0),不会睡满 Retry-After(无需网络)
func TestFetchOpenPriceContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.Polymarket.CryptoPriceURL = server.URL
	client := NewClient(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	openPrice, closePrice := client.FetchOpenPriceContext(ctx, BTC,
		time.Now().Add(-10*time.Minute), time.Now().Add(-5*time.Minute),
		Fiveminute, true, 60)
	elapsed := time.Since(start)
	if openPrice != 0 || closePrice != 0 {
		t.Fatalf("want (0, 0), got (%f, %f)", openPrice, closePrice)
	}
	if elapsed >= time.Second {
		t.Fatalf("elapsed %s, ctx 到期未中断重试等待", elapsed)
	}
}

// TestFetchOpenPriceRateLimit FetchOpenPrice 命中 429 时按 Retry-After 头重试,
// 最终拿到价格,且请求参数正确传递(无需网络)
func TestFetchOpenPriceRateLimit(t *testing.T) {
	var count atomic.Int32
	var lastQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastQuery = r.URL.RawQuery
		if count.Add(1) <= 2 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"openPrice":63263.639611640705,"closePrice":63233.40981220359}`))
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.Polymarket.CryptoPriceURL = server.URL
	client := NewClient(cfg)

	start := time.Now()
	openPrice, closePrice := client.FetchOpenPrice(BTC,
		time.Date(2026, 8, 16, 9, 30, 0, 0, time.UTC),
		time.Date(2026, 8, 16, 9, 35, 0, 0, time.UTC),
		Fiveminute, true, 60)
	elapsed := time.Since(start)

	if openPrice != 63263.639611640705 || closePrice != 63233.40981220359 {
		t.Fatalf("price mismatch: open=%f close=%f", openPrice, closePrice)
	}
	if count.Load() != 3 {
		t.Fatalf("request count: %d, want 3", count.Load())
	}
	if elapsed < 2*time.Second {
		t.Fatalf("elapsed %s, Retry-After 未生效", elapsed)
	}
	for _, kv := range []string{
		"symbol=BTC",
		"eventStartTime=2026-08-16T09%3A30%3A00.000Z",
		"endDate=2026-08-16T09%3A35%3A00.000Z",
		"variant=fiveminute",
		"twapEnabled=true",
		"twapLookbackSeconds=60",
	} {
		if !strings.Contains(lastQuery, kv) {
			t.Fatalf("query 缺少 %s: %s", kv, lastQuery)
		}
	}
}

// TestFetchOpenPriceRateLimitExhausted 一直 429 时重试耗尽,FetchOpenPrice 兜底返回 (0, 0)(无需网络)
func TestFetchOpenPriceRateLimitExhausted(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.Polymarket.CryptoPriceURL = server.URL
	cfg.RateLimitMaxRetries = 2
	cfg.RateLimitBaseDelay = 10 * time.Millisecond
	client := NewClient(cfg)

	openPrice, closePrice := client.FetchOpenPrice(BTC,
		time.Now().Add(-10*time.Minute), time.Now().Add(-5*time.Minute),
		Fiveminute, true, 60)
	if openPrice != 0 || closePrice != 0 {
		t.Fatalf("want (0, 0), got (%f, %f)", openPrice, closePrice)
	}
	if count.Load() != 3 { // 1 次原始请求 + 2 次重试
		t.Fatalf("request count: %d, want 3", count.Load())
	}
}

// TestFetchOpenPrice 实盘冒烟测试:并发快速请求真实 crypto-price 接口直到触发真实 429,
// 验证限流自动重试后请求仍能拿到 openPrice(必须观察到至少一次 429 才算通过)
// 运行: https_proxy=http://127.0.0.1:1087 go test -v -run TestFetchOpenPrice -timeout 240s ./polymarket/
func TestFetchOpenPrice(t *testing.T) {
	client := NewClient(DefaultConfig())
	// 取 10 分钟前结束的 5 分钟窗口,确保数据已完成缓存
	endDate := time.Now().UTC().Truncate(5*time.Minute).Add(-10 * time.Minute)
	startTime := endDate.Add(-5 * time.Minute)

	const workers = 3
	var (
		wg        sync.WaitGroup
		stop      atomic.Bool
		total     atomic.Int64
		zeroPrice atomic.Int64
	)
	// 兜底:120s 内没触发 429 也收尾(最终会报错)
	go func() {
		time.Sleep(120 * time.Second)
		stop.Store(true)
	}()

	for id := 0; id < workers; id++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for !stop.Load() {
				total.Add(1)
				openPrice, closePrice := client.FetchOpenPrice(BTC, startTime, endDate, Fiveminute, true, 60)
				if openPrice <= 0 {
					zeroPrice.Add(1)
					log.Printf("[worker %d] openPrice 异常: %f (closePrice: %f)", id, openPrice, closePrice)
				}
				// 观察到真实 429 后立即收尾
				if client.RateLimitRetryCount.Load() > 0 {
					stop.Store(true)
				}
			}
		}(id)
	}
	wg.Wait()

	t.Logf("total requests: %d, 429 retries: %d, zero-price: %d",
		total.Load(), client.RateLimitRetryCount.Load(), zeroPrice.Load())
	if client.RateLimitRetryCount.Load() == 0 {
		t.Fatalf("未触发真实 429,提高并发或延长兜底时间")
	}
	if zeroPrice.Load() > 0 {
		t.Fatalf("%d 次请求 429 重试耗尽后仍未拿到价格", zeroPrice.Load())
	}
}
