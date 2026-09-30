# DNS, router forwarding and network layout

This document is specific to the current People Dash deployment.

## 1. Internal addresses

| Component | Internal address | Public name |
|---|---:|---|
| NGINX Proxy Manager | `192.168.176.101` | — |
| Authentik | `192.168.176.102` | `auth.meyerwolke.de` |
| People Dashboard | `192.168.176.117` | `p.meyerbrief.de` |
| People Host | `192.168.176.118` | `*.p.meyerbrief.de` |
| People SSH | `192.168.176.118:22` | `ssh.p.meyerbrief.de` |

## 2. Public DNS records

Create these records at your DNS provider.

```text
p.meyerbrief.de        A       <YOUR_PUBLIC_WAN_IP>
*.p.meyerbrief.de      A       <YOUR_PUBLIC_WAN_IP>
ssh.p.meyerbrief.de    A       <YOUR_PUBLIC_WAN_IP>
```

`p.meyerbrief.de` must be a separate record because a wildcard record does not replace the apex hostname.

`auth.meyerwolke.de` is an existing service and is not changed by this project.

If your public address is dynamic, replace the `A` records with the DNS/DDNS mechanism you already use. The important result is that all three names resolve to the public address that receives inbound connections for this service.

### Optional CNAME alternative

If your DNS provider prefers a stable DDNS hostname, you can use:

```text
p.meyerbrief.de        CNAME   <your-ddns-name>
*.p.meyerbrief.de      CNAME   <your-ddns-name>
ssh.p.meyerbrief.de    CNAME   <your-ddns-name>
```

Follow your DNS provider's rules for apex records; some providers require an ALIAS/ANAME-style record instead of CNAME at the zone apex.

## 3. Router port forwarding

Create exactly these inbound forwards:

| WAN | Protocol | Destination |
|---|---|---|
| `80` | TCP | `192.168.176.101:80` |
| `443` | TCP | `192.168.176.101:443` |
| `22` | TCP | `192.168.176.118:22` |

Do **not** forward:

```text
192.168.176.117:80
192.168.176.118:80
192.168.176.118:8080
192.168.176.101:81
```

Those services are internal.

Port 81 is the NGINX Proxy Manager administrative interface. Keep it LAN-only.

Port 8080 is the privileged People Agent API. It must never be exposed to the public internet.

## 4. Internal network flow

The normal path is:

```text
Internet
   |
   +---- TCP 80/443 ----> NPM .101
   |                         |
   |                         +--> Dashboard .117:80
   |                         |
   |                         +--> People Host .118:80
   |
   +---- TCP 22 ----------> People Host .118:22
```

Application provisioning is independent of the router:

```text
Dashboard .117
      |
      | HTTP API, internal only
      v
People Host .118:8080
```

Authentik communication is:

```text
Dashboard .117
      |
      | HTTPS
      v
auth.meyerwolke.de
```

## 5. NAT loopback / hairpin NAT

Some routers cannot resolve a public hostname back into the LAN correctly when the request originates from inside the LAN.

Symptoms include:

```text
works on mobile data
fails on Wi-Fi
```

or:

```text
p.meyerbrief.de works externally but not internally
```

If that happens, use split DNS on your local resolver.

Recommended internal answers:

```text
p.meyerbrief.de           -> 192.168.176.101
*.p.meyerbrief.de         -> 192.168.176.101
ssh.p.meyerbrief.de       -> 192.168.176.118
auth.meyerwolke.de        -> 192.168.176.101
```

That keeps web traffic going through NPM internally while SSH goes directly to the People Host.

## 6. TLS placement

TLS terminates at NPM.

Internal traffic is intentionally simple:

```text
Browser --HTTPS--> NPM --HTTP--> Dashboard
Browser --HTTPS--> NPM --HTTP--> People Host
```

The dashboard's proxy-trust configuration therefore explicitly trusts only NPM (`192.168.176.101`).

The provisioning agent is a separate internal API and is not proxied by NPM.

## 7. Basic validation

From a LAN machine:

```bash
getent hosts p.meyerbrief.de
getent hosts henry.p.meyerbrief.de
getent hosts ssh.p.meyerbrief.de
```

From the dashboard LXC:

```bash
curl -I http://192.168.176.118
curl -fsS http://192.168.176.118:8080/health
```

From an external network, test:

```bash
curl -I https://p.meyerbrief.de
curl -I https://henry.p.meyerbrief.de
ssh -o PreferredAuthentications=publickey henry@ssh.p.meyerbrief.de
```

Do not use a real username in examples until that account exists.

## 8. If the WAN IP changes

The DNS records must follow the public address.

Do not hard-code a temporary residential WAN IP into the documentation. Use your existing DDNS solution if the address is not static.
