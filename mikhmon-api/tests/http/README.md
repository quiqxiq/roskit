# HTTP API Tests

Python-based integration tests for the Mikhmon API.  
Tests run against a **live running server** — not mocks.

---

## File Structure

```
tests/http/
├── config.py                   # Shared URL builder, APIClient, env config
├── conftest.py                 # pytest session-scoped fixtures
├── requirements.txt            # Python dependencies
├── test_auth.py                # Authentication endpoints
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
| `TestSetup` | `POST /auth/setup` |
| `TestLogin` | `POST /auth/login` |
| `TestRefreshToken` | `POST /auth/refresh` |
| `TestLogout` | `POST /auth/logout` |
| `TestMe` | `GET /auth/me` |
| `TestChangePassword` | `PUT /auth/password` |

Scenarios covered:
- Successful login, token shape, token uniqueness
- Wrong password / unknown user → 401
- Missing fields → 400
- Expired / invalid token → 401
- Token blacklisted after logout
- Password change + login with new password (restores original)

### `test_routers.py` — Router Management
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
- 404 for non-existent IDs
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
- Non-existent resource → 404/400

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

All tests in this file are skipped when `ROUTER_ID` is not set.

### `test_events.py` — Webhook Events
| Test class | Endpoints |
|---|---|
| `TestHealthCheck` | `GET /events/health` |
| `TestOnLoginEvent` | `POST /events/on-login` |

Scenarios covered:
- Health is always public (no auth required)
- On-login requires `router_session` + `username`
- Idempotency — posting the same event twice is safe
- JSON body rejected (expects `application/x-www-form-urlencoded`)
- Optional `X-Router-Token` header handling

---

## Setup

### 1. Prerequisites

- Python 3.10+
- The Mikhmon API server running locally (or set `API_BASE_URL`)
- PostgreSQL and Redis running (required by the API)

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

Copy and edit as needed:

```bash
# Required
export API_BASE_URL=http://localhost:8080/api/v1   # default
export TEST_USERNAME=admin                          # default
export TEST_PASSWORD=admin123                       # default

# Optional — needed for router/realtime/hotspot tests
export ROUTER_ID=1              # ID of a router already in the DB
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
$env:TEST_USERNAME = "admin"
$env:TEST_PASSWORD = "admin123"
$env:ROUTER_ID = "1"
```

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

### Only real-time Mikrotik tests

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

## Initial User Setup

If this is a brand-new database with no users, create the first user via:

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/setup \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | python -m json.tool
```

Then set `TEST_USERNAME` and `TEST_PASSWORD` to match.

---

## Notes

- Tests that modify router passwords restore the original password automatically.
- Tests marked with `pytest.skip` are safe to ignore — they need live hardware.
- The `conftest.py` creates a single authenticated session for the whole run; don't delete it.
- The `config.py` `APIClient` stores tokens after `login()` and attaches them to every subsequent request automatically.
