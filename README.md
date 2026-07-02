# Gaia Calendar

Gaia Calendar is a self-hosted roster dashboard for Gaia Workforce users. It stores a user's Gaia login, syncs roster data on a schedule, and exposes a private ICS subscription URL that can be added to iPhone Calendar, Google Calendar, Outlook, or any calendar app that supports subscribed calendars.

This project is designed for personal or small-team self-hosting. It does not include any hosted service, company code, employee account, password, API token, or calendar token.

## Features

- Email registration and password reset with Cloudflare Email Routing / Email Sending REST API.
- Gaia credential storage with server-side encryption.
- Scheduled roster sync, manual sync, and sync history.
- Private per-user ICS feed with rotating subscription URL.
- Shift events with clean titles such as `B` or `Manual shift`; time and hours are kept in event details.
- Monthly projected hours and leave balance summary events.
- Dark and light theme.
- English and Traditional Chinese UI.
- SQLite for simple local use, PostgreSQL for production.

## Privacy And Security

- Never commit `.env`, database files, `volumes/`, `data/`, or generated deploy bundles.
- Gaia passwords are encrypted before storage. Set a strong `CREDENTIAL_ENCRYPTION_KEY` before real use.
- Session tokens, reset tokens, verification codes, calendar tokens, and Gaia passwords are not intended to be logged.
- If any real token or password was ever committed or shared, rotate it before using this project publicly.
- The ICS URL is a bearer secret. Anyone with that URL can read the corresponding calendar feed until it is rotated.

## Requirements

- Go 1.26 or newer.
- Bun 1.3 or newer.
- Optional for production: Docker or Podman, PostgreSQL, Caddy or another reverse proxy.

## Quick Start With SQLite

```bash
cp .env.example .env
bun --cwd frontend install
bun --cwd frontend run build
go run ./cmd/server
```

Open `http://localhost:8080`.

The default `DATABASE_URL` in `.env.example` uses SQLite:

```env
DATABASE_URL=sqlite://data/gaia-calendar.db
```

## PostgreSQL

Use PostgreSQL for multi-user or long-running production deployments:

```env
DATABASE_URL=postgres://gaia_calendar:strong-password@localhost:5432/gaia_calendar?sslmode=disable
```

The default Docker Compose file starts PostgreSQL:

```bash
cp .env.example .env
docker compose up -d --build
```

Set these values in `.env` for Compose production use:

```env
POSTGRES_USER=gaia_calendar
POSTGRES_PASSWORD=replace-with-a-strong-password
POSTGRES_DB=gaia_calendar
APP_BASE_URL=https://calendar.example.com
SESSION_SECRET=replace-with-random-secret
CREDENTIAL_ENCRYPTION_KEY=replace-with-random-secret
```

When `APP_BASE_URL` points to a non-localhost domain, the server refuses to start if `SESSION_SECRET` or `CREDENTIAL_ENCRYPTION_KEY` still use the development defaults.

For a single-container SQLite deployment:

```bash
cp .env.example .env
docker compose -f docker-compose.sqlite.yml up -d --build
```

## Email Configuration

Registration and password reset emails require Cloudflare Email Sending compatible credentials:

```env
CLOUDFLARE_EMAIL_ACCOUNT_ID=
CLOUDFLARE_EMAIL_API_TOKEN=
CLOUDFLARE_EMAIL_FROM=no-reply@example.com
```

If email is not configured, registration and password reset requests will fail with an email configuration error. This is intentional so accounts are not created without a usable verification path.

## Gaia Configuration

Users can enter their Gaia company code, employee account, and password in the web UI.

Optionally, set a default company code for your own deployment:

```env
GAIA_DEFAULT_COMPANY_CODE=
```

Leave it empty for an open-source or multi-company deployment.

## Calendar Subscription

After logging in and saving Gaia credentials:

1. Click Sync once, or wait for the scheduled sync.
2. Use the Calendar subscription panel.
3. On iPhone, tap the `webcal://` subscribe button.

The server syncs Gaia in the background. Calendar apps read the ICS feed from the local database, so calendar refreshes do not wait for Gaia requests.

## Local Development

```bash
bun --cwd frontend install
bun --cwd frontend run build
go test ./...
go run ./cmd/server
```

Useful Make targets:

```bash
make frontend
make test
make build
make ent
```

