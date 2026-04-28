package orders

import (
	"crypto/rand"
	"math"
	"math/big"
	"strconv"
)

func IsTickSizeSmaller(a TickSize, b TickSize) bool {
	a1, err := strconv.ParseFloat(string(a), 64)
	if err != nil {
		return false
	}
	b1, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return false
	}
	return a1 < b1
}

func PriceValid(price float64, tickSize TickSize) bool {
	tickSize1, err := strconv.ParseFloat(string(tickSize), 64)
	if err != nil {
		return false
	}
	return price >= tickSize1 && price <= 1-tickSize1
}

func ConvertTickSize(tickSize TickSize) float64 {
	tickSize1, err := strconv.ParseFloat(string(tickSize), 64)
	if err != nil {
		return 0
	}
	return tickSize1
}

func GenerateRandomSalt() int64 {
	maxInt := math.Pow(2, 32)
	nBig, _ := rand.Int(rand.Reader, big.NewInt(int64(maxInt)))
	return nBig.Int64()
}
