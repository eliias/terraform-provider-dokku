package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestMariadbServiceUsesMariadbCommands(t *testing.T) {
	service := NewMariadbService("example")

	if service.CmdName != "mariadb" {
		t.Fatalf("unexpected command name: %q", service.CmdName)
	}
	if got := service.Cmd("info", service.Name); got != "mariadb:info example" {
		t.Fatalf("unexpected command: %q", got)
	}
}

func TestMariadbCreateFlagsIncludeResourceLimits(t *testing.T) {
	service := &DokkuGenericService{
		MemoryMB: 512,
		ShmSize:  "256m",
	}

	if got := createServiceFlagStr(service); got != " --memory 512 --shm-size 256m" {
		t.Fatalf("unexpected create flags: %q", got)
	}
}

func TestMariadbResourcesAreRegistered(t *testing.T) {
	resources := Provider().ResourcesMap

	for _, name := range []string{
		"dokku_mariadb_service",
		"dokku_mariadb_service_link",
	} {
		if resources[name] == nil {
			t.Errorf("provider does not register %s", name)
		}
	}
}

func TestImportMariadbServiceLink(t *testing.T) {
	resource := resourceMariadbServiceLink()
	data := schema.TestResourceDataRaw(t, resource.Schema, nil)
	data.SetId("database/application")

	imported, err := importServiceLinkState(context.Background(), data, nil)
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
}

func TestImportMariadbServiceLinkRejectsInvalidID(t *testing.T) {
	resource := resourceMariadbServiceLink()

	for _, id := range []string{"", "database", "/application", "database/", "database/application/extra"} {
		t.Run(id, func(t *testing.T) {
			data := schema.TestResourceDataRaw(t, resource.Schema, nil)
			data.SetId(id)

			if _, err := importServiceLinkState(context.Background(), data, nil); err == nil {
				t.Fatalf("expected import ID %q to fail", id)
			}
		})
	}
}
