package utils

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"
)

func ToTimestamp(dateStr string) (int64, error) {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return 0, err
	}
	return t.UnixMilli(), nil
}

// RoundTo15Minutes 将时间向下舍入到最近的15分钟边界，返回Unix时间戳（秒）
func RoundTo15Minutes(date ...time.Time) int64 {
	var d time.Time
	if len(date) == 0 {
		d = time.Now()
	} else {
		d = date[0]
	}

	minutes := d.Minute()
	floored := (minutes / 15) * 15

	rounded := time.Date(d.Year(), d.Month(), d.Day(), d.Hour(), floored, 0, 0, d.Location())
	return rounded.Unix()
}

func RoundNormal(num float64, decimals int) float64 {
	d := decimal.NewFromFloat(num)
	scale := decimal.New(1, int32(-decimals))

	return d.Div(scale).Round(0).Mul(scale).InexactFloat64()
}

func RoundDown(num float64, decimals int) float64 {
	// if DecimalPlaces(num) <= decimals {
	// 	log.Printf("1 === decimals: %d", decimals)
	// 	return num
	// }
	// multiplier := math.Pow10(decimals)
	// return math.Floor(num*multiplier) / multiplier

	d := decimal.NewFromFloat(num)
	scale := decimal.New(1, int32(-decimals))
	return d.Div(scale).Floor().Mul(scale).InexactFloat64()
}

func RoundUp(num float64, decimals int) float64 {
	// if DecimalPlaces(num) <= decimals {
	// 	return num
	// }
	// multiplier := math.Pow10(decimals)
	// return math.Ceil(num*multiplier) / multiplier

	d := decimal.NewFromFloat(num)
	scale := decimal.New(1, int32(-decimals))
	return d.Div(scale).Ceil().Mul(scale).InexactFloat64()
}

func ParseUnits(amount string, decimals int) (*big.Int, error) {
	parts := strings.SplitN(amount, ".", 2)
	intPart := parts[0]

	fracPart := ""
	if len(parts) == 2 {
		fracPart = parts[1]
		if len(fracPart) > decimals {
			fracPart = fracPart[:decimals] // 截断多余小数位
		}
	}

	// 补齐小数位
	for len(fracPart) < decimals {
		fracPart += "0"
	}

	result := new(big.Int)
	_, ok := result.SetString(intPart+fracPart, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %s", amount)
	}

	return result, nil
}

func FloatToString(num float64, maxDecimals int) string {
	// 如果是整数，直接返回整数部分
	if num == math.Trunc(num) {
		return strconv.FormatInt(int64(num), 10)
	}

	// 默认使用 strconv.FormatFloat 精度 -1（尽量少但不丢信息）
	s := strconv.FormatFloat(num, 'f', -1, 64)

	// 如果指定了最大小数位，进行截断
	if maxDecimals > 0 {
		parts := strings.SplitN(s, ".", 2)
		if len(parts) == 2 && len(parts[1]) > maxDecimals {
			frac := parts[1][:maxDecimals]
			// 去掉末尾多余的 0
			frac = strings.TrimRight(frac, "0")
			if frac == "" {
				return parts[0]
			}
			return parts[0] + "." + frac
		}
	}

	return s
}

func StringPtr(s string) *string {
	return &s
}

func GetStringArray(obj *gjson.Result, path string) []string {
	arr := obj.Get(path).Array()
	res := make([]string, 0, len(arr))
	for _, v := range arr {
		res = append(res, v.String())
	}
	return res
}

func SleepWithCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func SafeCall(fn func()) {
	defer func() {
		_ = recover()
	}()
	fn()
}

func ToISOString(t time.Time) string {
	return t.UTC().
		Truncate(time.Millisecond).
		Format("2006-01-02T15:04:05.000Z")
}
