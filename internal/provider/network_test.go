package provider

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseAppNetworkReport(t *testing.T) {
	stdout := `{
  "attach-post-create": "backend,metrics",
  "attach-post-deploy": "frontend",
  "bind-all-interfaces": "false",
  "computed-attach-post-create": "backend,metrics",
  "computed-attach-post-deploy": "frontend",
  "computed-bind-all-interfaces": "false",
  "computed-initial-network": "private",
  "computed-tld": "svc.internal",
  "initial-network": "private",
  "static-web-listener": "10.0.0.10:8080",
  "tld": "svc.internal",
  "web-listeners": "10.0.0.10:8080,10.0.0.11:8080"
}`

	got, err := parseAppNetworkReport("example", stdout)
	if err != nil {
		t.Fatalf("parseAppNetworkReport returned an error: %v", err)
	}
	want := dokkuAppNetwork{
		App:                       "example",
		AttachPostCreate:          []string{"backend", "metrics"},
		AttachPostDeploy:          []string{"frontend"},
		InitialNetwork:            "private",
		BindAllInterfaces:         "false",
		TLD:                       "svc.internal",
		StaticWebListener:         "10.0.0.10:8080",
		ComputedAttachPostCreate:  []string{"backend", "metrics"},
		ComputedAttachPostDeploy:  []string{"frontend"},
		ComputedInitialNetwork:    "private",
		ComputedBindAllInterfaces: false,
		ComputedTLD:               "svc.internal",
		WebListeners:              []string{"10.0.0.10:8080", "10.0.0.11:8080"},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("unexpected app network (-want +got):\n%s", diff)
	}
}

func TestParseAppNetworkReportLegacyKeys(t *testing.T) {
	stdout := `{
  "network-attach-post-create": "",
  "network-attach-post-deploy": "shared",
  "network-bind-all-interfaces": "true",
  "network-computed-attach-post-create": "",
  "network-computed-attach-post-deploy": "shared",
  "network-computed-bind-all-interfaces": "true",
  "network-computed-initial-network": "",
  "network-computed-tld": "",
  "network-initial-network": "",
  "network-static-web-listener": "",
  "network-tld": "",
  "network-web-listeners": "172.17.0.2:5000"
}`

	got, err := parseAppNetworkReport("legacy", stdout)
	if err != nil {
		t.Fatalf("parseAppNetworkReport returned an error: %v", err)
	}
	if diff := cmp.Diff([]string{"shared"}, got.AttachPostDeploy); diff != "" {
		t.Fatalf("unexpected post-deploy networks (-want +got):\n%s", diff)
	}
	if got.BindAllInterfaces != "true" || !got.ComputedBindAllInterfaces {
		t.Fatal("expected bind-all-interfaces legacy values to be true")
	}
	if diff := cmp.Diff([]string{"172.17.0.2:5000"}, got.WebListeners); diff != "" {
		t.Fatalf("unexpected web listeners (-want +got):\n%s", diff)
	}
}

func TestSplitNetworkValues(t *testing.T) {
	got := splitNetworkValues("alpha,beta gamma\tomega\n")
	want := []string{"alpha", "beta", "gamma", "omega"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("unexpected network values (-want +got):\n%s", diff)
	}
}
