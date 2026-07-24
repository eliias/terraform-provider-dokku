# Dokku feature support

This document tracks the provider against Dokku 0.38's declarative
configuration surface. It distinguishes persistent configuration, which is a
good fit for Terraform, from imperative operations such as deployments and
backups.

Status meanings:

- **Supported**: represented by a tested resource or attribute.
- **Partial**: useful coverage exists, but Dokku exposes additional persistent
  settings.
- **Planned**: persistent state that should be represented by the provider.
- **Action**: imperative behavior that should not be modeled as continuously
  reconciled state.
- **Host-level**: belongs in host configuration management rather than this
  provider.

The primary references are the official
[Dokku documentation](https://dokku.com/docs~v0.38.25/), the
[Dokku source tree](https://github.com/dokku/dokku/tree/v0.38.25), and the
official Dokku datastore plugin repositories.

## Safety and modeling principles

- Reads must distinguish raw app properties from inherited effective values.
- Import must adopt existing state without restarting, rebuilding, or
  reconnecting an application.
- Resources must not trigger deployments unless that behavior is intrinsic to
  the underlying persistent setting and clearly documented.
- Secrets may be managed when explicitly configured, but unrelated
  externally-managed config variables must be preserved.
- Destructive datastore lifecycle operations require explicit Terraform
  replacement or destruction.
- Privileged host files, packages, firewall rules, and custom nginx files are
  host configuration and belong in Ansible.

## Current support

| Area | Status | Current coverage |
| --- | --- | --- |
| Provider connection | Supported | SSH keys, inline keys, SSH agents, ordinary SSH users with a command prefix, host-version checks |
| Applications | Partial | Create, import, rename, destroy, lock state read, config variables, domains, ports, buildpacks |
| App resource limits | Partial | App-wide and per-process CPU, memory, swap, network, ingress, egress, and NVIDIA GPU limits |
| Storage | Supported | Global entries and app mounts with phases, process type, read-only mode, subpaths, chown, and volume options |
| App networks | Supported | Initial network, post-create and post-deploy attachments, interface binding, TLD, static listener, inherited effective values |
| App Docker options | Supported | Build, deploy, and run options with preservation prefixes for integration-owned values |
| Builder settings | Partial | Builder selection, build directory, and cleanup behavior |
| Proxy settings | Partial | Proxy enabled state and explicit proxy selection |
| nginx settings | Partial | Request body size and common proxy buffer settings; bind addresses remain on the app resource |
| Global networks | Supported | Global initial network, attachment phases, interface binding, and TLD |
| Docker networks | Supported | Managed network lifecycle and metadata data source for managed or external networks |
| PostgreSQL | Partial | Service lifecycle, image/version, stopped state, exposure, links, aliases, query strings, creation limits, and network phases |
| Redis | Partial | Service lifecycle, image/version, stopped state, exposure, links, aliases, query strings, creation limits, and network phases |
| MySQL | Partial | Service lifecycle, image/version, stopped state, exposure, links, aliases, query strings, creation limits, and network phases |
| MariaDB | Partial | Service lifecycle, image/version, stopped state, exposure, links, aliases, query strings, creation limits, and network phases |
| ClickHouse | Partial | Basic lifecycle, stopped state, links, and network phases |

## Planned declarative coverage

### Provider foundations

| Feature | Priority | Notes |
| --- | --- | --- |
| Capability data source | P0 | Report Dokku version, installed/enabled plugin versions, builders, schedulers, and proxies so resources can fail with actionable compatibility errors |
| Structured report parsing | P0 | Shared JSON/report helpers with compatibility aliases for supported Dokku releases |
| App and service data sources | P1 | Read existing objects without taking ownership |
| Consistent import verification | P1 | Acceptance coverage for every resource |

### Applications and deployment configuration

| Feature | Priority | Notes |
| --- | --- | --- |
| App creation/global properties | P0 | Disable autocreation, deploy locking, and other persistent `apps` properties |
| Global and per-app config | P0 | Preserve inheritance and distinguish raw from effective values |
| Global and per-app domains | P0 | Enabled state, global domains, app domains, and inherited values |
| Remaining builder properties | P1 | Dockerfile path, herokuish allowance, and builder manifest paths |
| Scheduler selection | P0 | docker-local, k3s, null, and custom schedulers |
| docker-local scheduler properties | P0 | Init process and parallel schedule count |
| Process scaling | P0 | Desired scale by process type without treating deploy status as configuration |
| Process properties | P0 | Procfile path, restart policy, skip-deploy, start command, stop timeout |
| Resource reservations | P0 | Reservations parallel to the existing limit model |
| Deployment checks | P0 | Disabled/skipped checks and wait-to-retire |
| `app.json` | P1 | Manifest path and supported persistent app-json behavior |
| Cron definitions | P1 | Declarative scheduled commands |
| Git/repository properties | P1 | Deploy branch, archive limits, keep-git-dir, source image, and related persistent settings |
| Buildpack properties | P1 | Stack and complete ordered buildpack management |
| Registry properties | P1 | Image repository/template, server, push-on-release, and extra tags |

### Routing, TLS, and web server configuration

| Feature | Priority | Notes |
| --- | --- | --- |
| Remaining generic proxy properties | P1 | Explicit HTTP and HTTPS property management independent of port mappings |
| Remaining nginx properties | P1 | Complete `nginx:report` coverage with raw/effective separation |
| Custom nginx config state | P0 | Detect and safely manage the selected config path; host file contents remain Ansible-owned |
| HTTP authentication | P1 | Enabled state, users through sensitive input, and allowed IPs |
| Redirect plugin | P1 | Persistent redirect definitions |
| Certificates | P1 | Certificate metadata and explicitly managed certificate material |
| Let's Encrypt | P1 | Active/autorenew state, email, server, DNS provider, grace period, and lego arguments |
| Other proxy integrations | P2 | Persistent Caddy, HAProxy, OpenResty, and Traefik properties when selected |

### Datastores

| Feature | Priority | Notes |
| --- | --- | --- |
| Persistent service properties | P0 | Config options and non-secret custom environment values where plugins expose readable state |
| Complete image/upgrade settings | P1 | Locked state and upgrade-time properties without silently upgrading imports |
| Backup configuration | P1 | Authentication, bucket, schedule, encryption, and retention state |
| Service data sources | P1 | Metadata excluding credentials by default |
| Additional official plugins | P2 | Add typed resources when a production use case or stable plugin contract exists |

### Operations and administration

| Feature | Classification | Notes |
| --- | --- | --- |
| Deploy, rebuild, restart, stop/start actions | Action | Run explicitly outside normal refresh/apply reconciliation |
| Database backup, restore, clone, export, import | Action | Some settings are declarative; executions are not |
| One-off `run` commands and tasks | Action | Use CI/CD or an explicit action mechanism |
| Logs and event streams | Action | Better exposed as diagnostics or data sources than managed resources |
| SSH keys and Dokku plugin installation | P2 | Host-wide state with high blast radius; support only with strong safeguards |
| Dokku installation, packages, UFW, fail2ban | Host-level | Managed by Ansible in the infrastructure repository |
| Privileged custom nginx file contents | Host-level | Managed and validated by Ansible; provider selects or reports them |

## Delivery order

1. Add the capability/report foundation.
2. Implement the P0 features required to import current production apps
   exactly.
3. Import every app, service, network, mount, and persistent plugin property.
4. Add the remaining P1 declarative features with tests and documentation.
5. Evaluate P2 features individually against their blast radius and stable
   upstream interfaces.
6. Keep imperative actions and host-level configuration outside normal
   Terraform reconciliation.
