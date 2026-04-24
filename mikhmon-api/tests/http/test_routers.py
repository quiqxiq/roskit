"""
Router management API tests.

Covers: list, get, create, update, delete, test-connection, migrate.
"""
import pytest
import requests
from config import url, TIMEOUT, APIClient, ROUTER_IP, ROUTER_USERNAME, ROUTER_PASSWORD


@pytest.fixture(scope="module")
def client() -> APIClient:
    c = APIClient()
    c.login()
    return c


@pytest.fixture(scope="module")
def created_router(client: APIClient):
    """Create a router for tests that need one, then clean up."""
    payload = {
        "session_name": "test-router-pytest",
        "ip": ROUTER_IP or "192.168.1.1",
        "username": ROUTER_USERNAME or "admin",
        "password": ROUTER_PASSWORD or "admin",
        "hotspot_name": "pytest-hotspot",
        "dns_name": "pytest.local",
        "currency": "IDR",
    }
    resp = client.post("routers", json=payload)
    # May fail with 400/500 if the router is not reachable — skip data-dependent tests
    if resp.status_code not in (200, 201):
        pytest.skip(f"Router creation failed ({resp.status_code}): {resp.text}")
    router = resp.json()["data"]
    yield router
    # Teardown
    client.delete(f"routers/{router['id']}")


# ---------------------------------------------------------------------------
# List routers
# ---------------------------------------------------------------------------

class TestListRouters:
    """GET /routers"""

    def test_list_requires_auth(self):
        resp = requests.get(url("routers"), timeout=TIMEOUT)
        assert resp.status_code == 401

    def test_list_returns_200(self, client: APIClient):
        resp = client.get("routers")
        assert resp.status_code == 200

    def test_list_response_is_array(self, client: APIClient):
        resp = client.get("routers")
        data = resp.json()["data"]
        assert isinstance(data, list)

    def test_list_response_envelope(self, client: APIClient):
        resp = client.get("routers")
        body = resp.json()
        assert "data" in body
        assert "error" in body


# ---------------------------------------------------------------------------
# Get single router
# ---------------------------------------------------------------------------

class TestGetRouter:
    """GET /routers/:routerId"""

    def test_get_nonexistent_router(self, client: APIClient):
        resp = client.get("routers/999999")
        assert resp.status_code == 404

    def test_get_router_returns_fields(self, created_router, client: APIClient):
        rid = created_router["id"]
        resp = client.get(f"routers/{rid}")
        assert resp.status_code == 200
        data = resp.json()["data"]
        for field in ("id", "session_name", "ip", "username"):
            assert field in data, f"Missing field: {field}"

    def test_get_router_password_not_exposed(self, created_router, client: APIClient):
        rid = created_router["id"]
        resp = client.get(f"routers/{rid}")
        data = resp.json()["data"]
        assert "password" not in data


# ---------------------------------------------------------------------------
# Create router
# ---------------------------------------------------------------------------

class TestCreateRouter:
    """POST /routers"""

    def test_create_requires_auth(self):
        resp = requests.post(url("routers"), json={"session_name": "x", "ip": "1.1.1.1", "username": "a", "password": "b"}, timeout=TIMEOUT)
        assert resp.status_code == 401

    def test_create_missing_required_fields(self, client: APIClient):
        resp = client.post("routers", json={"session_name": "test"})
        assert resp.status_code == 400

    def test_create_empty_body(self, client: APIClient):
        resp = client.post("routers", json={})
        assert resp.status_code == 400

    def test_create_router_with_real_credentials(self, client: APIClient):
        """Only runs when ROUTER_IP env var is set."""
        if not ROUTER_IP:
            pytest.skip("ROUTER_IP not set — skipping real connection test")
        payload = {
            "session_name": "pytest-real-router",
            "ip": ROUTER_IP,
            "username": ROUTER_USERNAME,
            "password": ROUTER_PASSWORD,
        }
        resp = client.post("routers", json=payload)
        assert resp.status_code in (200, 201)
        data = resp.json()["data"]
        assert data["ip"] == ROUTER_IP
        # Cleanup
        client.delete(f"routers/{data['id']}")


