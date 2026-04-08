package tests

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sdkModel "github.com/xiangxn/go-polymarket-sdk/model"
	"github.com/xiangxn/go-polymarket-sdk/polymarket"
)

func TestTradeMonitorLive_FromDotEnv(t *testing.T) {
	loadDotEnv()

	wsURL := getEnvOrDefault("CLOB_WS_BASE_URL", "wss://ws-subscriptions-clob.polymarket.com")
	funder := strings.TrimSpace(os.Getenv("FUNDERADDRESS"))
	if funder == "" {
		t.Skip("skip live test: FUNDERADDRESS is empty")
	}

	t.Logf("funder: %s", funder)

	creds := &sdkModel.ApiKeyCreds{
		Key:        strings.TrimSpace(os.Getenv("CLOB_API_KEY")),
		Secret:     strings.TrimSpace(os.Getenv("CLOB_SECRET")),
		Passphrase: strings.TrimSpace(os.Getenv("CLOB_PASSPHRASE")),
	}
	if creds.Key == "" || creds.Secret == "" || creds.Passphrase == "" {
		t.Skip("skip live test: CLOB_API_KEY/CLOB_SECRET/CLOB_PASSPHRASE is not fully set")
	}

	tm := polymarket.NewTradeMonitor(wsURL, "MINED", creds)

	ctx, cancel := context.WithTimeout(context.Background(), 480*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- tm.Run(ctx)
	}()
	defer tm.Close()

	fillCount := 0
	for {
		select {
		case fill := <-tm.Subscribe():
			fillCount++
			t.Logf("got live fill #%d: %+v", fillCount, fill)
		case err := <-errCh:
			if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("trade monitor run failed: %v", err)
			}
			t.Logf("trade monitor exited early, fillCount=%d, err=%v", fillCount, err)
			return
		case <-ctx.Done():
			err := <-errCh
			if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("trade monitor run failed: %v", err)
			}
			t.Logf("listen finished after 480s, fillCount=%d", fillCount)
			return
		}
	}
}

func getEnvOrDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func loadDotEnv() {
	paths := []string{".env", filepath.Join("..", ".env")}
	for _, p := range paths {
		content, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for line := range strings.SplitSeq(string(content), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "export ") {
				line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			v = strings.Trim(v, `"'`)
			if k != "" {
				_, exists := os.LookupEnv(k)
				if !exists {
					_ = os.Setenv(k, v)
				}
			}
		}
		return
	}
}
