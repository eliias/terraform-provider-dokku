---
page_title: "dokku_app_nginx_custom Resource - terraform-provider-dokku"
description: |-
  Manages custom nginx template selection state for a Dokku application.
---

# dokku_app_nginx_custom (Resource)

Manages whether an app uses custom nginx configuration and which sigil template
path Dokku selects. Template contents remain owned by the application repository
or host configuration management.

```terraform
resource "dokku_app_nginx_custom" "app" {
  app                       = dokku_app.app.name
  nginx_conf_sigil_path     = ".dokku/nginx.conf.sigil"
  disable_custom_config     = "false"
}
```

Empty optional values inherit Dokku's global and built-in defaults. The
`effective_*` attributes expose the resolved values.

Changing this resource persists selection state only. Dokku requires an
explicit `proxy:build-config` operation before an existing generated proxy
configuration changes; the provider does not trigger that potentially
disruptive action.

Import with the application name:

```shell
terraform import dokku_app_nginx_custom.app my-app
```

## Argument Reference

- `app` - (Required) Dokku app name.
- `nginx_conf_sigil_path` - (Optional) Template path relative to the app repository.
- `disable_custom_config` - (Optional) Explicit `true` or `false`; empty inherits.

## Read-Only

- `effective_nginx_conf_sigil_path` - Resolved template path.
- `effective_disable_custom_config` - Resolved custom-config disable state.
