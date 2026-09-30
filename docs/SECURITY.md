# Security model

## Dashboard

- Run production with `APP_DEBUG=false`.
- Keep `.env` outside Git.
- Use PostgreSQL in production.
- Serve only the `public/` directory from the web server.
- Give PHP-FPM write access to `storage/` and `bootstrap/cache`, not the application source.
- Keep the dashboard-to-agent credential private.

## OIDC

The stable Authentik subject is stored rather than treating the email address as the identity. Group membership is checked before a People account is provisioned.

The dashboard uses an established OpenID Connect client library and HTTPS certificate verification. Do not disable TLS verification in production.

## Provisioning agent

The agent is the critical component. It runs as root but exposes only explicit account-management operations. The bearer token is compared in constant time and must be long and random.

Network policy should allow the agent only from the dashboard LXC. Do not publish the agent port through NPM or a router.

## SSH keys

Only public keys are stored. Keys are validated before they reach `authorized_keys`. Private keys are never generated, uploaded or stored by People Dash.

## User isolation

Each user gets a normal Linux account and home directory. Linux ownership and permissions provide the basic isolation boundary. Never create a shared writable directory containing all user data.

## Website content

User HTML is untrusted content. The People Host should serve it as static content. Do not configure PHP execution inside `public_html` unless a future design explicitly introduces per-user process isolation and resource limits.

## Deletion

The delete operation calls `userdel --remove`. Backups must therefore exist before enabling automated deletion in a production environment.

## Future hardening

Before offering this to untrusted users, add host-level filesystem quotas, a real suspension operation that disables both SSH and web access, rate limiting, audit logging, agent mTLS or another strong service identity, and monitoring/alerting for provisioning failures.
