---
page_title: "dokku_app_checks Resource - terraform-provider-dokku"
description: |-
  Manages app-wide zero-downtime deployment-check state.
---

# dokku_app_checks (Resource)

Manages whether zero-downtime deployment checks are disabled for all process
types. Disabling checks can cause downtime during a future deployment; changing
this resource does not deploy or restart the app.

```terraform
resource "dokku_app_checks" "image_service" {
  app      = dokku_app.image_service.name
  disabled = true
}
```

Import with the application name:

```shell
terraform import dokku_app_checks.image_service my-app
```

## Argument Reference

- `app` - (Required) Dokku app name.
- `disabled` - (Required) Whether checks are disabled for all processes.
- `skipped_processes` - (Computed) Existing process-specific skip state, which
  this resource reports but does not alter.
