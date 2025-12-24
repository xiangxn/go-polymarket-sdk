package tests

import (
	"fmt"
	"log"
	"testing"
)

func TestFmt(t *testing.T) {
	rawData0 := []byte(fmt.Sprintf("\x19\x01%s%s", "12345", "45678"))
	rawData1 := fmt.Appendf(nil, "\x19\x01%s%s", "12345", "45678")

	log.Println(rawData0)
	log.Println(rawData1)
}
