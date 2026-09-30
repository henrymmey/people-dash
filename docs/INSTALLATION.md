# People Dash installation on Debian 13 (Trixie)

This is the reference installation for the current home-lab deployment.

## 0. Target layout

You already have:

```text
NPM:
192.168.176.101

Authentik:
192.168.176.102
https://auth.meyerwolke.de
```

Create two Debian 13 (Trixie) LXCs:

```text
People Dashboard:
192.168.176.117
https://p.meyerbrief.de

People Host:
192.168.176.118
https://*.p.meyerbrief.de
ssh.p.meyerbrief.de
```

An unprivileged LXC is preferred. The People Host does not need to be privileged.

### LXC notes

For the dashboard LXC:

- Debian 13 Trixie
- static IP `192.168.176.117`
- enough disk for the Laravel application and PostgreSQL
- normal systemd boot

For the People Host:

- Debian 13 Trixie
- static IP `192.168.176.118`
- disk sized for all People home directories
- normal systemd boot
- do not expose the agent port through the router

Do not put the application source in a user's `public_html`.

---

# Part I – People Dashboard LXC

## 1. Update Debian

Log into:

```text
192.168.176.117
```

as `root`.

Run:

```bash
apt update
apt full-upgrade -y
reboot
```

Reconnect after the reboot.

## 2. Install required packages

Run:

```bash
apt update
apt install -y \
  ca-certificates \
  curl \
  git \
  nginx \
  openssl \
  unzip \
  composer \
  postgresql \
  postgresql-client \
  php8.4-cli \
  php8.4-fpm \
  php8.4-pgsql \
  php8.4-mbstring \
  php8.4-xml \
  php8.4-curl \
  php8.4-zip \
  php8.4-bcmath \
  php8.4-intl \
  php8.4-opcache
```

Check versions:

```bash
php -v
composer --version
psql --version
nginx -v
```

Debian 13 provides PHP 8.4, PostgreSQL 17 and Nginx 1.26 in its normal package repositories. Laravel 13 requires PHP 8.3 or newer, so PHP 8.4 from Debian 13 is suitable.

## 3. Enable services

```bash
systemctl enable --now postgresql
systemctl enable --now php8.4-fpm
systemctl enable --now nginx
```

Check:

```bash
systemctl --failed
systemctl status postgresql --no-pager
systemctl status php8.4-fpm --no-pager
systemctl status nginx --no-pager
```

There should be no failed service.

## 4. Create the PostgreSQL database

Follow `docs/POSTGRESQL.md`.

At the end you must have:

```text
Database: people
Role:     people
Host:     127.0.0.1
Port:     5432
```

## 5. Download the application

```bash
mkdir -p /var/www
cd /var/www
git clone https://github.com/henrymmey/people-dash.git people-dash
cd /var/www/people-dash
```

## 6. Prepare Laravel writable directories

This is important because Composer runs Laravel's post-autoload scripts.

```bash
mkdir -p \
  bootstrap/cache \
  storage/framework/cache \
  storage/framework/sessions \
  storage/framework/views \
  storage/logs
```

## 7. Install PHP dependencies

```bash
composer install --no-dev --optimize-autoloader --no-interaction
```

If a production `composer.lock` is later committed, this command will become fully lockfile-reproducible.

## 8. Create `.env`

```bash
cp .env.example .env
chmod 640 .env
nano /var/www/people-dash/.env
```

Set at least:

```dotenv
APP_ENV=production
APP_DEBUG=false
APP_URL=https://p.meyerbrief.de

DB_CONNECTION=pgsql
DB_HOST=127.0.0.1
DB_PORT=5432
DB_DATABASE=people
DB_USERNAME=people
DB_PASSWORD=YOUR_REAL_POSTGRES_PASSWORD

OIDC_ISSUER=https://auth.meyerwolke.de/application/o/people/
OIDC_CLIENT_ID=YOUR_AUTHENTIK_CLIENT_ID
OIDC_CLIENT_SECRET=YOUR_AUTHENTIK_CLIENT_SECRET
OIDC_REDIRECT_URI=https://p.meyerbrief.de/auth/callback

PEOPLE_AUTHENTIK_GROUP=people-users
PEOPLE_AUTHENTIK_ADMIN_GROUP=people-admins

PEOPLE_PUBLIC_DOMAIN=p.meyerbrief.de
PEOPLE_SSH_HOST=ssh.p.meyerbrief.de

PEOPLE_AGENT_URL=http://192.168.176.118:8080
PEOPLE_AGENT_TOKEN=YOUR_SHARED_AGENT_TOKEN
```

