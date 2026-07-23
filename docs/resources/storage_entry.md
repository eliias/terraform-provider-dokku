---
page_title: "dokku_storage_entry Resource - terraform-provider-dokku"
subcategory: ""
description: |-
  Manages a named docker-local Dokku storage entry.
---

# dokku_storage_entry (Resource)

Manages a named Dokku storage entry backed by a host directory or Docker
volume. Named storage entries require Dokku 0.38 or newer.

Destroying a storage entry may remove its underlying storage. Production
entries should normally use `lifecycle.prevent_destroy`.

## Example Usage

```hcl
resource "dokku_storage_entry" "uploads" {
  name      = "example-uploads"
  host_path = "/var/lib/dokku/data/storage/example-uploads"

  lifecycle {
    prevent_destroy = true
  }
}
```

## Import

Import an existing entry by its globally unique name:

```shell
terraform import dokku_storage_entry.uploads example-uploads
```

## Schema

### Required

- `name` (String, ForceNew) Globally unique DNS-1123 storage entry name.

### Optional

- `host_path` (String, ForceNew) Host directory or Docker volume backing the entry. Dokku uses its default storage directory when omitted.
- `scheduler` (String, ForceNew) Storage scheduler. Currently only `docker-local` is supported.

### Read-Only

- `id` (String) The storage entry name.
