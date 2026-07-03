# Gaia Demo Values

Use these placeholder values when writing docs, screenshots, demos, or test deployments.
They are intentionally fake and should not connect to any real Gaia tenant.

## Example Gaia Credential Form

| Field | Example value |
| --- | --- |
| Company code | `DEMOCO` |
| Employee account | `E000001` |
| Gaia password | `demo-password-not-real` |

## Example Deployment Values

| Field | Example value |
| --- | --- |
| Public URL | `https://calendar.example.com` |
| Sender email | `no-reply@example.com` |
| SQLite database | `sqlite://data/gaia-calendar.db` |
| PostgreSQL database | `postgres://gaia_calendar:strong-password@127.0.0.1:5432/gaia_calendar?sslmode=disable` |

## Notes

- Real roster sync requires a user's own Gaia company code, employee account, and password.
- Registration and password reset require working Cloudflare email credentials.
- Never use a real company code, employee number, password, API token, or calendar token in public screenshots, issue reports, examples, or commits.

---

# Gaia 示範資料

以下資料只用於文件、截圖、demo 或測試部署。它們是刻意設計的假資料，不應連到任何真實 Gaia tenant。

## Gaia 登入資料示範

| 欄位 | 範例值 |
| --- | --- |
| 公司代碼 | `DEMOCO` |
| 員工帳號 | `E000001` |
| Gaia 密碼 | `demo-password-not-real` |

## 部署資料示範

| 欄位 | 範例值 |
| --- | --- |
| 公開網址 | `https://calendar.example.com` |
| 寄件地址 | `no-reply@example.com` |
| SQLite 資料庫 | `sqlite://data/gaia-calendar.db` |
| PostgreSQL 資料庫 | `postgres://gaia_calendar:strong-password@127.0.0.1:5432/gaia_calendar?sslmode=disable` |

## 注意事項

- 真正同步排班需要使用者自己的 Gaia 公司代碼、員工帳號和密碼。
- 註冊和忘記密碼需要有效的 Cloudflare email 設定。
- 公開截圖、issue、範例或 commit 中不要放真實公司代碼、員工號、密碼、API token 或 calendar token。
