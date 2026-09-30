# Development

## 1. Repository structure

```text
people-dash/
├── agent/
│   ├── go.mod
│   ├── main.go
│   └── main_test.go
├── app/
│   ├── Http/
│   ├── Models/
│   └── Services/
├── config/
├── database/migrations/
├── docs/
├── infra/
├── resources/views/
├── routes/
└── tests/
```

The application and privileged agent are deliberately separate.

## 2. Dashboard requirements

```text
PHP 8.3+
Composer
SQLite or PostgreSQL
Git
```

The production target is Debian 13/PHP 8.4.

## 3. Agent requirements

```text
Go 1.23+
```

The module uses the Go 1.23 language baseline, so newer Go releases can build it.

## 4. Run the Go tests

```bash
cd agent
gofmt -w .
go vet ./...
go test ./...
```

## 5. Run Laravel tests

```bash
composer install
mkdir -p bootstrap/cache
php artisan test
```

The test configuration uses an in-memory SQLite database.

## 6. Local configuration

```bash
cp .env.example .env
php artisan key:generate
```

For local development:

```dotenv
APP_ENV=local
APP_DEBUG=true
APP_URL=http://127.0.0.1:8000
```

Do not reuse production credentials.

## 7. Testing provisioning without root

The agent is a privileged service and should not be exposed from a laptop.

For application development, mock `ProvisioningService` in Laravel tests. The real agent should be integration-tested on a disposable Debian test container.

## 8. Integration test environment

```text
Authentik test instance
        |
        v
People Dashboard test LXC
        |
        v
People Host test LXC
```

Use separate credentials and domains. Never point automated tests at the production People Host.

## 9. Coding rules

### Dashboard

- Validate every external value.
- Keep privileged operations inside `ProvisioningService`.
- Never use `shell_exec()` from HTTP controllers.
- Never store user passwords.
- Escape rendered user values through Blade.
- Keep state transitions explicit.

### Agent

- Never introduce arbitrary command execution.
- Validate usernames before converting them into paths.
- Use fixed command names and argument arrays.
- Check managed-state markers before destructive operations.
- Validate `sshd` configuration before reloading it.
- Fail closed where possible.

## 10. Pull requests

Before opening a PR:

```bash
cd agent
gofmt -w .
go vet ./...
go test ./...
cd ..
php artisan test
```

Also review `infra/`, `.env.example` and `docs/` when changing infrastructure behavior.

## 11. CI

GitHub Actions runs:

```text
Agent:
    gofmt check
    go vet
    go test

Dashboard:
    Composer dependency validation
    Composer dependency installation
    Laravel tests
```

The workflow uses an in-memory SQLite database and needs no production secrets.

## 12. Dependency updates

After dependency changes:

```bash
composer update
composer test
```

Review the dependency changes before committing.

For Go:

```bash
go get -u ./...
go mod tidy
go test ./...
```

## 13. Deployment philosophy

The repository is the source of truth for application code, agent code, Nginx templates, the systemd unit, documentation and CI.

Production secrets and host-generated state are not.

Never commit:

```text
.env
database dumps
private SSH keys
Authentik client secrets
People agent tokens
DNS API tokens
```
