"""
Authentication API tests.

Covers: setup, login, refresh token, logout, /me, change password.
"""
import pytest
import requests
from config import url, bearer, TIMEOUT, TEST_USERNAME, TEST_PASSWORD, TEST_NEW_PASSWORD


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def login(username=TEST_USERNAME, password=TEST_PASSWORD) -> dict:
    resp = requests.post(
        url("auth/login"),
        json={"username": username, "password": password},
        timeout=TIMEOUT,
    )
    return resp


def do_login() -> dict:
    """Return parsed data block from a successful login."""
    resp = login()
    assert resp.status_code == 200, f"login failed: {resp.text}"
    return resp.json()["data"]


# ---------------------------------------------------------------------------
# Setup endpoint
# ---------------------------------------------------------------------------

class TestSetup:
    """POST /auth/setup — only works when no users exist."""

    def test_setup_already_done_returns_error(self):
        """After initial setup the endpoint must be locked."""
        resp = requests.post(
            url("auth/setup"),
            json={"username": "newadmin", "password": "password123"},
            timeout=TIMEOUT,
        )
        # Expect 409 Conflict or 403 once a user already exists
        assert resp.status_code in (400, 403, 409), (
            f"Expected setup to be disabled, got {resp.status_code}: {resp.text}"
        )

    def test_setup_validates_short_username(self):
        resp = requests.post(
            url("auth/setup"),
            json={"username": "ab", "password": "password123"},
            timeout=TIMEOUT,
        )
        assert resp.status_code in (400, 403, 409)

    def test_setup_validates_short_password(self):
        resp = requests.post(
            url("auth/setup"),
            json={"username": "validuser", "password": "12345"},
            timeout=TIMEOUT,
        )
        assert resp.status_code in (400, 403, 409)


# ---------------------------------------------------------------------------
# Login
# ---------------------------------------------------------------------------

class TestLogin:
    """POST /auth/login"""

    def test_login_success(self):
        data = do_login()
        assert "access_token" in data
        assert "refresh_token" in data
        assert "expires_in" in data
        assert "user" in data
        user = data["user"]
        assert "id" in user
        assert "username" in user
        assert "role" in user

    def test_login_returns_200(self):
        resp = login()
        assert resp.status_code == 200

    def test_login_wrong_password(self):
        resp = login(password="wrongpassword")
        assert resp.status_code == 401

    def test_login_unknown_user(self):
        resp = login(username="ghost_user_xyz")
        assert resp.status_code == 401

    def test_login_missing_username(self):
        resp = requests.post(
            url("auth/login"),
            json={"password": TEST_PASSWORD},
            timeout=TIMEOUT,
        )
        assert resp.status_code == 400

    def test_login_missing_password(self):
        resp = requests.post(
            url("auth/login"),
            json={"username": TEST_USERNAME},
            timeout=TIMEOUT,
        )
        assert resp.status_code == 400

    def test_login_empty_body(self):
        resp = requests.post(url("auth/login"), json={}, timeout=TIMEOUT)
        assert resp.status_code == 400

    def test_login_response_envelope(self):
        resp = login()
        body = resp.json()
        assert "data" in body
        assert "error" in body

    def test_access_token_is_non_empty_string(self):
        data = do_login()
        token = data["access_token"]
        assert isinstance(token, str) and len(token) > 20

    def test_refresh_token_differs_from_access_token(self):
        data = do_login()
        assert data["access_token"] != data["refresh_token"]

    def test_expires_in_positive_integer(self):
        data = do_login()
        assert isinstance(data["expires_in"], int)
        assert data["expires_in"] > 0


# ---------------------------------------------------------------------------
# Refresh token
# ---------------------------------------------------------------------------

class TestRefreshToken:
    """POST /auth/refresh"""

    def test_refresh_returns_new_tokens(self):
        data = do_login()
        old_access = data["access_token"]
        refresh_token = data["refresh_token"]

        resp = requests.post(
            url("auth/refresh"),
            json={"refresh_token": refresh_token},
            timeout=TIMEOUT,
        )
        assert resp.status_code == 200
        new_data = resp.json()["data"]
        assert "access_token" in new_data
        assert "refresh_token" in new_data
        # New tokens should be different from original
        assert new_data["access_token"] != old_access

    def test_refresh_with_invalid_token(self):
        resp = requests.post(
            url("auth/refresh"),
            json={"refresh_token": "invalid.token.value"},
            timeout=TIMEOUT,
        )
        assert resp.status_code in (400, 401)

    def test_refresh_with_access_token_fails(self):
        """An access token must not be usable as a refresh token."""
        data = do_login()
        resp = requests.post(
            url("auth/refresh"),
            json={"refresh_token": data["access_token"]},
            timeout=TIMEOUT,
        )
        assert resp.status_code in (400, 401)

    def test_refresh_missing_token_field(self):
        resp = requests.post(url("auth/refresh"), json={}, timeout=TIMEOUT)
        assert resp.status_code == 400

    def test_refresh_response_has_user(self):
        data = do_login()
        resp = requests.post(
            url("auth/refresh"),
            json={"refresh_token": data["refresh_token"]},
            timeout=TIMEOUT,
        )
        assert resp.status_code == 200
        new_data = resp.json()["data"]
        assert "user" in new_data


