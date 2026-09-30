# People Dash

Self-hosted, Authentik-backed People-style web hosting for a Proxmox home lab.

People Dash gives a user a normal Linux account and static web space after their first successful Authentik login:

```text
https://henry.p.meyerbrief.de/
        ->
/home/henry/public_html/
```

SSH:

```text
ssh henry@ssh.p.meyerbrief.de
```

## Architecture

```text
                         Internet
                            |
                            v
                 NGINX Proxy Manager
                  192.168.176.101
                     /        \\
                    /          \\
                   v            v
        p.meyerbrief.de      *.p.meyerbrief.de
              |                     |
              v                     v
     People Dashboard         People Host
      192.168.176.117        192.168.176.118
       Laravel + DB          Nginx + SSH
              |               + Go agent
              |
              v
     Authentik / OIDC
     auth.meyerwolke.de
     192.168.176.102
```

The design deliberately has no LDAP dependency.

## Components

| Component | Address | Responsibility |
|---|---:|---|
| NGINX Proxy Manager | `192.168.176.101` | Public HTTP(S), TLS and routing |
| Authentik | `192.168.176.102` | OIDC identity and groups |
| People Dashboard | `192.168.176.117` | Laravel UI, database and provisioning requests |
| People Host | `192.168.176.118` | Linux users, SSH, static websites and Go agent |

## Features

- Authentik OIDC login
- first-login Linux account provisioning
- immutable People username
- per-user static website at `https://<username>.p.meyerbrief.de`
- SSH public-key management
- storage-usage reporting
- admin account list
- real host-level suspend/resume
- existing SSH session termination on suspend
- provisioning audit trail
- safe managed-account deletion
- Debian 13 (Trixie) installation documentation
- NPM, DNS and router setup documentation
- GitHub Actions CI

## Suspension semantics

Suspend is a real access control operation. It creates a root-owned suspension marker, rebuilds `sshd` `DenyUsers`, validates and reloads SSH, terminates the user's existing sessions/processes, and makes People Nginx return HTTP 403. Laravel changes its database state only after the host operation succeeds.

Resume removes the marker and the generated `DenyUsers` entry.

## Security boundary

The Laravel application never gets arbitrary root shell access. It calls the Go provisioning agent through an authenticated internal API. The agent exposes fixed account operations only.

The agent token is a credential and must never be committed.

## Documentation

Read in this order:

1. [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)
2. [`PROXMOX.md`](PROXMOX.md)
3. [`docs/DNS-AND-NETWORK.md`](docs/DNS-AND-NETWORK.md)
4. [`docs/NPM.md`](docs/NPM.md)
5. [`docs/AUTHENTIK.md`](docs/AUTHENTIK.md)
6. [`docs/INSTALLATION.md`](docs/INSTALLATION.md)
7. [`docs/POSTGRESQL.md`](docs/POSTGRESQL.md)
8. [`docs/PEOPLE-HOST.md`](docs/PEOPLE-HOST.md)
9. [`docs/SECURITY.md`](docs/SECURITY.md)
10. [`docs/OPERATIONS.md`](docs/OPERATIONS.md)
11. [`docs/TROUBLESHOOTING.md`](docs/TROUBLESHOOTING.md)
12. [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md)

## Production network

Router forwards:

```text
TCP 80  -> 192.168.176.101:80
TCP 443 -> 192.168.176.101:443
TCP 22  -> 192.168.176.118:22
```

Do not publish:

```text
192.168.176.101:81
192.168.176.117:80
192.168.176.118:80
192.168.176.118:8080
```

## Runtime requirements

Dashboard: Debian 13, PHP 8.4, PHP-FPM, Composer, PostgreSQL and Nginx.

People Host: Debian 13, Nginx, OpenSSH, Go and ACL tools.

## Development

```bash
cd agent
gofmt -w .
go vet ./...
go test ./...
```

```bash
composer install
mkdir -p bootstrap/cache
php artisan test
```

## Current scope

Version 1 does not enforce per-user kernel disk quotas. It reports actual home-directory usage only.

A hard quota should be implemented only after choosing a storage-specific quota design for the Proxmox environment.
