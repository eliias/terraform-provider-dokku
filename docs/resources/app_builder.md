---
page_title: "dokku_app_builder Resource - terraform-provider-dokku"
description: |-
  Manages explicit builder properties for a Dokku application.
---

# dokku_app_builder (Resource)

Manages persistent builder settings without triggering a build or deployment.
Empty optional values inherit Dokku's global or detected setting.

```terraform
resource "dokku_app_builder" "api" {
  app       = dokku_app.api.name
  build_dir = "services/api"
}
```

Import with the application name:

```shell
terraform import dokku_app_builder.api my-app
```

## Argument Reference

- `app` - (Required) Dokku app name.
- `build_dir` - (Optional) Subdirectory used as the build context.
- `selected` - (Optional) Explicit builder, including `null` or a custom
  installed builder.
- `skip_cleanup` - (Optional) `true` or `false`.
