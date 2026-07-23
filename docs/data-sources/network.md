---
page_title: "dokku_network Data Source - terraform-provider-dokku"
subcategory: ""
description: |-
  Reads metadata for a Docker network visible to Dokku.
---

# dokku_network (Data Source)

Reads any Docker network returned by `dokku network:info`, including built-in
and externally managed networks.

```hcl
data "dokku_network" "bridge" {
  name = "bridge"
}
```

## Schema

### Required

- `name` (String) Network name.

### Read-Only

- `dokku_managed` (Boolean)
- `driver` (String)
- `internal` (Boolean)
- `ipv6` (Boolean)
- `labels` (Map of String)
- `network_id` (String)
- `scope` (String)
