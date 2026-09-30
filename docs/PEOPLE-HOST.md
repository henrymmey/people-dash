# People Host operations

This document describes what the People Host does after installation.

Host:

```text
192.168.176.118
```

## 1. Managed account definition

A People-managed Linux account must have all of these:

```text
Linux account exists
/home/<username> exists
/var/lib/people-agent/users/<username> exists
/home/<username>/public_html exists
/home/<username>/.ssh exists
```

The state marker is important. The agent will not delete a normal Linux account that just happens to have a People-compatible username.

## 2. User permissions

A typical user home is:

```text
drwx------ <username> <username> /home/<username>
```

plus an ACL entry that allows only the web server user to traverse the directory.

Inspect it with:

```bash
getfacl /home/<username>
```

You should see an entry similar to:

```text
user:www-data:--x
```

The user can access their own files normally. Other People users do not get a shared read permission on another user's private files.

## 3. public_html

The web root is:

```text
/home/<username>/public_html
```

Recommended content:

```text
HTML
CSS
JavaScript
images
static assets
```

Do not put private keys, .env files, database dumps, API credentials or server configuration there.

PHP execution is not configured in `public_html`.

## 4. Nginx behavior

The People Nginx server uses:

```text
<username>.p.meyerbrief.de
```

as the first DNS label. That label must satisfy the People username policy.

Nginx does:

```text
Host:
henry.p.meyerbrief.de

            |
            v

people_user = henry

            |
            v

root = /home/henry/public_html
```

Nginx also refuses hidden files. The configuration uses `disable_symlinks if_not_owner` to reduce cross-user symlink tricks.

## 5. Suspension markers

A suspended account has:

```text
/var/lib/people-agent/suspended/<username>
```

The marker is root-owned.

While it exists:

```text
People Nginx -> HTTP 403
sshd -> DenyUsers <username>
```

The marker directory is not writable by People users.

## 6. SSH suspension file

The agent maintains:

```text
/etc/ssh/sshd_config.d/90-people-suspended.conf
```

Do not modify it manually.

The agent:

1. reads all suspension marker filenames;
2. validates their usernames;
3. sorts them;
4. writes a fresh `DenyUsers` line;
5. runs `sshd -t`;
6. reloads the SSH service.

This avoids editing an existing configuration line in-place.

## 7. Existing sessions

Reloading `sshd` does not terminate already-established sessions.

Therefore the agent also terminates processes for the suspended Linux user.

It first tries:

```bash
loginctl terminate-user <username>
```

and falls back to:

```bash
pkill -KILL -u <username>
```

This is intentionally aggressive: a suspended hosting account should not keep running user processes.

## 8. SSH policy

The provisioning agent stores the user's public key as a normal `authorized_keys` line. The SSH daemon enforces the hosting policy globally through `infra/people-ssh.conf`: password authentication, agent forwarding, TCP forwarding, X11 forwarding and tunnels are disabled.

Do not loosen the SSH policy for People users without reviewing the security implications first.

## 9. User creation

The agent creates accounts with:

```text
useradd
--create-home
--user-group
--groups people
--home-dir /home/<username>
--shell /bin/bash
```

Afterwards it creates `public_html` and `.ssh` and applies the web-server traverse ACL.

## 10. User deletion

Deletion only works for managed accounts.

The sequence is:

```text
suspend
  -> verify sshd config
  -> reload ssh
  -> terminate processes
  -> userdel --remove
  -> remove People state
  -> rebuild DenyUsers
```

If `userdel` fails, the account remains suspended.

## 11. Storage reporting

Current version reports the size of `/home/<username>` using:

```bash
du -sb /home/<username>
```

A quota value of zero means no per-user quota is enforced.

The dashboard therefore displays actual usage and does not pretend that an arbitrary limit exists.

## 12. Backups

Back up:

```text
/home
/var/lib/people-agent
/etc/people-agent
/etc/ssh/sshd_config.d
/etc/nginx
```

before destructive maintenance.

## 13. Updating the agent

```bash
cd /opt/people-dash-src/agent
git pull --ff-only
go test ./...
go vet ./...
gofmt -w .
go build -trimpath -ldflags="-s -w" -o /tmp/people-agent .
install -m 0755 /tmp/people-agent /opt/people-agent/people-agent
systemctl restart people-agent
systemctl status people-agent --no-pager
```

The service rebuilds the suspension SSH config when it starts.

## 14. Logs

```bash
journalctl -u people-agent -n 200 --no-pager
journalctl -u people-agent -f
tail -f /var/log/nginx/access.log
tail -f /var/log/nginx/error.log
journalctl -u ssh -n 200 --no-pager
```
