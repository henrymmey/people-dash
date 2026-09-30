# PostgreSQL setup on Debian 13 (Trixie)

The reference installation keeps PostgreSQL on the People Dashboard LXC (`192.168.176.117`).

Debian 13 provides PostgreSQL as a normal distribution package. Install the distribution version instead of adding a third-party PostgreSQL repository unless you have a separate reason to do so.

## 1. Install PostgreSQL

Run these commands as `root` in the dashboard LXC:

```bash
apt update
apt full-upgrade -y
apt install -y postgresql postgresql-client openssl
systemctl enable --now postgresql
systemctl status postgresql --no-pager
```

Check the installed server:

```bash
su - postgres -c "psql -c 'SELECT version();'"
```

## 2. Generate a database password

Generate a random hexadecimal password. Hex avoids shell quoting problems.

```bash
openssl rand -hex 32
```

Copy the result somewhere safe temporarily.

Example shape:

```text
2c2b4f...64 hexadecimal characters...
```

Do not use the example value.

## 3. Create the role and database

Open PostgreSQL as the `postgres` operating-system account:

```bash
su - postgres -c psql
```

Then run:

```sql
CREATE ROLE people LOGIN PASSWORD 'PASTE_THE_RANDOM_PASSWORD_HERE';
CREATE DATABASE people OWNER people;
\connect people
REVOKE ALL ON SCHEMA public FROM PUBLIC;
GRANT ALL ON SCHEMA public TO people;
\q
```

The `people` role owns the database and is the only application role used by Laravel.

## 4. Verify password authentication

Test the exact connection Laravel will use:

```bash
psql -h 127.0.0.1 -U people -d people -W -c "SELECT current_user, current_database();"
```

Enter the generated password.

Expected result:

```text
 current_user | current_database
--------------+-----------------
 people       | people
```

## 5. PostgreSQL should stay local

The dashboard and PostgreSQL are on the same LXC. The application uses:

```text
DB_HOST=127.0.0.1
DB_PORT=5432
```

Do not expose PostgreSQL to the internet.

Check the listening sockets:

```bash
ss -ltnp | grep 5432
```

For this deployment, a loopback listener is sufficient.

If your PostgreSQL configuration was changed previously, verify:

```bash
su - postgres -c "psql -c \"SHOW listen_addresses;\""
```

If you need to change it, edit the PostgreSQL configuration and keep `listen_addresses` limited to `127.0.0.1`.

## 6. Put the credentials into `.env`

The People Dash `.env` file should contain:

```dotenv
DB_CONNECTION=pgsql
DB_HOST=127.0.0.1
DB_PORT=5432
DB_DATABASE=people
DB_USERNAME=people
DB_PASSWORD=THE_RANDOM_PASSWORD
```

Never put the real password into `.env.example`, GitHub, a screenshot, or a support issue.

## 7. Laravel migrations

After PostgreSQL is configured:

```bash
cd /var/www/people-dash
php artisan migrate --force
```

Verify:

```bash
php artisan migrate:status
```

You should see the four People Dash migrations as `Ran`.

## 8. Backups

At minimum, back up:

```text
people database
/home on the People Host
/etc/people-agent/people-agent.env
/etc/ssh/sshd_config.d/20-people.conf
/etc/ssh/sshd_config.d/90-people-suspended.conf
Nginx configuration
Authentik backups
```

A PostgreSQL dump can be created with:

```bash
sudo -u postgres pg_dump --format=custom --file=/root/people-$(date +%Y%m%d-%H%M%S).dump people
```

On a system where you are already `root`, the same command works exactly as written.

Restore to a test database first. Do not test restore procedures against the live database.

## Official references

- Debian packages: https://packages.debian.org/trixie/postgresql
- PostgreSQL documentation: https://www.postgresql.org/docs/
