package polymarket

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

var (
	TIME_UNITS    = []string{"15m", "hourly", "4h", "daily", "weekly", "monthly", "5m"}
	TIME_UNIT_MAP = map[string]string{
		"5m":      "fiveminute",
		"15m":     "fifteen",
		"hourly":  "hourly",
		"4h":      "fourhour",
		"daily":   "daily",
		"weekly":  "weekly",
		"monthly": "monthly",
	}
	SYMBOL_MAP = map[string]string{
		"sol":      "SOL",
		"solana":   "SOL",
		"eth":      "ETH",
		"ethereum": "ETH",
		"btc":      "BTC",
		"bitcoin":  "BTC",
		"xrp":      "XRP",
		"dogecoin": "DOGE",
	}
	SLUGS = utils.Keys(SYMBOL_MAP)
)

func GetSymbol(tags []gjson.Result) (CryptoPriceSymbol, error) {
	slugs := utils.Map(tags, func(t gjson.Result) string {
		return strings.ToLower(t.Get("slug").String())
	})
	for _, slug := range SLUGS {
		if slices.Contains(slugs, slug) {
			return CryptoPriceSymbol(SYMBOL_MAP[slug]), nil
		}
	}
	return "", errors.New("no symbol")
}

func GetTimeUnit(tags []gjson.Result) (string, error) {
	slugs := utils.Map(tags, func(t gjson.Result) string {
		return strings.ToLower(t.Get("slug").String())
	})
	// log.Printf("slugs: %+v", slugs)
	for _, unit := range TIME_UNITS {
		if slices.Contains(slugs, unit) {
			return unit, nil
		}
	}
	return "", errors.New("no timeUnit")
}

func GetSearchTimeUnit(unit string) (CryptoPriceUint, error) {
	if unit, ok := TIME_UNIT_MAP[unit]; ok {
		return CryptoPriceUint(unit), nil
	}
	return "", errors.New("no SearchTimeUnit")
}

func GetStartTime(unit string, endDate string) time.Time {
	startDate := utils.TimeParse(endDate)
	switch unit {
	case TIME_UNITS[0]:
		startDate = startDate.Add(-15 * time.Minute)
	case TIME_UNITS[1]:
		startDate = startDate.Add(-1 * time.Hour)
	case TIME_UNITS[2]:
		startDate = startDate.Add(-4 * time.Hour)
	case TIME_UNITS[3]:
		startDate = startDate.AddDate(0, 0, -1)
	case TIME_UNITS[4]:
		startDate = startDate.AddDate(0, 0, -7)
	case TIME_UNITS[5]:
		startDate = startDate.AddDate(0, 1, 0)
	case TIME_UNITS[6]:
		startDate = startDate.Add(-5 * time.Minute)
	}
	return startDate
}
