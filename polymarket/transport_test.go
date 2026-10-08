package polymarket

import (
	"crypto/tls"
	"net/http"
	"strings"
	"testing"
)

// TestNewClientHTTP1AndTLSConfig 校验 NewClient 构造出的 transport 真身（无网络）：
//   - TLS 配置确实挂在最终 transport 上（回归钉子：曾被 SetTransport 整体覆盖丢失）
//   - DisableHTTP2=true 时 HTTP/2 被彻底关死，且 DisableHTTP2=false 时能恢复
func TestNewClientHTTP1AndTLSConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Polymarket.OwnerKey = strings.Repeat("1", 64) // NewClient 会 HexToECDSA，需要合法占位 key
	c := NewClient(cfg)                               // DisableHTTP2 默认 true

	tr, ok := c.http.Transport().(*http.Transport)
	if !ok {
		t.Fatalf("transport 类型 = %T", c.http.Transport())
	}
	// ① TLS 配置没有被 SetTransport 丢弃（回归钉子）
	if tr.TLSClientConfig == nil {
		t.Fatal("TLSClientConfig 为 nil —— 被 SetTransport 覆盖丢失")
	}
	if tr.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = %v, want TLS 1.2", tr.TLSClientConfig.MinVersion)
	}
	if tr.TLSClientConfig.ClientSessionCache == nil {
		t.Error("ClientSessionCache 丢失")
	}
	// ② HTTP/2 关死
	if tr.ForceAttemptHTTP2 {
		t.Error("ForceAttemptHTTP2 仍为 true")
	}
	if tr.Protocols == nil || !tr.Protocols.HTTP1() || tr.Protocols.HTTP2() {
		t.Errorf("Protocols = %v, want 仅 HTTP/1.1", tr.Protocols)
	}
	for _, p := range tr.TLSClientConfig.NextProtos {
		if p == "h2" {
			t.Errorf("NextProtos 仍含 h2: %v", tr.TLSClientConfig.NextProtos)
		}
	}
	// ③ 开关为 false 时恢复 h2 能力（防开关写反）
	cfg2 := DefaultConfig()
	cfg2.Polymarket.OwnerKey = strings.Repeat("1", 64)
	cfg2.DisableHTTP2 = false
	tr2 := NewClient(cfg2).http.Transport().(*http.Transport)
	if !tr2.ForceAttemptHTTP2 {
		t.Error("DisableHTTP2=false 时未恢复 HTTP/2")
	}
	if tr2.Protocols != nil && !tr2.Protocols.HTTP2() {
		t.Errorf("DisableHTTP2=false 时 Protocols 未放开 HTTP/2: %v", tr2.Protocols)
	}
	if tr2.TLSClientConfig == nil || tr2.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Error("关闭 HTTP/2 时不应影响 TLS 配置")
	}
}
