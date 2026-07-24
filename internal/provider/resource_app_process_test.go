package provider

import (
	"reflect"
	"testing"
)

func TestProcessScaleToMap(t *testing.T) {
	t.Parallel()

	got := processScaleToMap([]dokkuAppProcessScale{
		{ProcessType: "web", Quantity: 1},
		{ProcessType: "worker", Quantity: 0},
	})
	want := map[string]int{"web": 1, "worker": 0}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