Generate the Laravel key:

```bash
php artisan key:generate --force
```

## 9. Create the shared agent token

On the dashboard LXC:

```bash
openssl rand -hex 32
```

Copy the result.

Use exactly that value for `PEOPLE_AGENT_TOKEN` on the dashboard and later in:

```text
/etc/people-agent/people-agent.env
```

on the People Host.

The token is a credential. Do not commit it.

## 10. Prepare Laravel permissions

The web worker needs write access to:

```text
storage
bootstrap/cache
```

Run:

```bash
chown -R root:root /var/www/people-dash
chown -R www-data:www-data /var/www/people-dash/storage
chown -R www-data:www-data /var/www/people-dash/bootstrap/cache

find /var/www/people-dash -type d -exec chmod 755 {} \;
find /var/www/people-dash -type f -exec chmod 644 {} \;

chmod 775 /var/www/people-dash/storage
chmod 775 /var/www/people-dash/bootstrap/cache
chmod 640 /var/www/people-dash/.env
```

The web worker must not be able to modify application source files.

## 11. Configure Laravel

```bash
cd /var/www/people-dash
php artisan migrate --force
php artisan config:cache
php artisan route:cache
php artisan view:cache
php artisan about
php artisan migrate:status
```

## 12. Configure Nginx on the Dashboard LXC

```bash
rm -f /etc/nginx/sites-enabled/default

ln -sf /var/www/people-dash/infra/dashboard-nginx.conf \
  /etc/nginx/sites-available/people-dash.conf

ln -sf /etc/nginx/sites-available/people-dash.conf \
  /etc/nginx/sites-enabled/people-dash.conf

nginx -t
systemctl reload nginx
```

The config uses:

```text
/run/php/php8.4-fpm.sock
```

Test locally:

```bash
curl -i http://127.0.0.1/up
```

Expected:

```text
HTTP/1.1 200 OK
```

## 13. Trust only NPM as a proxy

The Laravel application is published through NPM.

The app therefore trusts only:

```text
192.168.176.101
```

Do not change this to `*`.

---

# Part II – Authentik

Follow `docs/AUTHENTIK.md`.

You need:

```text
Group:
people-users

Admin group:
people-admins

OIDC application:
People Dash

Slug:
people

Redirect:
https://p.meyerbrief.de/auth/callback
```

Then place the provider's client ID and secret into `.env`.

Before continuing, verify the issuer from Authentik rather than guessing it.

---

# Part III – People Host LXC

## 14. Update Debian

Log into:

```text
192.168.176.118
```

as `root`.

```bash
apt update
apt full-upgrade -y
reboot
```

Reconnect after reboot.

## 15. Install required packages

```bash
apt update
apt install -y \
  acl \
  ca-certificates \
  curl \
  git \
  golang-go \
  nginx \
  openssh-server \
  openssl \
  procps
```

Verify:

```bash
nginx -v
ssh -V
go version
setfacl --version
```

`acl` is required because the web server needs traverse permission on otherwise private home directories.

## 16. Get the source

```bash
mkdir -p /opt/people-dash-src
git clone https://github.com/henrymmey/people-dash.git /opt/people-dash-src
```

## 17. Build the provisioning agent

```bash
cd /opt/people-dash-src/agent

go test ./...
gofmt -d .
go vet ./...

go build -trimpath -ldflags="-s -w" -o /tmp/people-agent .
```

Install:

```bash
mkdir -p /opt/people-agent
install -m 0755 /tmp/people-agent /opt/people-agent/people-agent
```

## 18. Install the agent environment

```bash
install -d -m 0750 /etc/people-agent
cp /opt/people-dash-src/agent/people-agent.env.example /etc/people-agent/people-agent.env
nano /etc/people-agent/people-agent.env
```