# ---------------------------------------------------------------------------
# Update router
# ---------------------------------------------------------------------------

class TestUpdateRouter:
    """PUT /routers/:routerId"""

    def test_update_requires_auth(self, created_router):
        rid = created_router["id"]
        resp = requests.put(url(f"routers/{rid}"), json={"currency": "USD"}, timeout=TIMEOUT)
        assert resp.status_code == 401

    def test_update_session_name(self, created_router, client: APIClient):
        rid = created_router["id"]
        resp = client.put(f"routers/{rid}", json={"session_name": "updated-name"})
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert data["session_name"] == "updated-name"

    def test_update_nonexistent_router(self, client: APIClient):
        resp = client.put("routers/999999", json={"session_name": "ghost"})
        assert resp.status_code == 404


# ---------------------------------------------------------------------------
# Delete router
# ---------------------------------------------------------------------------

class TestDeleteRouter:
    """DELETE /routers/:routerId"""

    def test_delete_requires_auth(self):
        resp = requests.delete(url("routers/1"), timeout=TIMEOUT)
        assert resp.status_code == 401

    def test_delete_nonexistent_router(self, client: APIClient):
        resp = client.delete("routers/999999")
        assert resp.status_code == 404

    def test_delete_returns_message(self, client: APIClient):
        """Create a throwaway router and delete it."""
        if not ROUTER_IP:
            pytest.skip("ROUTER_IP not set")
        payload = {
            "session_name": "throwaway-delete-test",
            "ip": ROUTER_IP,
            "username": ROUTER_USERNAME,
            "password": ROUTER_PASSWORD,
        }
        create_resp = client.post("routers", json=payload)
        if create_resp.status_code not in (200, 201):
            pytest.skip("Could not create router for delete test")
        rid = create_resp.json()["data"]["id"]
        del_resp = client.delete(f"routers/{rid}")
        assert del_resp.status_code == 200
        body = del_resp.json()
        assert "data" in body


# ---------------------------------------------------------------------------
# Test connection
# ---------------------------------------------------------------------------

class TestRouterConnection:
    """POST /routers/:routerId/test"""

    def test_test_connection_requires_auth(self, created_router):
        rid = created_router["id"]
        resp = requests.post(url(f"routers/{rid}/test"), json={}, timeout=TIMEOUT)
        assert resp.status_code == 401

    def test_test_connection_unreachable_router(self, client: APIClient):
        """Creating a router with unreachable IP fails at creation (pre-flight test).
        The API may timeout or drop connection — both are acceptable failure modes."""
        import requests as req_lib
        payload = {
            "session_name": "test-unreachable",
            "ip": "10.255.255.1",
            "username": "admin",
            "password": "admin",
        }
        try:
            create_resp = client.session.post(
                url("routers"), json=payload, headers=client.auth_headers(), timeout=30
            )
            # Should fail — unreachable IP triggers connection test failure
            assert create_resp.status_code in (400, 500, 408, 504)
            body = create_resp.json()
            assert body.get("error") is not None
        except (req_lib.exceptions.Timeout, req_lib.exceptions.ConnectionError):
            pass  # Timeout or disconnect is also a valid failure mode

    def test_test_connection_real_router(self, created_router, client: APIClient):
        if not ROUTER_IP:
            pytest.skip("ROUTER_IP not set")
        rid = created_router["id"]
        resp = client.post(
            f"routers/{rid}/test",
            json={"ip": ROUTER_IP, "username": ROUTER_USERNAME, "password": ROUTER_PASSWORD},
        )
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert "connected" in data
        assert "latency_ms" in data
        assert "routeros_version" in data
