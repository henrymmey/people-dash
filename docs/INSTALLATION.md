# Installation

This guide assumes the planned Proxmox layout already exists:

- NPM: `192.168.176.101`
- Authentik: `192.168.176.102`
- Dashboard: `192.168.176.117`
- People Host: `192.168.176.118`

Port forwarding is intentionally not specified here; keep it in your existing network documentation.

## 1. Authentik

Create an OAuth2/OpenID provider and application for the dashboard.

Recommended values:

- Client type: confidential
- Redirect URI: `https://p.meyerbrief.de/auth/callback`
- Scopes: `openid`, `profile`, `email`, `groups`
- Allowed group: `people-users`
- Admin group: `people-admins`

The exact Authentik UI labels can change between releases. The important requirement is a standards-compliant OIDC issuer and the two group names used by the dashboard.

Set the issuer in `.env` to the provider's actual issuer URL. Do not guess the issuer path; copy it from Authentik's provider metadata.

## 2. Dashboard LXC

Install PHP 8.3+, PHP-FPM, the PostgreSQL PHP extension, Composer, Git and PostgreSQL client tools. Laravel 13 requires PHP 8.3 or newer.

Clone the repository and install dependencies:

```bash
git clone https://github.com/henrymmey/people-dash /var/www/people-dash
cd /var/www/people-dash
composer install --no-dev --optimize-autoloader
cp .env.example .env
php artisan key:generate
```

Create a PostgreSQL database and role for the dashboard. Put those values into `.env` and set `APP_DEBUG=false`.

Generate a long random value for `PEOPLE_AGENT_TOKEN` and use the exact same value on the People Host. Keep it out of Git.

Then run:

```bash
php artisan migrate --force
php artisan config:cache
php artisan route:cache
```

Set the application directory ownership so PHP-FPM can write only to `storage` and `bootstrap/cache`. The application source should not be writable by the web worker.

Use `infra/dashboard-nginx.conf` as the basis for the local Nginx/PHP-FPM site. NPM should proxy `p.meyerbrief.de` to this local site.

## 3. People Host

Install Nginx, OpenSSH, Go (or build the agent elsewhere) and standard account-management tools such as `useradd` and `userdel`.

Build the agent:

```bash
cd agent
go build -trimpath -ldflags='-s -w' -o people-agent .
install -m 0755 people-agent /opt/people-agent/people-agent
```

Create `/etc/people-agent/people-agent.env` from `agent/people-agent.env.example`. The token must match the dashboard configuration.

Install the systemd unit from `infra/people-agent.service` and enable it.

Configure Nginx from `infra/people-host-nginx.conf`. The server must only serve valid usernames and should reject unknown/malformed hostnames.

Create the wildcard DNS record for `*.p.meyerbrief.de` and point it at the same public entry point used by NPM. Configure NPM to send the wildcard hostname to the People Host.

## 4. SSH

Expose the People Host through your chosen SSH hostname. The dashboard displays:

```text
ssh <username>@ssh.p.meyerbrief.de
```

Use public-key authentication. Password authentication should be disabled once you have verified that key login works.

## 5. Verify

Open `https://p.meyerbrief.de`. The first login should:

1. redirect to Authentik;
2. create a local dashboard record;
3. call the Go agent;
4. create `/home/<username>/public_html`;
5. return to the dashboard.

Create an SSH key in the dashboard, then verify that the public key appears in the user's `authorized_keys` file and that SSH login succeeds.

Create `/home/<username>/public_html/index.html` and verify `https://<username>.p.meyerbrief.de/` serves it.

## Production checklist

- `APP_DEBUG=false`
- strong `APP_KEY`
- PostgreSQL password stored only in `.env`
- strong agent token stored only in environment files
- agent reachable only from `192.168.176.117`
- Nginx denies hidden files and application source on the People Host
- SSH password authentication disabled
- backups exist for PostgreSQL and `/home`
- regular OS security updates
- logs are monitored
