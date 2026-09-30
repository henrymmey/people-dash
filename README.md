# People Dash

Self-hosted people-style web hosting dashboard for the planned Meyerwolke/People infrastructure.

## What it does

People Dash lets an Authentik user sign in with OIDC and, on first successful login, automatically provisions a normal Linux account on a dedicated People LXC. The user can then manage SSH public keys and see the website/SSH details for that account.

The website model is intentionally simple:

```text
https://henry.p.meyerbrief.de/
        -> /home/henry/public_html/
```

SSH is equally simple:

```text
ssh henry@ssh.p.meyerbrief.de
```

## Planned infrastructure

| Component | Address | Role |
|---|---|---|
| NGINX Proxy Manager | `192.168.176.101` | Public HTTP(S) reverse proxy |
| Authentik | `192.168.176.102` / `auth.meyerwolke.de` | Identity provider |
| People Dashboard | `192.168.176.117` / `p.meyerbrief.de` | Laravel application |
| People Host | `192.168.176.118` | SSH, Nginx, Linux users and provisioning |

No LDAP is required.

## Repository

```text
.
├── agent/                         Go provisioning agent
├── app/                           Laravel application
├── bootstrap/                     Laravel bootstrap
├── config/                       Laravel configuration
├── database/migrations/           PostgreSQL schema
├── docs/                          Architecture, installation and security
├── infra/                         Nginx/systemd examples
├── resources/views/               Dashboard UI
└── routes/                        HTTP/console routes
```

## Security architecture

The dashboard never gets arbitrary root access. It calls a small Go API on the People Host. That API exposes only fixed operations such as creating a user and adding/removing an SSH public key. The API is protected by a bearer token and must be reachable only from the dashboard LXC.

Auth is handled by Authentik. The dashboard stores the stable OIDC `sub`, not a password. Linux accounts are created only after the first successful People Dashboard login and People-group membership check.

## Current implementation notes

This repository is a strong initial implementation, but it is not a claim that every host-specific hardening step is already complete. In particular, filesystem quotas need to be implemented/enabled at the storage layer, and the current dashboard suspension action is an administrative state rather than a complete SSH/web revocation. Read `docs/PEOPLE-HOST.md` and `docs/SECURITY.md` before exposing the service to untrusted users.

## Getting started

Read these in order:

1. `docs/ARCHITECTURE.md`
2. `docs/INSTALLATION.md`
3. `docs/AUTHENTIK.md`
4. `docs/PEOPLE-HOST.md`
5. `docs/SECURITY.md`

For the Laravel app, use PHP 8.3+, Composer and PostgreSQL. Laravel 13 requires PHP 8.3 or newer.

For the agent, build `agent/` with Go and install the resulting binary as `/opt/people-agent/people-agent`.

Never commit `.env`, OIDC client secrets or the agent token.
