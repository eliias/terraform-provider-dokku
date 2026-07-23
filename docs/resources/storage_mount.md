---
page_title: "dokku_storage_mount Resource - terraform-provider-dokku"
subcategory: ""
description: |-
  Mounts a named Dokku storage entry into an application.
---

# dokku_storage_mount (Resource)

Mounts a named storage entry into a Dokku application. Changes are stored
immediately but affect containers created by a subsequent restart or
deployment.

## Example Usage

```hcl
resource "dokku_storage_mount" "uploads" {
  app            = dokku_app.example.name
  storage_entry  = dokku_storage_entry.uploads.name
  container_path = "/app/uploads"
  phases         = ["deploy", "run"]
}
```

## Import

Import IDs contain the app, entry, container path, and process type separated
by `|`:

```shell
terraform import dokku_storage_mount.uploads \
  'example|example-uploads|/app/uploads|_default_'
```

## Schema

### Required

- `app` (String, ForceNew) Dokku application receiving the mount.
- `storage_entry` (String, ForceNew) Named storage entry to mount.
- `container_path` (String, ForceNew) Absolute path inside the container.

### Optional

- `phases` (Set of String) Container phases receiving the mount. Dokku defaults to `deploy` and `run`.
- `process_type` (String, ForceNew) Process type receiving the mount. Defaults to `_default_`.
- `subpath` (String) Subdirectory within the storage entry to mount.
- `readonly` (Boolean) Whether to mount the entry read-only. Defaults to `false`.
- `volume_options` (String) Comma-separated Docker mount options.
- `volume_chown` (String) Ownership mode applied by Dokku at mount time.

### Read-Only

- `id` (String) Composite mount identity.
