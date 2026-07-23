package provider

import (
	"reflect"
	"testing"
)

func TestInterfaceSliceToStrSlice(t *testing.T) {
	got := interfaceSliceToStrSlice([]interface{}{"backend", "metrics"})
	want := []string{"backend", "metrics"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected conversion: got %#v, want %#v", got, want)
	}
}
