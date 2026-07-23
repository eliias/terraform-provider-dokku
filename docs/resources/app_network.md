---
page_title: "dokku_app_network Resource - terraform-provider-dokku"
subcategory: ""
description: |-
  Manages explicit network properties for a Dokku application.
---

# dokku_app_network (Resource)

Manages the raw, app-specific properties exposed by `dokku network:set`.
Computed fields show the effective values after global inheritance.

Network property changes affect containers created by a subsequent build,
deploy, run, or explicit `dokku ps:rebuild`. This resource deliberately does
not rebuild or restart the app. `dokku network:rebuild` only refreshes listener
metadata and does not reconnect containers. The attachment phases are
supported by Dokku's docker-local scheduler.

## Example Usage

```hcl
resource "dokku_app_network" "example" {
  app = dokku_app.example.name

  attach_post_deploy = [
    dokku_network.backend.name,
  ]
}
```

## Import

Import the settings using the app name:

```shell
terraform import dokku_app_network.example example
```

## Schema

### Required

- `app` (String, ForceNew) Dokku application whose network properties are managed.

### Optional

- `attach_post_create` (Set of String) Networks attached before container startup.
- `attach_post_deploy` (Set of String) Networks attached after a successful deployment and before the proxy update.
- `bind_all_interfaces` (String) `inherit`, `true`, or `false`. When enabled,
  docker-local publishes web container ports as random host ports on
  `0.0.0.0`, making them reachable on host interfaces subject to firewall
  rules.
- `initial_network` (String) Network assigned when containers are created.
- `static_web_listener` (String) Static listener used by proxy integrations, commonly with the null scheduler.
- `tld` (String) Network DNS suffix used for automatic app/process aliases.

The same network cannot be present in both attachment phases. With the
docker-local scheduler, Dokku adds `APP.PROCESS_TYPE` aliases when connecting
deploy containers during a post-create or post-deploy attachment. An initial
network alone does not receive that stable alias in Dokku 0.38.

### Read-Only

- `effective_attach_post_create` (Set of String)
- `effective_attach_post_deploy` (Set of String)
- `effective_bind_all_interfaces` (Boolean)
- `effective_initial_network` (String)
- `effective_tld` (String)
- `web_listeners` (Set of String)
