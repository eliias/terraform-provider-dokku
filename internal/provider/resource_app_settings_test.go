package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestAppSettingsReportSchemas(t *testing.T) {
	t.Parallel()

	resources := map[string]*schema.Resource{
		"builder": resourceAppBuilder(),
		"proxy":   resourceAppProxy(),
		"nginx":   resourceAppNginx(),
	}
	for name, resource := range resources {
		if resource.Importer == nil {
			t.Fatalf("%s resource has no importer", name)
		}
		if resource.Schema["app"] == nil || !resource.Schema["app"].ForceNew {
			t.Fatalf("%s resource does not have a ForceNew app attribute", name)
		}
	}
}

func TestDokkuPropertyName(t *testing.T) {
	t.Parallel()
	if got := dokkuPropertyName("proxy_busy_buffers_size"); got != "proxy-busy-buffers-size" {
		t.Fatalf("unexpected property name %q", got)
	}
}
