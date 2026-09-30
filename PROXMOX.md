# Proxmox LXC layout

People Dash uses two dedicated Debian 13 (Trixie) containers.

## 1. Dashboard LXC

```text
CT: People Dashboard
IP: 192.168.176.117
OS: Debian 13 Trixie
Services: nginx, php8.4-fpm, postgresql
```

Recommended: unprivileged container, static IP, normal systemd boot, enough rootfs for PostgreSQL, regular Proxmox backups.

The dashboard container does not require host-level access.

## 2. People Host LXC

```text
CT: People Host
IP: 192.168.176.118
OS: Debian 13 Trixie
Services: nginx, openssh-server, people-agent
```

Recommended: unprivileged container, sufficiently large storage allocation, static IP, regular Proxmox backups.

The People Host does not need Docker, Kubernetes or a privileged LXC.

## 3. Storage

Version 1 stores People home directories below:

```text
/home
```

For a small deployment this can be part of the container rootfs. For a larger deployment, mount a dedicated storage volume into the People Host as `/home`.

Per-user hard quotas require a deliberate design based on the actual filesystem and LXC/storage arrangement. The current project reports usage only.

## 4. Debian template

In the Proxmox UI, download the current Debian 13 standard CT template available for your storage.

Create each container with:

```text
Unprivileged: Yes
Networking: bridge connected to your LAN
IPv4: static
Gateway: your LAN gateway
DNS: your normal resolver
```

Give the People Host enough disk for all hosted users.

## 5. Proxmox firewall

Defense-in-depth is recommended. The intent is:

```text
Internet -> NPM .101
NPM .101 -> Dashboard .117:80
NPM .101 -> People Host .118:80
Internet -> People Host .118:22
Dashboard .117 -> People Host .118:8080
Dashboard .117 -> Authentik / HTTPS
```

The agent port must never be allowed from the internet.

## 6. Snapshots and backups

Use Proxmox snapshots before major changes, but do not treat snapshots as your only backup.

At minimum, back up the Dashboard LXC, People Host LXC, PostgreSQL, `/home`, the agent environment, NPM configuration and Authentik data.

Test restores periodically.
