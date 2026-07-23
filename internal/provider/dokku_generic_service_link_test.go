package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestServiceLinkImporters(t *testing.T) {
	resources := map[string]*schema.Resource{
		"postgres": resourcePostgresServiceLink(),
		"redis":    resourceRedisServiceLink(),
	}

	for name, resource := range resources {
		t.Run(name, func(t *testing.T) {
			if resource.Importer == nil {
				t.Fatal("expected resource importer")
			}

			data := schema.TestResourceDataRaw(t, resource.Schema, nil)
			data.SetId("database/application")

			imported, err := resource.Importer.StateContext(context.Background(), data, nil)
			if err != nil {
				t.Fatalf("import failed: %v", err)
			}
			if len(imported) != 1 {
				t.Fatalf("expected one imported resource, got %d", len(imported))
			}
			if got := data.Get("service"); got != "database" {
				t.Errorf("unexpected service: %q", got)
			}
			if got := data.Get("app"); got != "application" {
				t.Errorf("unexpected app: %q", got)
			}
			if got := data.Id(); got != "database-application" {
				t.Errorf("unexpected ID: %q", got)
			}
		})
	}
}
