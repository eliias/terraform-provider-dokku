---
page_title: "dokku_app_scheduler_docker_local Resource - terraform-provider-dokku"
description: |-
  Manages docker-local scheduler properties for a Dokku application.
---

# dokku_app_scheduler_docker_local (Resource)

Manages raw docker-local scheduler settings while reporting their effective
inherited values. Changes affect subsequent scheduling and do not trigger a
deployment or rebuild.

```terraform
resource "dokku_app_scheduler_docker_local" "web" {
  app = dokku_app.web.name
}
```

Import with the application name:

```shell
terraform import dokku_app_scheduler_docker_local.web my-app
```

## Argument Reference

- `app` - (Required) Dokku app name.
- `init_process` - (Optional) Explicit `true` or `false`; empty inherits.
- `parallel_schedule_count` - (Optional) Positive number of process types that
  may be scheduled concurrently; empty inherits.
- `effective_init_process` - (Computed) Effective init-process behavior.
- `effective_parallel_schedule_count` - (Computed) Effective concurrency.
