package tests

import (
	"testing"
	"time"

	"github.com/xiangxn/go-polymarket-sdk/utils"
)

func TestUtilsToISOString(t *testing.T) {
	str := utils.ToISOString(time.Now())
	t.Logf("str: %s", str)
}
