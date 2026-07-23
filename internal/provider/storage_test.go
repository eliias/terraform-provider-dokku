package provider

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestParseStorageMountReport(t *testing.T) {
	stdout := `{
  "attachment.1.container-path": "/app/uploads",
  "attachment.1.entry-name": "app-data",
  "attachment.1.host-path": "/var/lib/dokku/data/storage/app-data",
  "attachment.1.phases": "deploy,run",
  "attachment.1.process-type": "_default_",
  "attachment.1.readonly": "true",
  "attachment.1.subpath": "uploads",
  "attachment.1.volume-chown": "herokuish",
  "attachment.1.volume-options": "noexec,nosuid",
  "storage-attachment.1.entry-name": "app-data",
  "deploy-mounts": "-v ignored"
}`

	got, err := parseStorageMountReport("example", stdout)
	if err != nil {
		t.Fatalf("parseStorageMountReport returned an error: %v", err)
	}
	want := []dokkuStorageMount{
		{
			App:           "example",
			StorageEntry:  "app-data",
			ContainerPath: "/app/uploads",
			Phases:        []string{"deploy", "run"},
			ProcessType:   "_default_",
			Subpath:       "uploads",
			Readonly:      true,
			VolumeOptions: "noexec,nosuid",
			VolumeChown:   "herokuish",
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("unexpected storage mounts (-want +got):\n%s", diff)
	}
}

func TestParseStorageMountReportEmpty(t *testing.T) {
	got, err := parseStorageMountReport("example", `{"deploy-mounts":""}`)
	if err != nil {
		t.Fatalf("parseStorageMountReport returned an error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no mounts, got %#v", got)
	}
}

func TestImportStorageMount(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceStorageMount().Schema, nil)
	d.SetId("gefa|legacy-033136a327|/app/wp/wp-content/themes|_default_")

	resources, err := importStorageMount(context.Background(), d, nil)
	if err != nil {
		t.Fatalf("importStorageMount returned an error: %v", err)
	}
	imported := resources[0]

	if got := imported.Get("app"); got != "gefa" {
		t.Errorf("unexpected app: %q", got)
	}
	if got := imported.Get("storage_entry"); got != "legacy-033136a327" {
		t.Errorf("unexpected storage entry: %q", got)
	}
	if got := imported.Get("container_path"); got != "/app/wp/wp-content/themes" {
		t.Errorf("unexpected container path: %q", got)
	}
	if got := imported.Get("process_type"); got != "_default_" {
		t.Errorf("unexpected process type: %q", got)
	}
}

func TestImportStorageMountRejectsMalformedID(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceStorageMount().Schema, nil)
	d.SetId("gefa|app-data")

	if _, err := importStorageMount(context.Background(), d, nil); err == nil {
		t.Fatal("expected malformed import ID to be rejected")
	}
}
