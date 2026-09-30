# People Host operations

The People Host is the most privileged machine in this project. It contains all user homes and the provisioning agent.

## Filesystem

```text
/home/
  alice/
    .ssh/
      authorized_keys
    public_html/
  bob/
    .ssh/
    public_html/
```

Each home belongs to its Linux user. Users should not be members of a shared group that grants access to other homes.

`public_html` is intentionally readable by the web server. Do not put private material there.

## Nginx

The repository's `infra/people-host-nginx.conf` uses a server-name regex. The first DNS label becomes the Linux username. Nginx then uses `/home/<username>/public_html` as its root.

This is deliberately not generated per user. Adding a user therefore does not require an Nginx reload.

## Provisioning agent

The agent runs as root because Linux account and ownership changes require it. It is protected by a bearer token and should listen only on the private address.

The agent has no endpoint for arbitrary shell execution. Its supported operations are:

- create user
- delete user
- add SSH public key
- remove SSH public key
- report storage usage

Do not add an endpoint that accepts arbitrary commands.

## Quotas

The current agent reports `DEFAULT_QUOTA_BYTES` as the advertised quota but does not enforce a kernel filesystem quota. For a real quota, configure filesystem/project quotas on the storage backing `/home` and make the provisioning workflow set the user's quota. This is intentionally left as a host-specific step because Proxmox LXC storage and the underlying filesystem determine which quota mechanism is appropriate.

Do not advertise a storage limit to users until the host actually enforces it.

## Backups

Back up at least PostgreSQL, `/home`, `/etc/people-agent/people-agent.env`, Nginx configuration and your existing Authentik backup set. The agent token is a credential and must be stored securely.

## User lifecycle

The dashboard uses these states:

- `provisioning` — database record exists but initial host provisioning is incomplete
- `active` — normal access
- `suspended` — administrative state; current code does not yet disable SSH/web access on the host
- `deleted` — host account has been removed

Before offering suspension as a security control, extend the agent with explicit disable/enable operations.
