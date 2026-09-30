# People Dash – Architecture

People Dash is a small, deliberately boring hosting platform: Authentik handles identity, NGINX Proxy Manager handles the public HTTP(S) entry point, Laravel manages application state, and a small Go service performs the few privileged Linux operations that the web application needs.

## 1. Target infrastructure

```text
                                        INTERNET
                                            |
                              +-------------+-------------+
                              |                           |
                    auth.meyerwolke.de            p.meyerbrief.de
                              |                     *.p.meyerbrief.de
                              |                           |
                              +-------------+-------------+
                                            |
                                            v
                                  NGINX Proxy Manager
                                   192.168.176.101
                                            |
                           +----------------+----------------+
                           |                                 |
                           v                                 v
                 People Dashboard                     People Host
                 192.168.176.117                     192.168.176.118
                 Laravel + PostgreSQL                Nginx + SSH
                           |                         Go agent + /home
                           |                                 ^
                           |                                 |
                           +---------- internal API --------+
```

Authentik already exists at `192.168.176.102` and is published as `https://auth.meyerwolke.de`.

The People Dashboard is published as `https://p.meyerbrief.de`.

Every hosted website is published as:

```text
https://<username>.p.meyerbrief.de/
```

SSH is published as:

```text
ssh <username>@ssh.p.meyerbrief.de
```

## 2. Responsibility boundaries

### Authentik

Authentik is the source of identity and authentication.

It owns:

- username
- email address
- display name
- group membership
- login policy
- the OIDC client

People Dash never stores a password.

### NGINX Proxy Manager

NPM is the public HTTP(S) edge.

It owns:

- TLS certificates
- public DNS host routing
- redirecting HTTP to HTTPS
- forwarding `p.meyerbrief.de` to the dashboard
- forwarding `*.p.meyerbrief.de` to the People Host

It does not know individual People users.

### People Dashboard

The dashboard owns application state.

It stores:

- the stable Authentik OIDC subject
- the immutable People username
- current People account state
- SSH public keys
- provisioning job history

The dashboard never receives root credentials for the People Host.

### People Host

The People Host owns:

- Linux accounts
- home directories
- `public_html`
- SSH
- the People Nginx virtual host
- the Go provisioning agent

The host is intentionally a normal multi-user Linux system. User isolation is provided by Unix ownership/permissions, not by one container per user.

### Provisioning Agent

The agent is the only component that performs privileged account operations.

Supported operations:

- create account
- delete account
- suspend account
- resume account
- add SSH public key
- remove SSH public key
- read storage usage

There is no generic command execution endpoint.

## 3. First login lifecycle

1. A user opens `https://p.meyerbrief.de`.
2. The dashboard redirects to Authentik.
3. Authentik authenticates the user and returns an OIDC authorization response.
4. The dashboard validates the OIDC subject and username.
5. The dashboard checks that the user belongs to `people-users`.
6. If this is a first login, a local People record is created with status `provisioning`.
7. The dashboard calls the agent's `create user` operation.
8. The agent creates the Linux account and its managed state.
9. The dashboard changes the application record to `active`.
10. The user can now add SSH keys and publish files.

If the provisioning operation fails, the local record stays in `provisioning` and the failure is written to the provisioning job log.

## 4. Linux account layout

A managed account looks like this:

```text
/home/
└── henry/
    ├── .ssh/
    │   └── authorized_keys
    └── public_html/
        └── index.html
```

The home directory is private to the user. Nginx receives an ACL entry allowing the web server to traverse the home directory without granting the web server general read access to private files.

`public_html` is intentionally public.

Do not put passwords, private keys, API tokens or other secrets below `public_html`.

## 5. Website request flow

For:

```text
https://henry.p.meyerbrief.de/project/index.html
```

the flow is:

```text
Browser
  |
  | HTTPS
  v
NPM
  |
  | HTTP
  v
People Host / Nginx
  |
  | Host = henry.p.meyerbrief.de
  v
username = henry
  |
  v
/home/henry/public_html/project/index.html
```

The People Nginx configuration uses a safe hostname regex and a variable document root. It is not generated once per user.

Creating a new user therefore does not require an Nginx configuration reload.

Suspension is represented by a root-owned marker under:

```text
/var/lib/people-agent/suspended/<username>
```

Nginx returns HTTP 403 while that marker exists.

## 6. SSH request flow

For:

```text
ssh henry@ssh.p.meyerbrief.de
```

the router forwards TCP/22 to the People Host.

OpenSSH authenticates against:

```text
/home/henry/.ssh/authorized_keys
```

The dashboard only ever handles the public key.

The People Host uses a restrictive global SSH policy. Password authentication, agent forwarding, TCP forwarding, X11 forwarding and tunneling are disabled for People accounts. The provisioning agent stores the supplied public key as a normal `authorized_keys` line.

## 7. Suspend lifecycle

Suspending an account is a host-level operation, not just a dashboard flag.

```text
Admin presses "Suspend"
        |
        v
Dashboard
        |
        v
Provisioning Agent
        |
        +--> create suspension marker
        |
        +--> rebuild sshd DenyUsers list
        |
        +--> sshd -t
        |
        +--> reload ssh
        |
        +--> terminate existing user sessions
        |
        v
Dashboard stores status = suspended
```

After suspension:

- existing SSH sessions are terminated
- new SSH logins are denied by `DenyUsers`
- the website returns HTTP 403
- the home directory is preserved
- the account can be resumed later

If an SSH reload or configuration validation fails, the agent rolls the state back instead of reporting a successful suspension.

## 8. Delete lifecycle

Deletion first suspends the account. Only after SSH access is denied does the agent run:

```text
userdel --remove <username>
```

The agent only deletes accounts that it originally marked as People-managed. It will refuse to delete an arbitrary pre-existing Linux account.

Backups must exist before enabling destructive deletion in production.

## 9. State and trust model

```text
                   untrusted public traffic
                              |
                              v
                          NPM / TLS
                              |
             +----------------+----------------+
             |                                 |
             v                                 v
       Laravel app                         static web
             |                                 |
             | OIDC                           |
             v                                 |
         Authentik                            |
             |                                 |
             +-------------> Linux agent <----+
                                  |
                                  v
                              root operations
```

The most sensitive boundary is the dashboard-to-agent API.

The agent token must be treated as a root-equivalent credential. It must never be committed to Git and the API must never be published through NPM or the router.

## 10. Why there is no LDAP

LDAP is intentionally not part of this design.

Authentik already provides the identity system we need. The dashboard maps the OIDC `sub` to a local application record and creates a local Linux account only when the user first signs in.

This keeps the number of moving parts small while preserving a central authentication source.

## 11. Known scope of version 1

Version 1 provides:

- OIDC login
- first-login Linux provisioning
- static website hosting
- SSH public-key management
- storage usage reporting
- suspend/resume
- destructive account deletion
- admin view
- provisioning audit log

Version 1 does not provide per-user kernel-enforced disk quotas. The storage value shown by the dashboard is usage, not a hard limit. A future quota implementation should be designed around the actual Proxmox storage/filesystem in use rather than pretending that an arbitrary byte count is enforced.
