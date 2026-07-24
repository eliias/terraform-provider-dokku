---
page_title: "dokku_app_docker_options Resource - terraform-provider-dokku"
description: |-
  Manages phase-scoped Docker options for a Dokku application.
---

# dokku_app_docker_options (Resource)

Manages the complete ordered list of Docker options for each Dokku container
phase. Changes affect containers created by a later deploy, rebuild, or run;
this resource does not restart or rebuild the application.

Each set item is passed to Dokku as one option entry. Preserve that grouping
when importing legacy configuration. Dokku canonicalizes entry order.

```terraform
resource "dokku_app_docker_options" "mail" {
  app = dokku_app.mail.name

  deploy = [
    "-p 25:25",
    "-p 587:587",
  ]
}
```

Import an application's Docker options using its app name:

```shell
terraform import dokku_app_docker_options.mail mail
```

Deleting this resource clears the build, deploy, and run Docker options without
restarting the application.

## Argument Reference

- `app` - (Required) Dokku app name.
- `build` - (Optional) Docker options for build containers.
- `deploy` - (Optional) Docker options for deploy containers.
- `run` - (Optional) Docker options for one-off run containers.
