---
page_title: "dokku_app_proxy Resource - terraform-provider-dokku"
description: |-
  Manages proxy selection and enabled state for a Dokku application.
---

# dokku_app_proxy (Resource)

Manages whether Dokku's proxy integration is enabled and optionally selects an
explicit proxy implementation.

```terraform
resource "dokku_app_proxy" "worker" {
  app     = dokku_app.worker.name
  enabled = false
}
```

Import with the application name:

```shell
terraform import dokku_app_proxy.worker my-app
```

Deleting the resource clears the app-specific proxy type and enables proxying.

## Argument Reference

- `app` - (Required) Dokku app name.
- `enabled` - (Required) Whether proxy integration is enabled.
- `type` - (Optional) Explicit proxy implementation. Empty inherits the global
  selection.
