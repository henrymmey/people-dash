# Security model and hardening

People Dash deliberately gives one service root privileges. The strategy is to minimize what that service can be asked to do and keep every privileged interface private.

## 1. Threat model

Treat these as untrusted:

- People website content
- SSH users
- uploaded public keys
- Host headers from the internet
- all HTTP request parameters

Highly sensitive:

- Authentik client secret
- PostgreSQL password
- People Agent bearer token
- `/home`
- `/var/lib/people-agent`
- SSH configuration

## 2. Authentication

People Dash does not implement passwords. Authentication is:

```text
Browser -> People Dash -> Authentik OIDC -> People Dash local session
```

The stable OIDC `sub` is stored. Email is not used as the account identity.

## 3. Group authorization

The dashboard requires `people-users` before provisioning. Admin access additionally requires `people-admins`. The Authentik application is also recommended to be bound to `people-users`.

This gives two authorization boundaries:

```text
Authentik application binding
        +
Laravel group claim check
```

## 4. Username security

People hostnames are built directly from the People username. The policy is:

```text
one to 32 characters
first character: a-z
remaining characters: a-z, 0-9, -
```

Examples: `alice`, `henry`, `henry1`, `web-user`.

Rejected: `Alice`, `a_b`, `-alice`, `alice-`, `../../etc`.

This is more restrictive than the full set of possible Linux usernames because the same value is also a DNS label.

## 5. Proxy trust

Laravel trusts only:

```text
192.168.176.101
```

as a reverse proxy. Do not change this to all networks.

## 6. Dashboard filesystem

The Nginx web root is:

```text
/var/www/people-dash/public
```

The source code is not exposed directly. The web worker writes only to `storage` and `bootstrap/cache`.

## 7. Provisioning agent

The agent runs as root because account management requires it. Its API has fixed routes, bearer authentication, strict username validation, managed-account markers, bounded request bodies and no arbitrary shell endpoint.

Never add an endpoint such as:

```text
POST /execute
{
  "command": "..."
}
```

## 8. Agent network exposure

The agent listens on:

```text
192.168.176.118:8080
```

and should be reachable only from:

```text
192.168.176.117
```

Never expose it through NPM, the internet or the router.

## 9. SSH

The People Host disables:

```text
PasswordAuthentication
KbdInteractiveAuthentication
agent forwarding
TCP port forwarding
X11 forwarding
tunneling
user-provided environment
```

People public keys are stored as normal `authorized_keys` lines. The global SSH policy prevents forwarding and tunneling, keeping the host suitable for interactive shell/SFTP use without turning it into a jump host.

## 10. Existing-session termination

Suspension terminates running processes for the user because an sshd reload alone cannot end established sessions.

## 11. Website isolation

The People Host serves user content as static files. There is no PHP execution inside `public_html`. Nginx rejects hidden files and uses `disable_symlinks if_not_owner`.

Users must not be placed into a shared Unix group that grants read access to other home directories.

## 12. Secrets

Never commit `.env`, OIDC client secrets, DB passwords, the agent token or DNS provider API credentials.

## 13. Suspension

Suspension is successful only after:

```text
marker created
+
sshd config rebuilt
+
sshd -t succeeds
+
ssh reload succeeds
+
existing user sessions are terminated
```

The dashboard changes its database state after the agent reports success. If the host operation fails, Laravel leaves the user active and the job records the failure.

## 14. Deletion

Delete is destructive. It invokes `userdel --remove` after first suspending the account. The agent checks its managed marker before deletion so unrelated system accounts cannot be removed through this API.

## 15. Resource limits

Version 1 does not enforce per-user kernel disk quotas. Do not advertise a hard quota until one is actually configured.

For a future quota implementation, choose the mechanism from the actual Proxmox storage/filesystem rather than hard-coding an assumption.

## 16. Updates

Keep current and review changes before major upgrades:

```text
Debian 13 packages
Laravel dependencies
PHP
Nginx
OpenSSH
Go
Authentik
NPM
```

## References

- OpenSSH `sshd_config`: https://man.openbsd.org/sshd_config
- Nginx `disable_symlinks`: https://nginx.org/en/docs/http/ngx_http_core_module.html#disable_symlinks
- Laravel deployment: https://laravel.com/docs/13.x/deployment
