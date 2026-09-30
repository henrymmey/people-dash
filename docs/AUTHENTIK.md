# Authentik integration

People Dash uses Authentik only as an OIDC identity provider. There is no LDAP integration.

## Provider

Create a confidential OAuth2/OIDC provider for the dashboard. Configure the redirect URI exactly as:

```text
https://p.meyerbrief.de/auth/callback
```

The dashboard requests these scopes:

```text
openid profile email groups
```

The `groups` claim is used to decide whether the account is allowed to use People Dash and whether it is an administrator.

## Groups

Create two Authentik groups:

- `people-users` — permitted to use People Dash
- `people-admins` — permitted to use the admin area

An administrator should normally also be a member of `people-users`.

## Username policy

The Authentik `preferred_username` becomes the Linux username on first provisioning. It must match:

```text
^[a-z_][a-z0-9_-]{0,31}$
```

This prevents path traversal, shell metacharacters and invalid Linux account names from reaching the provisioning layer.

The People username is intentionally immutable after provisioning. If you change the Authentik username later, the existing People account remains linked to its original username.

## Issuer

Set `OIDC_ISSUER` to the issuer shown by Authentik for the provider. The repository's example value is only a placeholder for the expected Authentik URL shape; verify it against your actual provider metadata.

Never put the OIDC client secret in Git.
