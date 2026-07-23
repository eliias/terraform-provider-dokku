package provider

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseAppResourceLimits(t *testing.T) {
	stdout := `=====> example resource information
       resource _default_ limit cpu:             0.5
       resource _default_ limit memory:          512m
       resource _default_ limit memory swap:     1g
       resource web limit network:               10mbit
       resource web limit network ingress:       8mbit
       resource web limit network egress:        2mbit
       resource worker limit nvidia gpu:         1
       resource worker reservation memory:       256m
`

	got := resourceLimitLookup(parseAppResourceLimits(stdout))
	want := map[string]DokkuAppResourceLimit{
		"_default_": {
			ProcessType: "_default_",
			CPU:         "0.5",
			Memory:      "512m",
			MemorySwap:  "1g",
		},
		"web": {
			ProcessType:    "web",
			Network:        "10mbit",
			NetworkIngress: "8mbit",
			NetworkEgress:  "2mbit",
		},
		"worker": {
			ProcessType: "worker",
			NvidiaGPU:   "1",
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("unexpected parsed resource limits (-want +got):\n%s", diff)
	}
}

func TestParseAppResourceLimitsEmptyReport(t *testing.T) {
	got := parseAppResourceLimits("=====> example resource information\n")
	if len(got) != 0 {
		t.Fatalf("expected no resource limits, got %#v", got)
	}
}

func TestResourceLimitsInterfaceRoundTrip(t *testing.T) {
	want := []DokkuAppResourceLimit{
		{
			ProcessType:    "web",
			CPU:            "1",
			Memory:         "512m",
			MemorySwap:     "1g",
			Network:        "10mbit",
			NetworkIngress: "8mbit",
			NetworkEgress:  "2mbit",
			NvidiaGPU:      "1",
		},
	}

	got := resourceLimitsFromInterfaces(resourceLimitsToInterfaces(want))
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("resource limit conversion did not round-trip (-want +got):\n%s", diff)
	}
}

func resourceLimitLookup(limits []DokkuAppResourceLimit) map[string]DokkuAppResourceLimit {
	lookup := make(map[string]DokkuAppResourceLimit, len(limits))
	for _, limit := range limits {
		lookup[limit.ProcessType] = limit
	}
	return lookup
}
