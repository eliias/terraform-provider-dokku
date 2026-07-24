---
page_title: "dokku_capabilities Data Source - terraform-provider-dokku"
description: |-
  Reports the Dokku version and enabled host capabilities.
---

# dokku_capabilities (Data Source)

Reports the configured host's Dokku version and enabled plugins without taking
ownership or changing host state.

```terraform
data "dokku_capabilities" "host" {}
```

## Attribute Reference

- `dokku_version` - Reported Dokku version.
- `tested_version` - Whether the version is in the provider's tested range.
- `plugins` - Enabled plugin names, versions, core status, descriptions, and
  source URLs.
- `builders` - Enabled builder integration names.
- `schedulers` - Enabled scheduler integration names.
- `proxies` - Enabled proxy integration names.
