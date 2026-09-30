# Architecture

## Components

```text
Internet
  |
  v
NGINX Proxy Manager .101
  |-----------------------------|
  v                             v
p.meyerbrief.de             *.p.meyerbrief.de
  |                             |
  v                             v
Dashboard .117              People Host .118
  |
  | OIDC
  v
Authentik .102
```

The dashboard talks to the People Host only through the small provisioning API. The dashboard does not access `/home` directly.

## Identity

Authentik remains the identity provider. The dashboard uses OIDC and requests `openid`, `profile`, `email` and `groups`. The stable Authentik `sub` is stored in PostgreSQL and is the primary link between the external account and the People account.

## First login

1. User opens `p.meyerbrief.de`.
2. Dashboard redirects to Authentik.
3. Authentik returns an authorization response.
4. Dashboard checks the People group.
5. Dashboard validates the Authentik username against the Linux username policy.
6. If no local record exists, the dashboard creates a provisioning job.
7. The Go agent creates `/home/<username>`, `.ssh` and `public_html`.
8. The dashboard marks the account active.

## Website flow

A wildcard DNS record sends `*.p.meyerbrief.de` to NPM. NPM sends those requests to the People Host. Nginx on the People Host extracts the first DNS label and serves `/home/<username>/public_html`.

## SSH flow

`ssh <username>@ssh.p.meyerbrief.de` reaches the People Host. OpenSSH performs normal Linux public-key authentication against that user's `authorized_keys` file.

## Security boundaries

- NPM is the public HTTP(S) entry point.
- Authentik owns authentication and group membership.
- The dashboard owns application state.
- The Go agent is the only component allowed to perform privileged account operations.
- Linux permissions isolate users from one another.
- Only public SSH keys are stored.
