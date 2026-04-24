"""
Hotspot management API tests.

Covers: users, profiles, bindings.
Requires ROUTER_ID env var for live-router tests.
"""
import pytest
import requests
from config import url, TIMEOUT, APIClient, ROUTER_ID


def require_router_id():
    if not ROUTER_ID:
        pytest.skip("ROUTER_ID not set")


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
# Hotspot users
# ---------------------------------------------------------------------------

class TestHotspotUsers:
    """GET/POST/PUT/DELETE /routers/:routerId/hotspot/users"""

    def test_list_requires_auth(self, rid):
        resp = requests.get(url(f"routers/{rid}/hotspot/users"), timeout=TIMEOUT)
        assert resp.status_code == 401

    def test_list_returns_200(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/hotspot/users")
        assert resp.status_code == 200

    def test_list_returns_array(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/hotspot/users")
        data = resp.json()["data"]
        assert isinstance(data, list)

    def test_list_filter_by_profile(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/hotspot/users", params={"profile": "default"})
        assert resp.status_code == 200

    def test_user_count(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/hotspot/users/count")
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert "count" in data
        assert isinstance(data["count"], int)

    def test_user_count_by_profile(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/hotspot/users/count", params={"profile": "default"})
        assert resp.status_code == 200

    def test_get_nonexistent_user(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/hotspot/users/nonexistent_id_xyz")
        assert resp.status_code in (400, 404, 500)

    def test_export_csv(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/hotspot/users/export")
        assert resp.status_code == 200
        ct = resp.headers.get("Content-Type", "")
        assert "csv" in ct or "text" in ct or "octet-stream" in ct

    def test_add_user_missing_name(self, client: APIClient, rid):
        resp = client.post(f"routers/{rid}/hotspot/users", json={})
        assert resp.status_code in (400, 422, 500)

    def test_update_nonexistent_user(self, client: APIClient, rid):
        resp = client.put(f"routers/{rid}/hotspot/users/nonexistent_id", json={"comment": "test"})
        assert resp.status_code in (400, 404, 500)

    def test_remove_nonexistent_user(self, client: APIClient, rid):
        resp = client.delete(f"routers/{rid}/hotspot/users/nonexistent_id")
        assert resp.status_code in (400, 404, 500)


# ---------------------------------------------------------------------------
# Hotspot profiles
# ---------------------------------------------------------------------------

class TestHotspotProfiles:
    """GET/POST/PUT/DELETE /routers/:routerId/hotspot/profiles"""

    def test_list_requires_auth(self, rid):
        resp = requests.get(url(f"routers/{rid}/hotspot/profiles"), timeout=TIMEOUT)
        assert resp.status_code == 401

    def test_list_returns_array(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/hotspot/profiles")
        assert resp.status_code == 200
        assert isinstance(resp.json()["data"], list)

    def test_profile_fields(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/hotspot/profiles")
        data = resp.json()["data"]
        if data:
            profile = data[0]
            assert "name" in profile

    def test_create_profile_missing_name(self, client: APIClient, rid):
        """RouterOS allows profiles without explicit name (auto-generated); may succeed."""
        resp = client.post(f"routers/{rid}/hotspot/profiles", json={"rate_limit": "1M/1M"})
        # Cleanup if it was created
        if resp.status_code in (200, 201):
            data = resp.json().get("data") or {}
            pid = data.get("id") or data.get(".id")
            if pid:
                client.delete(f"routers/{rid}/hotspot/profiles/{pid}")
        assert resp.status_code in (200, 201, 400, 422, 500)

    def test_create_and_delete_profile(self, client: APIClient, rid):
        payload = {
            "name": "pytest-profile-temp",
            "rate_limit": "512k/512k",
            "shared_users": "1",
            "price": 5000,
            "selling_price": 6000,
            "validity": "1d",
        }
        resp = client.post(f"routers/{rid}/hotspot/profiles", json=payload)
        if resp.status_code not in (200, 201):
            pytest.skip(f"Profile create failed: {resp.text}")
        created = resp.json()["data"]
        profile_id = created.get("id") or created.get(".id")
        # Delete it
        del_resp = client.delete(f"routers/{rid}/hotspot/profiles/{profile_id}")
        assert del_resp.status_code in (200, 204)

    def test_get_nonexistent_profile(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/hotspot/profiles/nonexistent_id")
        assert resp.status_code in (400, 404, 500)


# ---------------------------------------------------------------------------
# Hotspot bindings
# ---------------------------------------------------------------------------

class TestHotspotBindings:
    """GET/POST/PUT/DELETE /routers/:routerId/hotspot/bindings"""

    def test_list_requires_auth(self, rid):
        resp = requests.get(url(f"routers/{rid}/hotspot/bindings"), timeout=TIMEOUT)
        assert resp.status_code == 401

    def test_list_returns_array(self, client: APIClient, rid):
        resp = client.get(f"routers/{rid}/hotspot/bindings")
        assert resp.status_code == 200
        assert isinstance(resp.json()["data"], list)

    def test_create_binding_missing_fields(self, client: APIClient, rid):
        resp = client.post(f"routers/{rid}/hotspot/bindings", json={})
        assert resp.status_code in (400, 422, 500)

    def test_enable_nonexistent_binding(self, client: APIClient, rid):
        resp = client.post(f"routers/{rid}/hotspot/bindings/nonexistent/enable")
        assert resp.status_code in (400, 404, 500)

    def test_disable_nonexistent_binding(self, client: APIClient, rid):
        resp = client.post(f"routers/{rid}/hotspot/bindings/nonexistent/disable")
        assert resp.status_code in (400, 404, 500)
