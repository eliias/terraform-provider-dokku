---
page_title: "dokku_app_process Resource - terraform-provider-dokku"
description: |-
  Manages persistent process properties and process scale.
---

# dokku_app_process (Resource)

Manages raw per-app process properties and the exact scale map. Effective
inherited properties and current runtime status are computed separately.

```terraform
resource "dokku_app_process" "web" {
  app = dokku_app.web.name

  scale = {
    web    = 1
    worker = 1
  }
}
```

Changing `scale` invokes `ps:scale` immediately. Import and refresh are
read-only. Property changes do not trigger a rebuild; notably, an updated
restart policy requires a later explicit rebuild to affect existing
containers.

Deleting this resource clears its persistent process-property overrides but
retains the current process scale.

Import with the application name:

```shell
terraform import dokku_app_process.web my-app
```

## Argument Reference

- `app` - (Required) Dokku app name.
- `procfile_path` - (Optional) Explicit Procfile path.
- `restart_policy` - (Optional) Explicit Docker restart policy.
- `restore` - (Optional/Computed) Whether the app is restored after reboot.
- `skip_deploy` - (Optional) Explicit `true` or `false`.
- `start_cmd` - (Optional) Buildpack start-command override.
- `dockerfile_start_cmd` - (Optional) Dockerfile start-command override.
- `stop_timeout_seconds` - (Optional) Positive stop timeout.
- `scale` - (Optional/Computed) Exact process quantities, including zero-count
  process types.

The resource also reports effective property values, `can_scale`, `deployed`,
`running`, and the total `processes` count.