# ---------------------------------------------------------------------------
# Logout
# ---------------------------------------------------------------------------

class TestLogout:
    """POST /auth/logout"""

    def test_logout_success(self):
        data = do_login()
        resp = requests.post(
            url("auth/logout"),
            json={"refresh_token": data["refresh_token"]},
            headers=bearer(data["access_token"]),
            timeout=TIMEOUT,
        )
        assert resp.status_code == 200

    def test_logout_blacklists_access_token(self):
        data = do_login()
        token = data["access_token"]

        requests.post(
            url("auth/logout"),
            headers=bearer(token),
            timeout=TIMEOUT,
        )

        # Token should now be rejected
        resp = requests.get(url("auth/me"), headers=bearer(token), timeout=TIMEOUT)
        assert resp.status_code == 401

    def test_logout_without_auth_fails(self):
        resp = requests.post(url("auth/logout"), timeout=TIMEOUT)
        assert resp.status_code == 401

    def test_logout_with_invalid_token(self):
        resp = requests.post(
            url("auth/logout"),
            headers=bearer("totally.invalid.token"),
            timeout=TIMEOUT,
        )
        assert resp.status_code == 401

    def test_logout_response_has_message(self):
        data = do_login()
        resp = requests.post(
            url("auth/logout"),
            headers=bearer(data["access_token"]),
            timeout=TIMEOUT,
        )
        body = resp.json()
        assert "data" in body


# ---------------------------------------------------------------------------
# Current user (/me)
# ---------------------------------------------------------------------------

class TestMe:
    """GET /auth/me"""

    def test_me_returns_user(self):
        data = do_login()
        resp = requests.get(
            url("auth/me"),
            headers=bearer(data["access_token"]),
            timeout=TIMEOUT,
        )
        assert resp.status_code == 200
        user = resp.json()["data"]
        assert "id" in user
        assert "username" in user
        assert "role" in user

    def test_me_username_matches_login(self):
        data = do_login()
        resp = requests.get(
            url("auth/me"),
            headers=bearer(data["access_token"]),
            timeout=TIMEOUT,
        )
        user = resp.json()["data"]
        assert user["username"] == TEST_USERNAME

    def test_me_without_auth_fails(self):
        resp = requests.get(url("auth/me"), timeout=TIMEOUT)
        assert resp.status_code == 401

    def test_me_with_expired_token_fails(self):
        fake_expired = (
            "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9."
            "eyJ1aWQiOjEsInN1YiI6ImFkbWluIiwicm9sZSI6Im93bmVyIiwianRpIjoieHh4IiwiaWF0IjoxNjAwMDAwMDAwLCJleHAiOjE2MDAwMDAwMDEsImlzcyI6Im1pa2htb24tYXBpIn0."
            "invalidsignature"
        )
        resp = requests.get(url("auth/me"), headers=bearer(fake_expired), timeout=TIMEOUT)
        assert resp.status_code == 401


# ---------------------------------------------------------------------------
# Change password
# ---------------------------------------------------------------------------

class TestChangePassword:
    """PUT /auth/password"""

    def test_change_password_and_login_with_new(self):
        data = do_login()
        token = data["access_token"]

        # Change password
        resp = requests.put(
            url("auth/password"),
            json={"old_password": TEST_PASSWORD, "new_password": TEST_NEW_PASSWORD},
            headers=bearer(token),
            timeout=TIMEOUT,
        )
        assert resp.status_code == 200, f"change failed: {resp.text}"

        # Login with new password
        resp2 = login(password=TEST_NEW_PASSWORD)
        assert resp2.status_code == 200

        # Restore original password
        data2 = resp2.json()["data"]
        requests.put(
            url("auth/password"),
            json={"old_password": TEST_NEW_PASSWORD, "new_password": TEST_PASSWORD},
            headers=bearer(data2["access_token"]),
            timeout=TIMEOUT,
        )

    def test_change_password_wrong_old_password(self):
        data = do_login()
        resp = requests.put(
            url("auth/password"),
            json={"old_password": "wrongpassword", "new_password": TEST_NEW_PASSWORD},
            headers=bearer(data["access_token"]),
            timeout=TIMEOUT,
        )
        assert resp.status_code in (400, 401)

    def test_change_password_too_short_new_password(self):
        data = do_login()
        resp = requests.put(
            url("auth/password"),
            json={"old_password": TEST_PASSWORD, "new_password": "12345"},
            headers=bearer(data["access_token"]),
            timeout=TIMEOUT,
        )
        assert resp.status_code == 400

    def test_change_password_without_auth(self):
        resp = requests.put(
            url("auth/password"),
            json={"old_password": TEST_PASSWORD, "new_password": TEST_NEW_PASSWORD},
            timeout=TIMEOUT,
        )
        assert resp.status_code == 401

    def test_change_password_missing_fields(self):
        data = do_login()
        resp = requests.put(
            url("auth/password"),
            json={"old_password": TEST_PASSWORD},
            headers=bearer(data["access_token"]),
            timeout=TIMEOUT,
        )
        assert resp.status_code == 400