## VPS Deployment

Build and run with Docker Compose:

```bash
mkdir -p /opt/gaia-calendar
cd /opt/gaia-calendar
cp .env.example .env
docker compose up -d --build
```

Bind the app to localhost and put Caddy in front:

```caddyfile
calendar.example.com {
	reverse_proxy 127.0.0.1:8080
}
```

For local-build deployment to a small VPS:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\deploy-local.ps1 -Remote root@example.com -RemoteDir /opt/gaia-calendar
```

## API Overview

- `POST /api/auth/register`
- `POST /api/auth/verify`
- `POST /api/auth/login`
- `POST /api/auth/request-password-reset`
- `POST /api/auth/reset-password`
- `GET /api/me`
- `GET /api/gaia-credential`
- `PUT /api/gaia-credential`
- `POST /api/schedules/sync`
- `GET /api/schedules?month=YYYY-MM`
- `GET /api/calendar-subscription`
- `POST /api/calendar-subscription/rotate`
- `GET /calendar/{token}.ics`

## License

MIT. See `LICENSE`.

---

# Gaia Calendar 繁體中文說明

Gaia Calendar 是一個自架 Gaia Workforce 排班同步工具。它可以保存使用者的 Gaia 登入資料、定時同步排班，並為每位使用者提供獨立的私人 ICS 訂閱 URL，可加入 iPhone Calendar、Google Calendar、Outlook 或其他支援訂閱行事曆的工具。

本專案不包含任何真實公司代碼、員工帳號、密碼、API token 或 calendar token。

## 功能

- Email 註冊、Email 驗證碼、忘記密碼重設連結。
- 使用 Cloudflare Email Sending REST API 寄信。
- Gaia 登入資料會在伺服器端加密保存。
- 手動同步、背景定時同步、同步記錄。
- 每位使用者獨立 ICS 訂閱 URL，可重置。
- 排班事件標題簡潔，例如 `B`、`手工班`，時間和工時放在事件詳情。
- 每月預計總工時與年假餘額摘要。
- 深色 / 淺色主題。
- 英文與繁體中文 UI。
- SQLite 與 PostgreSQL 可選。

## 隱私與安全

- 不要 commit `.env`、資料庫檔案、`volumes/`、`data/` 或部署產物。
- 真正使用前，請設定強度足夠的 `CREDENTIAL_ENCRYPTION_KEY`。
- ICS URL 等同讀取行事曆的 bearer secret，外洩後請立即重置訂閱 URL。
- 如果任何真實 token、密碼、公司代碼或員工資料曾經被公開，請先輪換或更新後再開源。

## SQLite 快速啟動

```bash
cp .env.example .env
bun --cwd frontend install
bun --cwd frontend run build
go run ./cmd/server
```

開啟 `http://localhost:8080`。

## PostgreSQL 正式部署

`.env` 可改成：

```env
DATABASE_URL=postgres://gaia_calendar:strong-password@localhost:5432/gaia_calendar?sslmode=disable
```

Docker Compose 預設使用 PostgreSQL：

```bash
cp .env.example .env
docker compose up -d --build
```

如果想用 SQLite 單容器部署：

```bash
cp .env.example .env
docker compose -f docker-compose.sqlite.yml up -d --build
```

## 設定

必要設定：

```env
APP_BASE_URL=https://calendar.example.com
SESSION_SECRET=replace-with-random-secret
CREDENTIAL_ENCRYPTION_KEY=replace-with-random-secret
CLOUDFLARE_EMAIL_ACCOUNT_ID=
CLOUDFLARE_EMAIL_API_TOKEN=
CLOUDFLARE_EMAIL_FROM=no-reply@example.com
```

`GAIA_DEFAULT_COMPANY_CODE` 可以留空，讓使用者自行輸入公司代碼。

當 `APP_BASE_URL` 是非 localhost 網域時，如果 `SESSION_SECRET` 或 `CREDENTIAL_ENCRYPTION_KEY` 仍是開發預設值，伺服器會拒絕啟動。

## 開發

```bash
bun --cwd frontend install
bun --cwd frontend run build
go test ./...
go run ./cmd/server
```

## 反向代理

Caddy 範例：

```caddyfile
calendar.example.com {
	reverse_proxy 127.0.0.1:8080
}
```
