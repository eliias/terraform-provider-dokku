---
page_title: "dokku_network Resource - terraform-provider-dokku"
subcategory: ""
description: |-
  Manages an attachable Docker network created by Dokku.
---

# dokku_network (Resource)

Manages a Docker bridge network through Dokku. Only networks created and
labelled by Dokku can be imported as resources. Use the `dokku_network` data
source to inspect built-in or externally managed Docker networks.

Docker refuses to destroy a network while containers remain attached.
Production networks should normally use `lifecycle.prevent_destroy`.

## Example Usage

```hcl
resource "dokku_network" "backend" {
  name = "backend"

  lifecycle {
    prevent_destroy = true
  }
}

resource "dokku_app" "example" {
  name       = "example"
  depends_on = [dokku_network.backend]
}
```

When an app uses a managed network, make the app depend on the network as
shown above. This guarantees that Terraform destroys the app containers before
attempting to destroy the network.

## Import

```shell
terraform import dokku_network.backend backend
```

## Schema

### Required

- `name` (String, ForceNew) Dokku network name.

### Read-Only

- `dokku_managed` (Boolean) Whether Dokku created and owns the network.
- `driver` (String) Docker network driver.
- `internal` (Boolean) Whether the network is internal.
- `ipv6` (Boolean) Whether IPv6 is enabled.
- `labels` (Map of String) Docker network labels.
- `network_id` (String) Docker network ID.
- `scope` (String) Docker network scope.
