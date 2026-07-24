---
page_title: "dokku_app_nginx Resource - terraform-provider-dokku"
description: |-
  Manages explicit nginx tuning properties for a Dokku application.
---

# dokku_app_nginx (Resource)

Manages commonly used per-app nginx tuning values. It deliberately does not
manage custom nginx template contents; privileged host files belong in host
configuration management.

```terraform
resource "dokku_app_nginx" "uploads" {
  app                  = dokku_app.uploads.name
  client_max_body_size = "128m"
}
```

Import with the application name:

```shell
terraform import dokku_app_nginx.uploads my-app
```

## Argument Reference

- `app` - (Required) Dokku app name.
- `client_max_body_size` - (Optional) Maximum request-body size.
- `proxy_buffer_size` - (Optional) Proxy response-header buffer size.
- `proxy_buffers` - (Optional) Proxy response buffer count and size.
- `proxy_busy_buffers_size` - (Optional) Maximum busy proxy buffer size.
