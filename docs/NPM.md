# NGINX Proxy Manager setup

This assumes NPM is already running at:

```text
192.168.176.101
```

The NPM admin UI is normally:

```text
http://192.168.176.101:81
```

Keep this interface LAN-only.

## 1. Dashboard proxy host

Create a new Proxy Host.

### Details

```text
Domain Names:
p.meyerbrief.de

Scheme:
http

Forward Hostname / IP:
192.168.176.117

Forward Port:
80
```

Enable:

```text
Block Common Exploits: ON
Websockets Support: ON
```

WebSocket support is not required by the current dashboard, but enabling it is harmless and leaves room for future dashboard features.

Do not use an Access List that blocks normal authenticated users.

## 2. Wildcard People proxy host

Create a second Proxy Host.

### Details

```text
Domain Names:
*.p.meyerbrief.de

Scheme:
http

Forward Hostname / IP:
192.168.176.118

Forward Port:
80
```

Enable:

```text
Block Common Exploits: ON
Websockets Support: ON
```

Do not create one Proxy Host per People user. The People Host Nginx performs the username-to-home-directory mapping.

## 3. TLS certificate

The dashboard apex and wildcard are different TLS names.

Request one certificate containing both:

```text
p.meyerbrief.de
*.p.meyerbrief.de
```

A wildcard certificate requires a DNS-01 validation flow. Configure the DNS provider credentials in NPM's certificate dialog.

Do not put DNS provider API tokens into this Git repository.

Once the certificate is issued, select the same certificate for both Proxy Hosts.

Enable:

```text
Force SSL: ON
HTTP/2 Support: ON
```

Only add HSTS after you have verified that HTTPS is working correctly from external networks.

## 4. DNS challenge

In current NPM versions the exact UI wording can differ slightly, but the required values are always:

```text
Certificate provider:
Let's Encrypt

Domains:
p.meyerbrief.de
*.p.meyerbrief.de

Validation:
DNS Challenge

DNS provider:
your actual DNS provider

DNS credentials:
provider-specific API credential
```

The API credential should be scoped to DNS editing for this zone only when your DNS provider supports scoped tokens.

## 5. Do not publish the NPM admin panel

The router forwards only:

```text
80 -> 192.168.176.101:80
443 -> 192.168.176.101:443
```

Do not create:

```text
81 -> 192.168.176.101:81
```

from the internet.

Access port 81 from your LAN or management network only.

## 6. Expected routing

After setup:

```text
https://p.meyerbrief.de
    -> NPM
    -> 192.168.176.117:80

https://anything.p.meyerbrief.de
    -> NPM
    -> 192.168.176.118:80
```

SSH never goes through NPM:

```text
ssh <user>@ssh.p.meyerbrief.de
    -> router TCP/22
    -> 192.168.176.118:22
```

## 7. Troubleshooting NPM

### Dashboard returns 502

Check from NPM:

```bash
curl -I http://192.168.176.117
```

Then on the dashboard LXC:

```bash
systemctl status nginx --no-pager
systemctl status php8.4-fpm --no-pager
nginx -t
```

### People sites return 502

Check from NPM:

```bash
curl -I http://192.168.176.118
```

Then on People Host:

```bash
systemctl status nginx --no-pager
nginx -t
```

### Wildcard certificate fails

Check that:

- `p.meyerbrief.de` and `*.p.meyerbrief.de` are present in the requested certificate
- your DNS API credential can edit the zone
- stale TXT validation records are not interfering
- the DNS provider actually supports the challenge flow configured in NPM

## References

- NGINX Proxy Manager: https://nginxproxymanager.com/
- Let's Encrypt challenge types: https://letsencrypt.org/docs/challenge-types/
