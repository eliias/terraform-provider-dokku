package provider

import (
	"reflect"
	"testing"
)

func TestChangedConfigVars(t *testing.T) {
	desired := map[string]string{
		"ALREADY_SET": "same",
		"CHANGED":     "new",
		"NEW":         "value",
	}
	current := map[string]string{
		"ALREADY_SET": "same",
		"CHANGED":     "old",
		"UNMANAGED":   "preserved",
	}
	expected := map[string]string{
		"CHANGED": "new",
		"NEW":     "value",
	}

	if got := changedConfigVars(desired, current); !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected changed variables: %#v", got)
	}
}
