package tests

import (
	"reflect"
	"testing"
	"time"

	"github.com/tidwall/gjson"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

func TestUtilsToISOString(t *testing.T) {
	str := utils.ToISOString(time.Now())
	t.Logf("str: %s", str)
}

func TestGetStringArray(t *testing.T) {
	obj := gjson.Parse(`{"prices":"[\"0.12\",\"0.34\",\"0.56\"]","empty":[]}`)

	got := utils.GetStringArray(&obj, "prices")
	want := []string{"0.12", "0.34", "0.56"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetStringArray() = %v, want %v", got, want)
	}

	empty := utils.GetStringArray(&obj, "empty")
	if len(empty) != 0 {
		t.Fatalf("GetStringArray(empty) len = %d, want 0", len(empty))
	}
}

func TestGetFloatArray(t *testing.T) {
	obj := gjson.Parse(`{"prices":"[\"0.12\",\"abc\",\"0.56\"]","empty":[]}`)

	got := utils.GetFloatArray(&obj, "prices")
	want := []float64{0.12, 0, 0.56}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetFloatArray() = %v, want %v", got, want)
	}

	empty := utils.GetFloatArray(&obj, "empty")
	if empty != nil {
		t.Fatalf("GetFloatArray(empty) = %v, want nil", empty)
	}

	missing := utils.GetFloatArray(&obj, "notFound")
	if missing != nil {
		t.Fatalf("GetFloatArray(notFound) = %v, want nil", missing)
	}
}
