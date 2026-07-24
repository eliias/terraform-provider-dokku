---
page_title: "dokku_app_git Resource - terraform-provider-dokku"
description: |-
  Manages stable Git deployment properties for a Dokku application.
---

# dokku_app_git (Resource)

Manages the accepted deployment branch. The current source image is exposed as
computed pipeline-owned metadata and is never reconciled.

```terraform
resource "dokku_app_git" "web" {
  app           = dokku_app.web.name
  deploy_branch = "main"
}
```

Import with the application name:

```shell
terraform import dokku_app_git.web my-app
```

## Argument Reference

- `app` - (Required) Dokku app name.
- `deploy_branch` - (Required) Accepted Git deployment branch.
- `source_image` - (Computed) Current deployment source image.
