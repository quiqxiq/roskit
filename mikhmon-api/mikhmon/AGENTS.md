# Mikhmon v4 — Workspace AGENTS.md

## Workspace Structure

| Directory | Purpose |
|---|---|
| `mikhmon-api/` | Go backend (Gin + GORM + PostgreSQL + Redis). **Active development.** Has its own `AGENTS.md` — read it first. |
| (other dirs) | Legacy PHP monolith — source of truth for domain logic until Go backend is feature-complete. |

## Domain Reference

Full domain analysis: `ANALYSIS.md`. Key rules any agent in this workspace must follow:

- **RouterOS is source of truth** for all network data (users, profiles, sessions, queues, interfaces)
- Backend DB stores **ONLY**: voucher sales, router configs, system users, audit logs, templates
- **NEVER create GORM models for**: `hotspot_users`, `queues`, `interfaces`, `active_sessions`

### Encryption Schemes (PHP migration)

| Scheme | Used For | Algorithm |
|---|---|---|
| `enc_rypt`/`dec_rypt` | Router passwords | XOR key `"128"` cycled, then base64 |
| `blah()`/`unblah()` | Credential transport | `base64decode → XOR 10 → double base64decode` |
| `jsEncode` (key=25) | Response obfuscation | **Do NOT replicate** — use HTTPS+JSON |

### User Comment Formats (DO NOT change — frontend parses these)

- Expiry: `mmm/dd/yyyy hh:mm:ss N` or `mmm/dd/yyyy hh:mm:ss X` (21 chars)
- Voucher: `vc-{code}-{MM.DD.YY}-{text}` or `up-{code}-{MM.DD.YY}-{text}`
- Frontend parses via position checks at chars 3, 6, 14, 17

### On-Login Expiration Modes

| Mode | Action | Sales Record? |
|---|---|---|
| `ntf` | `limit-uptime=1s` | No |
| `ntfc` | `limit-uptime=1s` | Yes (`/system/script`) |
| `rem` | Delete user | No |
| `remc` | Delete user | Yes (`/system/script`) |
| `0` + price | None | No |

### Config.php Migration Target

Delimiters: `!` (IP), `@|@` (user), `#|#` (password), `%` (hotspot), `^` (DNS), `&` (currency), `*` (phone), `(` (email), `)` (info LP), `=` (idle timeout), `@!@` (report mode), `#!#` (token). Migrate to PostgreSQL with AES-encrypted passwords.

### Migration Phases

1. Keep RouterOS expire-monitor scheduler
2. Add Go backend expiry job
3. Replace on-login with HTTP POST to `/api/v1/events/on-login`
4. Remove RouterOS scheduler

### Hotspot User Count Off-by-One

PHP returns `count - 1` to exclude admin user. Replicate this in Go.
