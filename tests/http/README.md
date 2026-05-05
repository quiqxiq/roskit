# HTTP API Tests

Python-based integration tests for the Roskit API.
Tests run against a **live running server** — not mocks.

> Roskit is multi-tenant + Casbin RBAC. Most tests run inside one tenant context. See [Multi-Tenancy](#multi-tenancy) below.

---

## File Structure

```
tests/http/
├── config.py                   # Shared URL builder, APIClient, env config
├── conftest.py                 # pytest session-scoped fixtures
├── requirements.txt            # Python dependencies
├── test_auth.py                # Authentication endpoints (login, refresh, etc.)
├── test_routers.py             # Router CRUD + connection test
├── test_hotspot.py             # Hotspot users, profiles, bindings
├── test_mikrotik_realtime.py   # Live RouterOS data (system, network, PPP, DHCP)
└── test_events.py              # Webhook / on-login event endpoint
```

---

## What Is Tested

### `test_auth.py` — Authentication
| Test class | Endpoints |
|---|---|
| `TestSetup` | `POST /auth/setup` (one-time tenant + owner bootstrap) |
| `TestLogin` | `POST /auth/login` (tenant + username + password) |
| `TestRefreshToken` | `POST /auth/refresh` |
| `TestLogout` | `POST /auth/logout` |
| `TestMe` | `GET /auth/me` |
| `TestChangePassword` | `PUT /auth/password` |

Scenarios covered:
- Successful login, token shape, token uniqueness
- Wrong password / unknown user / unknown tenant → 401
- Login with suspended tenant → 403
- Missing `tenant` field for non-superadmin → 401
- Expired / invalid token → 401
- Token blacklisted after logout
- Password change + login with new password (restores original)

### `test_routers.py` — Router Management (tenant-scoped)
| Test class | Endpoints |
|---|---|
| `TestListRouters` | `GET /routers` |
| `TestGetRouter` | `GET /routers/:id` |
| `TestCreateRouter` | `POST /routers` |
| `TestUpdateRouter` | `PUT /routers/:id` |
| `TestDeleteRouter` | `DELETE /routers/:id` |
| `TestRouterConnection` | `POST /routers/:id/test` |

Scenarios covered:
- Auth guard on every endpoint
- Casbin enforcement: `staff` cannot create/delete routers (403)
- Tenant isolation: a router from another tenant returns 404
- Validation errors on missing required fields
- Password not exposed in response
- Real connection test (skipped if `ROUTER_IP` not set)

### `test_hotspot.py` — Hotspot Management
| Test class | Endpoints |
|---|---|
| `TestHotspotUsers` | `GET/POST/PUT/DELETE /routers/:id/hotspot/users` |
| `TestHotspotProfiles` | `GET/POST/PUT/DELETE /routers/:id/hotspot/profiles` |
| `TestHotspotBindings` | `GET/POST/PUT/DELETE /routers/:id/hotspot/bindings` |

Scenarios covered:
- List returns array
- Count endpoint with optional profile filter
- CSV export with correct Content-Type
- Create profile + delete lifecycle
- Missing required fields → 400
- Cross-tenant router ID → 404

### `test_mikrotik_realtime.py` — Live RouterOS Data
| Test class | Endpoints |
|---|---|
| `TestSystemResource` | `GET /routers/:id/system/resource` |
| `TestSystemClock` | `GET /routers/:id/system/clock` |
| `TestSystemIdentity` | `GET /routers/:id/system/identity` |
| `TestSystemLog` | `GET /routers/:id/system/log` |
| `TestSystemRouterboard` | `GET /routers/:id/system/routerboard` |
| `TestSystemSchedulers` | `GET /routers/:id/system/schedulers` |
| `TestNetworkInterfaces` | `GET /routers/:id/network/interfaces` |
| `TestNetworkTraffic` | `GET /routers/:id/network/traffic/:iface` |
| `TestIPPools` | `GET /routers/:id/network/pools` |
| `TestNATRules` | `GET /routers/:id/network/nat` |
| `TestParentQueues` | `GET /routers/:id/network/queues` |
| `TestDHCPLeases` | `GET /routers/:id/network/dhcp/leases` |
| `TestHotspotActive` | `GET /routers/:id/hotspot/active` |
| `TestHotspotHosts` | `GET /routers/:id/hotspot/hosts` |
| `TestHotspotCookies` | `GET /routers/:id/hotspot/cookies` |
| `TestHotspotServers` | `GET /routers/:id/hotspot/servers` |
| `TestPPPActive` | `GET /routers/:id/ppp/active` |
| `TestPPPProfiles` | `GET /routers/:id/ppp/profiles` |
| `TestExpireMonitor` | `GET /routers/:id/system/expire-monitor` |

Skipped when `ROUTER_ID` is not set.

### `test_events.py` — Webhook Events (public, tenant resolved by webhook_token)
| Test class | Endpoints |
|---|---|
| `TestHealthCheck` | `GET /events/health` |
| `TestOnLoginEvent` | `POST /events/on-login` |

Scenarios covered:
- Health is always public (no auth required)
- On-login requires `router_name` + `username` + `token` (or `X-Router-Token` header)
- Tenant resolution: invalid token → 200 silent reject (security)
- Idempotency — posting the same event twice is safe
- JSON body rejected (expects `application/x-www-form-urlencoded`)

---

## Multi-Tenancy

Tests run as a single tenant user (default role: `owner`). The `conftest.py` session login uses:

```json
POST /auth/login
{
  "tenant":   "<TEST_TENANT>",
  "username": "<TEST_USERNAME>",
  "password": "<TEST_PASSWORD>"
}
```

To test cross-tenant isolation or superadmin flows, run the suite multiple times with different env vars, or extend `config.py` with helper fixtures.

To test platform admin endpoints (`/admin/tenants/*`, `/admin/templates/*`), log in as a superadmin user (created manually or via `cmd/seed`) and **omit** `TEST_TENANT`. Pass `X-Tenant-Slug: <slug>` header on requests that need a target tenant.

---

## Setup

### 1. Prerequisites

- Python 3.10+
- The Roskit API server running locally (or set `API_BASE_URL`)
- PostgreSQL and Redis running (required by the API)
- A bootstrapped tenant + owner — see Initial Setup below

### 2. Install Dependencies

```bash
cd tests/http
pip install -r requirements.txt
```

Or with a virtual environment:

```bash
python -m venv .venv
source .venv/bin/activate      # Windows: .venv\Scripts\activate
pip install -r requirements.txt
```

### 3. Environment Variables

```bash
# Required
export API_BASE_URL=http://localhost:8080/api/v1   # default
export TEST_TENANT=my-hotspot                       # tenant slug (omit for superadmin)
export TEST_USERNAME=owner                          # user within the tenant
export TEST_PASSWORD=owner1234

# Optional — needed for router/realtime/hotspot tests
export ROUTER_ID=1              # ID of a router already in the tenant
export ROUTER_IP=192.168.88.1   # RouterOS IP address
export ROUTER_USERNAME=admin    # RouterOS username
export ROUTER_PASSWORD=secret   # RouterOS password

# Optional
export API_TIMEOUT=10           # seconds, default: 10
export TEST_NEW_PASSWORD=newpassword123  # used in change-password test
```

On Windows (PowerShell):

```powershell
$env:API_BASE_URL = "http://localhost:8080/api/v1"
$env:TEST_TENANT = "my-hotspot"
$env:TEST_USERNAME = "owner"
$env:TEST_PASSWORD = "owner1234"
$env:ROUTER_ID = "1"
```

> If `config.py` has not yet been updated to read `TEST_TENANT`, add the field manually to `APIClient.login()` payload, or upgrade `config.py` to follow the new login schema.

---

## Running Tests

### All tests
```bash
cd tests/http
pytest -v
```

### Only auth tests
```bash
pytest test_auth.py -v
```

### Only real-time MikroTik tests
```bash
ROUTER_ID=1 pytest test_mikrotik_realtime.py -v
```

### Skip slow/live tests (auth + events only)
```bash
pytest test_auth.py test_events.py -v
```

### Parallel execution (faster)
```bash
pytest -v -n auto
```

### Stop on first failure
```bash
pytest -v -x
```

### Show only failures
```bash
pytest -v --tb=short -q
```

---

## Initial Setup

For a brand-new database, bootstrap the first tenant + owner:

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/setup \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_name": "My Hotspot",
    "tenant_slug": "my-hotspot",
    "username":    "owner",
    "password":    "owner1234"
  }' | python -m json.tool
```

Then set:
```bash
export TEST_TENANT=my-hotspot
export TEST_USERNAME=owner
export TEST_PASSWORD=owner1234
```

The `setup` endpoint refuses requests once any user exists in the database. To re-bootstrap, drop the schema and re-run `make migrate-up`.

---

## Notes

- Tests that modify router passwords restore the original password automatically.
- Tests marked with `pytest.skip` are safe to ignore — they need live hardware.
- The `conftest.py` creates a single authenticated session for the whole run; do not delete it.
- The `config.py` `APIClient` stores tokens after `login()` and attaches them to every subsequent request automatically.
- Casbin enforcement is route-pattern based. If a test fails with 403, double-check the role of the test user against the policies in `internal/casbin/policies.go`.
- Router IDs from another tenant return **404** (not 403) by design — `RouterTenantMiddleware` makes them appear non-existent to prevent enumeration.
