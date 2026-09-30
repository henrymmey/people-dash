# Operations

## Daily health checks

Dashboard:

```bash
systemctl --failed
curl -fsS http://127.0.0.1/up >/dev/null
```

People Host:

```bash
systemctl --failed
nginx -t
sshd -t
systemctl is-active people-agent
```

## Check suspended accounts

```bash
find /var/lib/people-agent/suspended -maxdepth 1 -type f -printf '%f\n' | sort
grep '^DenyUsers' /etc/ssh/sshd_config.d/90-people-suspended.conf
```

They should represent the same set.

## Check managed accounts

```bash
find /var/lib/people-agent/users -maxdepth 1 -type f -printf '%f\n' | sort
getent passwd YOUR_USERNAME
test -d /home/YOUR_USERNAME
```

## Check agent API

```bash
curl -fsS http://192.168.176.118:8080/health
```

Never expose this API externally.

## Logs

Dashboard:

```bash
journalctl -u nginx -n 100 --no-pager
tail -n 100 /var/www/people-dash/storage/logs/laravel.log
```

People Host:

```bash
journalctl -u people-agent -n 100 --no-pager
journalctl -u ssh -n 100 --no-pager
tail -n 100 /var/log/nginx/access.log
tail -n 100 /var/log/nginx/error.log
```

## Updating Dashboard

```bash
cd /var/www/people-dash
git pull --ff-only
composer install --no-dev --optimize-autoloader --no-interaction
php artisan migrate --force
php artisan config:cache
php artisan route:cache
php artisan view:cache
systemctl reload php8.4-fpm
systemctl reload nginx
```

Take a backup before applying database migrations.

## Updating People Agent

```bash
cd /opt/people-dash-src
git pull --ff-only
cd agent
go test ./...
go vet ./...
gofmt -w .
go build -trimpath -ldflags="-s -w" -o /tmp/people-agent .
install -m 0755 /tmp/people-agent /opt/people-agent/people-agent
systemctl restart people-agent
systemctl status people-agent --no-pager
```

## SSH changes

1. validate with `sshd -t`;
2. keep an existing admin SSH session open;
3. test a second session;
4. only then reload/restart SSH.

## Backups before deletion

```bash
sudo -u postgres pg_dump --format=custom --file=/root/people-$(date +%Y%m%d-%H%M%S).dump people
```

Ensure `/home` is included in your normal Proxmox backup plan.

## Disaster recovery

Restore in this order:

```text
1. Proxmox/Debian host
2. PostgreSQL
3. People application
4. People Host /home
5. people-agent state
6. Authentik
7. NPM/TLS
8. DNS
```

Verify a test account before allowing new users.

## User lifecycle repair

If the dashboard says `active` but the Linux account does not exist, inspect the provisioning jobs and agent journal. Do not manually create a second account with a different UID.

If the dashboard says `suspended` but the host marker is missing, inspect the latest `SUSPEND_USER` job and verify agent state before resuming or recreating the user.

## Security incident response

If the agent token is exposed: generate a new token, update both secret files, restart the agent/dashboard as needed, and inspect logs.

If an SSH private key is compromised: revoke its public key in the dashboard, verify it disappeared from `authorized_keys`, and issue a new key.

## Never do this

```text
publish the agent port
share the agent token
run Laravel as root
put private files in public_html
enable PHP execution in public_html
hand users sudo
give users shared write access to /home
manually edit 90-people-suspended.conf
```
