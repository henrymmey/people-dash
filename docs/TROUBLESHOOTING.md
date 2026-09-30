# Troubleshooting

Use this document before changing the architecture.

## Dashboard does not load

On `192.168.176.117`:

```bash
systemctl status nginx --no-pager
systemctl status php8.4-fpm --no-pager
nginx -t
curl -i http://127.0.0.1/up
```

Expected for `/up` is HTTP 200.

Inspect Laravel logs:

```bash
tail -n 200 /var/www/people-dash/storage/logs/laravel.log
```

Permissions:

```bash
namei -l /var/www/people-dash/public/index.php
ls -ld /var/www/people-dash/storage
ls -ld /var/www/people-dash/bootstrap/cache
```

## Dashboard returns HTTP 500

```bash
cd /var/www/people-dash
php artisan config:clear
php artisan cache:clear
php artisan about
```

Check `.env` keys without printing secrets into tickets:

```bash
grep -E '^(APP_|DB_|OIDC_|PEOPLE_)' /var/www/people-dash/.env
```

Then:

```bash
php artisan config:cache
systemctl reload php8.4-fpm
systemctl reload nginx
```

## OIDC login loops or fails

Verify the Authentik provider and exact redirect URI:

```text
https://p.meyerbrief.de/auth/callback
```

Check in Authentik:

- OAuth2/OIDC provider
- confidential client
- application enabled
- redirect URI exact
- user can access application
- `people-users` access
- `sub` and `preferred_username` claims
- group membership claim

The issuer must be the actual provider issuer.

## User is not a member of the People group

The OIDC response worked but did not contain `people-users` in `groups`. Current Authentik releases include group membership in the standard `profile` scope, but custom mappings can change this. Inspect the provider mappings before creating a custom mapping.

## First login creates DB record but not Linux account

On dashboard:

```bash
journalctl -u nginx --no-pager -n 100
tail -n 200 /var/www/people-dash/storage/logs/laravel.log
curl -fsS http://192.168.176.118:8080/health
curl -i -H "Authorization: Bearer YOUR_AGENT_TOKEN" http://192.168.176.118:8080/v1/users/nope/status
```

A `404` for the nonexistent user means the API route/authentication is working.

On People Host:

```bash
systemctl status people-agent --no-pager
journalctl -u people-agent -n 200 --no-pager
ss -ltnp | grep 8080
```

Verify the shared token matches without exposing it.

## Website returns 404

People Host:

```bash
nginx -t
systemctl status nginx --no-pager
getent passwd YOUR_USERNAME
ls -ld /home/YOUR_USERNAME
ls -ld /home/YOUR_USERNAME/public_html
curl -i -H "Host: YOUR_USERNAME.p.meyerbrief.de" http://127.0.0.1/
```

If local access works but the public hostname does not, investigate NPM or DNS.

## Website returns 403

Check suspension:

```bash
ls -l /var/lib/people-agent/suspended/YOUR_USERNAME
grep YOUR_USERNAME /etc/ssh/sshd_config.d/90-people-suspended.conf
```

If the marker exists, the account is suspended. Resume through the dashboard.

## Website returns 444

The Host header did not match the People hostname regex. Check the hostname and the DNS-safe username policy.

## Website returns 502 from NPM

From NPM:

```bash
curl -I http://192.168.176.118
```

Then People Host:

```bash
systemctl status nginx --no-pager
nginx -t
```

## Dashboard returns 502 from NPM

From NPM:

```bash
curl -I http://192.168.176.117
```

On dashboard:

```bash
systemctl status nginx --no-pager
systemctl status php8.4-fpm --no-pager
nginx -t
```

## SSH does not connect

Client:

```bash
ssh -vvv YOUR_USERNAME@ssh.p.meyerbrief.de
```

People Host:

```bash
systemctl status ssh --no-pager
sshd -t
ss -ltnp | grep ':22'
ls -ld /home/YOUR_USERNAME/.ssh
ls -l /home/YOUR_USERNAME/.ssh/authorized_keys
cat /home/YOUR_USERNAME/.ssh/authorized_keys
test ! -e /var/lib/people-agent/suspended/YOUR_USERNAME
```

## SSH says Permission denied (publickey)

Check client keys:

```bash
ssh-add -L
ssh -i ~/.ssh/id_ed25519 YOUR_USERNAME@ssh.p.meyerbrief.de
```

People Host:

```bash
namei -l /home/YOUR_USERNAME/.ssh/authorized_keys
```

Expected:

```text
home: 0700 + www-data traverse ACL
.ssh: 0700
authorized_keys: 0600
```

## User still has a session after suspend

Check:

```bash
loginctl list-users
ps -u YOUR_USERNAME -f
journalctl -u people-agent -n 200 --no-pager
```

The agent tries `loginctl terminate-user` and falls back to `pkill -KILL -u`.

New SSH connections must still be blocked by `DenyUsers`.

## Agent does not start

```bash
systemctl status people-agent --no-pager
journalctl -u people-agent -n 200 --no-pager
command -v useradd
command -v userdel
command -v setfacl
command -v sshd
command -v systemctl
```

Common causes include a missing token, invalid environment file, missing `setfacl`, an invalid sshd configuration, or permissions under `/var/lib/people-agent`.

## CI fails during Composer

The old failing run failed before tests because Laravel's Composer script could not use `bootstrap/cache`.

The fixed workflow creates writable Laravel directories before `composer install` and installs `pdo_sqlite` for the test environment.

## PostgreSQL connection fails

```bash
systemctl status postgresql --no-pager
ss -ltnp | grep 5432
psql -h 127.0.0.1 -U people -d people -W -c "SELECT 1;"
php artisan migrate:status
```

Do not change `DB_HOST` to the public address.

## DNS works externally but not internally

This is usually NAT loopback. Implement split DNS as described in `docs/DNS-AND-NETWORK.md`.

## NPM certificate does not cover the user website

The certificate must include:

```text
p.meyerbrief.de
*.p.meyerbrief.de
```

A wildcard certificate requires DNS-based validation.

## Where to look first

HTTP:

```text
DNS -> Router -> NPM -> upstream Nginx -> PHP-FPM/Laravel -> Authentik/Agent
```

SSH:

```text
DNS -> Router TCP/22 -> sshd -> DenyUsers -> authorized_keys
```
