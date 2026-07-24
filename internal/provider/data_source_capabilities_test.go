package provider

import (
	"reflect"
	"testing"
)

func TestParseAndClassifyDokkuPluginCapabilities(t *testing.T) {
	t.Parallel()

	plugins, err := parseDokkuPluginCapabilities(`[
		{"name":"scheduler-null","version":"0.38.25","enabled":true,"core":true},
		{"name":"disabled-plugin","version":"1.0.0","enabled":false,"core":false},
		{"name":"builder-dockerfile","version":"0.38.25","enabled":true,"core":true},
		{"name":"nginx-vhosts","version":"0.38.25","enabled":true,"core":true},
		{"name":"scheduler-docker-local","version":"0.38.25","enabled":true,"core":true}
	]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 4 {
		t.Fatalf("got %d enabled plugins, want 4", len(plugins))
	}

	builders, schedulers, proxies := classifyDokkuPluginCapabilities(plugins)
	if !reflect.DeepEqual(builders, []string{"dockerfile"}) {
		t.Fatalf("unexpected builders: %#v", builders)
	}
	if !reflect.DeepEqual(schedulers, []string{"docker-local", "null"}) {
		t.Fatalf("unexpected schedulers: %#v", schedulers)
	}
	if !reflect.DeepEqual(proxies, []string{"nginx"}) {
		t.Fatalf("unexpected proxies: %#v", proxies)
	}
}
