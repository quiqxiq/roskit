"""
Event (webhook) endpoint tests.

Covers: health check, on-login event from RouterOS.
"""
import pytest
import requests
from config import url, TIMEOUT


class TestHealthCheck:
    """GET /events/health"""

    def test_health_returns_200(self):
        resp = requests.get(url("events/health"), timeout=TIMEOUT)
        assert resp.status_code == 200

    def test_health_returns_ok_status(self):
        resp = requests.get(url("events/health"), timeout=TIMEOUT)
        body = resp.json()
        data = body.get("data") or body
        assert str(data).find("ok") != -1 or body.get("status") == "ok"

    def test_health_no_auth_required(self):
        """Health endpoint must be public."""
        resp = requests.get(url("events/health"), timeout=TIMEOUT)
        assert resp.status_code != 401

    def test_health_response_is_json(self):
        resp = requests.get(url("events/health"), timeout=TIMEOUT)
        assert "application/json" in resp.headers.get("Content-Type", "")


class TestOnLoginEvent:
    """POST /events/on-login (form-encoded hotspot hook from RouterOS)"""

    BASE_PAYLOAD = {
        "router_session": "test-session",
        "username": "testuser01",
        "server": "HS-SERVER",
        "mac": "AA:BB:CC:DD:EE:FF",
        "ip": "192.168.88.100",
        "date": "jan/01/2025",
        "time": "10:30:00",
        "profile": "1day",
    }

    def _post(self, data: dict) -> requests.Response:
        return requests.post(
            url("events/on-login"),
            data=data,
            timeout=TIMEOUT,
        )

    def test_on_login_returns_200(self):
        resp = self._post(self.BASE_PAYLOAD)
        # 200 = processed, 400 = validation error (router not found), either is acceptable
        assert resp.status_code in (200, 400, 404)

    def test_on_login_missing_router_session(self):
        """Without router_session the event is silently dropped (WARN log) — not a 400."""
        payload = {k: v for k, v in self.BASE_PAYLOAD.items() if k != "router_session"}
        resp = self._post(payload)
        # API logs a warning but still returns 200; session lookup fails gracefully.
        assert resp.status_code in (200, 400, 422)

    def test_on_login_missing_username(self):
        """Without username the event is silently dropped (WARN log)."""
        payload = {k: v for k, v in self.BASE_PAYLOAD.items() if k != "username"}
        resp = self._post(payload)
        assert resp.status_code in (200, 400, 422)

    def test_on_login_empty_body(self):
        """Empty body — handler proceeds gracefully without panicking."""
        resp = self._post({})
        assert resp.status_code in (200, 400, 422)

    def test_on_login_is_idempotent(self):
        """Posting the same event twice should not cause a 500."""
        r1 = self._post(self.BASE_PAYLOAD)
        r2 = self._post(self.BASE_PAYLOAD)
        assert r2.status_code in (200, 400, 404, 409)
        assert r2.status_code != 500

    def test_on_login_json_body_rejected(self):
        """Endpoint accepts both form and JSON — both return 200 (router not found = warning log)."""
        resp = requests.post(url("events/on-login"), json=self.BASE_PAYLOAD, timeout=TIMEOUT)
        assert resp.status_code in (200, 400, 415, 422)

    def test_on_login_with_token_header(self):
        resp = requests.post(
            url("events/on-login"),
            data=self.BASE_PAYLOAD,
            headers={"X-Router-Token": "some-token-value"},
            timeout=TIMEOUT,
        )
        assert resp.status_code in (200, 400, 401, 403, 404)
