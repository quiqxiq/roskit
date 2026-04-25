"""
Real-time Mikrotik data API tests.

Tests endpoints that proxy live data from RouterOS:
- System resources (CPU, RAM, uptime)
- Network interfaces and traffic
- Hotspot active sessions
- DHCP leases
- PPP active sessions
- System logs and clock

All tests require ROUTER_ID env var pointing to a router already in the DB.
If ROUTER_IP is also set, connection/integration tests run as well.

Skip behaviour: tests that need a live router are skipped automatically
when the environment variables are absent.
"""
import pytest
import requests
from config import url, TIMEOUT, APIClient, ROUTER_ID, ROUTER_IP


def require_router_id():
    if not ROUTER_ID:
        pytest.skip("ROUTER_ID not set — skipping live Mikrotik test")


@pytest.fixture(scope="module")
def client() -> APIClient:
    c = APIClient()
    c.login()
    return c


@pytest.fixture(scope="module")
def rid() -> str:
    require_router_id()
    return ROUTER_ID


# ---------------------------------------------------------------------------
# System resources
# ---------------------------------------------------------------------------

class TestSystemResource:
    """GET /routers/:routerId/system/resource"""

    def test_requires_auth(self, rid):
        resp = requests.get(url(f"routers/{rid}/system/resource"), timeout=TIMEOUT)
        assert resp.status_code == 401

    def test_returns_200(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/system/resource")
        assert resp.status_code == 200

    def test_response_has_cpu_and_memory(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/system/resource")
        data = resp.json()["data"]
        assert data is not None
        # RouterOS resource keys (may be nested or flat depending on mapping)
        body = str(data)
        assert any(k in body for k in ("cpu", "free-memory", "uptime", "version"))

    def test_nonexistent_router_returns_error(self, client: APIClient):
        resp = client.get("routers/999999/system/resource")
        assert resp.status_code in (404, 500)


class TestSystemClock:
    """GET /routers/:routerId/system/clock"""

    def test_returns_200(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/system/clock")
        assert resp.status_code == 200

    def test_response_envelope(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/system/clock")
        body = resp.json()
        assert "data" in body
        assert "error" in body


class TestSystemIdentity:
    """GET /routers/:routerId/system/identity"""

    def test_returns_200(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/system/identity")
        assert resp.status_code == 200

    def test_identity_field_present(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/system/identity")
        data = resp.json()["data"]
        assert data is not None


class TestSystemLog:
    """GET /routers/:routerId/system/log"""

    def test_returns_200(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/system/log")
        assert resp.status_code == 200

    def test_returns_list(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/system/log")
        data = resp.json()["data"]
        assert isinstance(data, list)

    def test_limit_query_param(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/system/log", params={"limit": 5})
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert len(data) <= 5

    def test_dashboard(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/system/dashboard")
        assert resp.status_code == 200


class TestSystemRouterboard:
    """GET /routers/:routerId/system/routerboard"""

    def test_returns_200(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/system/routerboard")
        assert resp.status_code == 200


class TestSystemSchedulers:
    """GET /routers/:routerId/system/schedulers"""

    def test_returns_list(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/system/schedulers")
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert isinstance(data, list)


# ---------------------------------------------------------------------------
# Network interfaces
# ---------------------------------------------------------------------------

class TestNetworkInterfaces:
    """GET /routers/:routerId/network/interfaces"""

    def test_requires_auth(self, rid):
        resp = requests.get(url(f"routers/{rid}/network/interfaces"), timeout=TIMEOUT)
        assert resp.status_code == 401

    def test_returns_200(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/network/interfaces")
        assert resp.status_code == 200

    def test_returns_list(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/network/interfaces")
        data = resp.json()["data"]
        assert isinstance(data, list)

    def test_interface_has_name(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/network/interfaces")
        data = resp.json()["data"]
        if data:
            iface = data[0]
            assert "name" in iface or ".id" in iface


class TestNetworkTraffic:
    """GET /routers/:routerId/network/traffic/:iface"""

    def test_requires_auth(self, rid):
        resp = requests.get(url(f"routers/{rid}/network/traffic/ether1"), timeout=TIMEOUT)
        assert resp.status_code == 401

    def test_returns_traffic_for_ether1(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/network/traffic/ether1")
        assert resp.status_code in (200, 404)  # 404 if iface not present on device

    def test_invalid_interface_returns_result(self, client: APIClient, rid):
        """RouterOS returns empty stats for unknown interfaces — not an error."""
        resp = client.get(f"routers/{rid}/network/traffic/nonexistent99")
        assert resp.status_code in (200, 400, 404, 500)


class TestIPPools:
    """GET /routers/:routerId/network/pools"""

    def test_returns_list(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/network/pools")
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert isinstance(data, list)


class TestNATRules:
    """GET /routers/:routerId/network/nat"""

    def test_returns_list(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/network/nat")
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert isinstance(data, list)


class TestParentQueues:
    """GET /routers/:routerId/network/queues"""

    def test_returns_list(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/network/queues")
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert isinstance(data, list)


# ---------------------------------------------------------------------------
# DHCP leases
# ---------------------------------------------------------------------------

class TestDHCPLeases:
    """GET /routers/:routerId/network/dhcp/leases"""

    def test_requires_auth(self, rid):
        resp = requests.get(url(f"routers/{rid}/network/dhcp/leases"), timeout=TIMEOUT)
        assert resp.status_code == 401

    def test_returns_list(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/network/dhcp/leases")
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert isinstance(data, list)

    def test_lease_fields(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/network/dhcp/leases")
        data = resp.json()["data"]
        if data:
            lease = data[0]
            body = str(lease)
            assert any(k in body for k in ("address", "mac-address", "host-name", "status"))

    def test_release_nonexistent_lease(self, client: APIClient, rid):
        resp = client.delete(f"routers/{rid}/network/dhcp/nonexistent_id/release")
        assert resp.status_code in (400, 404, 500)


# ---------------------------------------------------------------------------
# Hotspot active sessions
# ---------------------------------------------------------------------------

class TestHotspotActive:
    """GET /routers/:routerId/hotspot/active"""

    def test_requires_auth(self, rid):
        resp = requests.get(url(f"routers/{rid}/hotspot/active"), timeout=TIMEOUT)
        assert resp.status_code == 401

    def test_returns_list(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/hotspot/active")
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert isinstance(data, list)

    def test_active_session_fields(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/hotspot/active")
        data = resp.json()["data"]
        if data:
            session = data[0]
            body = str(session)
            assert any(k in body for k in ("user", "server", "address", "uptime"))

    def test_disconnect_nonexistent_active(self, client: APIClient, rid):
        resp = client.post(f"routers/{rid}/hotspot/active/nonexistent_id/disconnect")
        assert resp.status_code in (400, 404, 500)

    def test_remove_nonexistent_active(self, client: APIClient, rid):
        resp = client.delete(f"routers/{rid}/hotspot/active/nonexistent_id")
        assert resp.status_code in (400, 404, 500)


# ---------------------------------------------------------------------------
# Hotspot hosts / cookies / servers
# ---------------------------------------------------------------------------

class TestHotspotHosts:
    """GET /routers/:routerId/hotspot/hosts"""

    def test_returns_list(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/hotspot/hosts")
        assert resp.status_code == 200
        assert isinstance(resp.json()["data"], list)


class TestHotspotCookies:
    """GET /routers/:routerId/hotspot/cookies"""

    def test_returns_list(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/hotspot/cookies")
        assert resp.status_code == 200
        assert isinstance(resp.json()["data"], list)


class TestHotspotServers:
    """GET /routers/:routerId/hotspot/servers"""

    def test_returns_list(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/hotspot/servers")
        assert resp.status_code == 200
        assert isinstance(resp.json()["data"], list)


# ---------------------------------------------------------------------------
# PPP active sessions
# ---------------------------------------------------------------------------

class TestPPPActive:
    """GET /routers/:routerId/ppp/active"""

    def test_requires_auth(self, rid):
        resp = requests.get(url(f"routers/{rid}/ppp/active"), timeout=TIMEOUT)
        assert resp.status_code == 401

    def test_returns_list(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/ppp/active")
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert isinstance(data, list)

    def test_disconnect_nonexistent_ppp(self, client: APIClient, rid):
        resp = client.delete(f"routers/{rid}/ppp/active/nonexistent_id")
        assert resp.status_code in (400, 404, 500)


class TestPPPProfiles:
    """GET /routers/:routerId/ppp/profiles"""

    def test_returns_list(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/ppp/profiles")
        assert resp.status_code == 200
        assert isinstance(resp.json()["data"], list)


# ---------------------------------------------------------------------------
# Expire monitor
# ---------------------------------------------------------------------------

class TestExpireMonitor:
    """GET /routers/:routerId/system/expire-monitor"""

    def test_returns_200(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/system/expire-monitor")
        assert resp.status_code == 200
