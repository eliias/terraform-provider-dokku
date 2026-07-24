---
page_title: "dokku_app_scheduler Resource - terraform-provider-dokku"
description: |-
  Manages generic scheduler properties for a Dokku application.
---

# dokku_app_scheduler (Resource)

Manages raw per-app scheduler selection and shell properties while reporting
their effective inherited values. Empty arguments preserve global and built-in
inheritance.

```terraform
resource "dokku_app_scheduler" "web" {
  app = dokku_app.web.name
}
```

The effective scheduler is available as `effective_selected`. Import with the
application name:

```shell
terraform import dokku_app_scheduler.web my-app
```

## Argument Reference

- `app` - (Required) Dokku app name.
- `selected` - (Optional) Explicit scheduler integration, such as
  `docker-local`, `null`, `k3s`, or a custom installed scheduler.
- `shell` - (Optional) Explicit shell used by `dokku run` and `dokku enter`.
- `effective_selected` - (Computed) Scheduler after inheritance.
- `effective_shell` - (Computed) Shell after inheritance.
