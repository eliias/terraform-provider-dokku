---
page_title: "dokku_network_global Resource - terraform-provider-dokku"
subcategory: ""
description: |-
  Manages global Dokku network defaults.
---

# dokku_network_global (Resource)

Manages global network properties inherited by applications without explicit
app-level overrides. Global changes affect every inheriting app on its next
build, deploy, or run.

```hcl
resource "dokku_network_global" "defaults" {
  bind_all_interfaces = "false"
}
```

Import the singleton using `global`:

```shell
terraform import dokku_network_global.defaults global
```

## Schema

### Optional

- `attach_post_create` (Set of String)
- `attach_post_deploy` (Set of String)
- `bind_all_interfaces` (String) `inherit`, `true`, or `false`. Enabling this
  publishes random web-container ports on host `0.0.0.0`.
- `initial_network` (String)
- `tld` (String)

### Read-Only

- `effective_attach_post_create` (Set of String)
- `effective_attach_post_deploy` (Set of String)
- `effective_bind_all_interfaces` (Boolean)
- `effective_initial_network` (String)
- `effective_tld` (String)