Set:

```dotenv
LISTEN=192.168.176.118:8080
PEOPLE_AGENT_TOKEN=THE_EXACT_SAME_TOKEN_AS_THE_DASHBOARD
HOME_ROOT=/home
PUBLIC_DOMAIN=p.meyerbrief.de
SSH_HOST=ssh.p.meyerbrief.de
STATE_ROOT=/var/lib/people-agent/users
SUSPENDED_ROOT=/var/lib/people-agent/suspended
SUSPENDED_SSH_CONFIG=/etc/ssh/sshd_config.d/90-people-suspended.conf
DEFAULT_QUOTA_BYTES=0
```

Protect it:

```bash
chown root:root /etc/people-agent/people-agent.env
chmod 600 /etc/people-agent/people-agent.env
```

## 19. Create state directories

```bash
install -d -o root -g root -m 0755 /var/lib/people-agent
install -d -o root -g root -m 0755 /var/lib/people-agent/users
install -d -o root -g root -m 0755 /var/lib/people-agent/suspended
```

Users must not be able to write here.

## 20. Install systemd service

```bash
cp /opt/people-dash-src/infra/people-agent.service /etc/systemd/system/people-agent.service
systemctl daemon-reload
systemctl enable --now people-agent
```

Check:

```bash
systemctl status people-agent --no-pager
journalctl -u people-agent -n 100 --no-pager
```

## 21. Test the agent

Health:

```bash
curl -fsS http://192.168.176.118:8080/health
```

Expected:

```json
{"status":"ok"}
```

Authenticated request:

```bash
curl -i \
  -H "Authorization: Bearer YOUR_SHARED_AGENT_TOKEN" \
  http://192.168.176.118:8080/v1/users/nope/status
```

Because `nope` does not exist, `404` is expected.

Do not publish port 8080.

---

# Part IV – People Host SSH

## 22. Install SSH baseline

```bash
cp /opt/people-dash-src/infra/people-ssh.conf /etc/ssh/sshd_config.d/20-people.conf
sshd -t
systemctl restart ssh
systemctl status ssh --no-pager
```

The configuration disables password authentication and common forwarding mechanisms.

Do not close your current administrative SSH session until a second session is tested successfully.

## 23. Agent-managed suspension configuration

The agent owns:

```text
/etc/ssh/sshd_config.d/90-people-suspended.conf
```

Do not edit it manually.

It contains either:

```text
# No suspended People users.
```

or:

```text
DenyUsers alice bob henry
```

The agent runs `sshd -t` before reloading SSH.

---

# Part V – People Host Nginx

## 24. Install Nginx

```bash
rm -f /etc/nginx/sites-enabled/default

ln -sf /opt/people-dash-src/infra/people-host-nginx.conf \
  /etc/nginx/sites-available/people-host.conf

ln -sf /etc/nginx/sites-available/people-host.conf \
  /etc/nginx/sites-enabled/people-host.conf

nginx -t
systemctl enable --now nginx
systemctl reload nginx
```

The server uses:

```text
<username>.p.meyerbrief.de
```

as the lookup key and serves:

```text
/home/<username>/public_html
```

No per-user Nginx files are generated.

## 25. Check Nginx locally

```bash
curl -i -H "Host: does-not-exist.p.meyerbrief.de" http://127.0.0.1/
```

An unknown-host rejection is expected; it must not expose an arbitrary directory.

---

# Part VI – DNS and NPM

Follow:

```text
docs/DNS-AND-NETWORK.md
docs/NPM.md
```

The final public routing must be:

```text
p.meyerbrief.de
    -> NPM .101
    -> Dashboard .117

*.p.meyerbrief.de
    -> NPM .101
    -> People Host .118

ssh.p.meyerbrief.de
    -> router TCP/22
    -> People Host .118
```

Router forwarding:

```text
TCP 80  -> 192.168.176.101:80
TCP 443 -> 192.168.176.101:443
TCP 22  -> 192.168.176.118:22
```

Do not forward NPM port 81 or the agent port 8080.

---

# Part VII – First user

## 26. First login

Open:

```text
https://p.meyerbrief.de
```

