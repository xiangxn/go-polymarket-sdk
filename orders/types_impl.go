package orders

import "fmt"

func (t TickSize) IsValid() bool {
	switch t {
	case TickSize01, TickSize001, TickSize0001, TickSize00001:
		return true
	default:
		return false
	}
}

func NewTickSize(v string) (TickSize, error) {
	ts := TickSize(v)
	if !ts.IsValid() {
		return "", fmt.Errorf("invalid TickSize: %s", v)
	}
	return ts, nil
}

func GetRoundConfig(t TickSize) *RoundConfig {
	switch t {
	case TickSize01:
		return &RoundConfig{Price: 1, Size: 2, Amount: 3}
	case TickSize001:
		return &RoundConfig{Price: 2, Size: 2, Amount: 4}
	case TickSize0001:
		return &RoundConfig{Price: 3, Size: 2, Amount: 5}
	case TickSize00001:
		return &RoundConfig{Price: 4, Size: 2, Amount: 6}
	default:
		panic("invalid TickSize: " + string(t))
	}
}