The successful first login must create:

```text
Dashboard DB record
+
Linux account
+
home directory
+
.ssh
+
public_html
```

On the People Host:

```bash
getent passwd YOUR_USERNAME
ls -la /home/YOUR_USERNAME
ls -la /home/YOUR_USERNAME/public_html
ls -la /home/YOUR_USERNAME/.ssh
ls -l /var/lib/people-agent/users/YOUR_USERNAME
```

## 27. Create a test website

For a simple test:

```bash
cat >/home/YOUR_USERNAME/public_html/index.html <<'EOF'
<!doctype html>
<html>
  <body>
    <h1>People Dash works</h1>
  </body>
</html>
EOF
chown YOUR_USERNAME:YOUR_USERNAME /home/YOUR_USERNAME/public_html/index.html
chmod 644 /home/YOUR_USERNAME/public_html/index.html
```

Open:

```text
https://YOUR_USERNAME.p.meyerbrief.de/
```

---

# Part VIII – SSH key test

## 28. Add a key in the dashboard

Open:

```text
https://p.meyerbrief.de/ssh-keys
```

Paste a public key only.

Example:

```text
ssh-ed25519 AAAA... laptop
```

Then:

```bash
cat /home/YOUR_USERNAME/.ssh/authorized_keys
```

The stored value is the normal public-key line. The global SSH policy in `20-people.conf` disables forwarding/tunneling features.

Check ownership:

```bash
ls -ld /home/YOUR_USERNAME
ls -ld /home/YOUR_USERNAME/.ssh
ls -l /home/YOUR_USERNAME/.ssh/authorized_keys
```

## 29. SSH from a client

```bash
ssh YOUR_USERNAME@ssh.p.meyerbrief.de
```

With a specific key:

```bash
ssh -i ~/.ssh/id_ed25519 YOUR_USERNAME@ssh.p.meyerbrief.de
```

Password login should be disabled.

---

# Part IX – Suspend/resume test

## 30. Suspend

Log in as an admin and open the admin area.

Click:

```text
Suspend
```

Laravel only changes its local state after the agent succeeds.

Then verify:

```bash
ls -l /var/lib/people-agent/suspended/YOUR_USERNAME
cat /etc/ssh/sshd_config.d/90-people-suspended.conf
sshd -t
```

Website:

```bash
curl -I https://YOUR_USERNAME.p.meyerbrief.de/
```

Expected:

```text
HTTP/2 403
```

New SSH login must fail:

```bash
ssh YOUR_USERNAME@ssh.p.meyerbrief.de
```

The agent also terminates existing sessions/processes for the user.

## 31. Resume

Use:

```text
Admin -> Resume
```

Then:

```bash
test ! -e /var/lib/people-agent/suspended/YOUR_USERNAME
cat /etc/ssh/sshd_config.d/90-people-suspended.conf
sshd -t
```

The website and SSH key access should work again.

## 32. What suspend actually does

Suspend is deliberately more than:

```php
$user->update(['status' => 'suspended']);
```

The agent performs:

```text
create suspension marker
+
rebuild DenyUsers
+
sshd -t
+
reload SSH
+
terminate existing sessions
```

The People Host Nginx checks the same suspension marker and returns 403.

If validation or reload fails, the agent restores the previous state and reports failure.

---

# Part X – Production checks

Dashboard:

```bash
php artisan about
php artisan migrate:status
systemctl --failed
nginx -t
```

People Host:

```bash
systemctl --failed
nginx -t
sshd -t
systemctl status people-agent --no-pager
```

Confirm:

```text
APP_DEBUG=false
.env is not in Git
agent token is not in Git
OIDC secret is not in Git
PostgreSQL is local-only
NPM admin port 81 is LAN-only
agent port 8080 is LAN-only
router forwards only 80/443/22
password SSH authentication is disabled
backups exist
```

## Official references

- Debian 13: https://www.debian.org/releases/trixie/
- Laravel deployment: https://laravel.com/docs/13.x/deployment
- Nginx: https://nginx.org/en/docs/
- OpenSSH: https://man.openbsd.org/sshd_config
- Composer: https://getcomposer.org/doc/
