"""
Shared configuration and helpers for all HTTP tests.
"""
import os
import requests
from typing import Optional

BASE_URL = os.getenv("API_BASE_URL", "http://localhost:8080/api/v1")
TIMEOUT = int(os.getenv("API_TIMEOUT", "10"))

# Credentials used across test files (override via env vars)
TEST_USERNAME = os.getenv("TEST_USERNAME", "admin")
TEST_PASSWORD = os.getenv("TEST_PASSWORD", "admin123")
TEST_NEW_PASSWORD = os.getenv("TEST_NEW_PASSWORD", "newpassword123")

# Real Mikrotik router credentials for integration tests (optional)
ROUTER_IP = os.getenv("ROUTER_IP", "")
ROUTER_USERNAME = os.getenv("ROUTER_USERNAME", "admin")
ROUTER_PASSWORD = os.getenv("ROUTER_PASSWORD", "")
ROUTER_ID = os.getenv("ROUTER_ID", "")  # pre-existing router ID in DB


def url(path: str) -> str:
    """Build full URL from path."""
    return f"{BASE_URL}/{path.lstrip('/')}"


def bearer(token: str) -> dict:
    """Build Authorization header."""
    return {"Authorization": f"Bearer {token}"}


class APIClient:
    """Thin wrapper around requests with shared auth state."""

    def __init__(self):
        self.session = requests.Session()
        self.access_token: Optional[str] = None
        self.refresh_token: Optional[str] = None

    def login(self, username: str = TEST_USERNAME, password: str = TEST_PASSWORD) -> dict:
        resp = self.session.post(
            url("auth/login"),
            json={"username": username, "password": password},
            timeout=TIMEOUT,
        )
        resp.raise_for_status()
        body = resp.json()
        data = body.get("data", {})
        self.access_token = data.get("access_token")
        self.refresh_token = data.get("refresh_token")
        return data

    def auth_headers(self) -> dict:
        if not self.access_token:
            self.login()
        return bearer(self.access_token)

    def get(self, path: str, **kwargs) -> requests.Response:
        return self.session.get(url(path), headers=self.auth_headers(), timeout=TIMEOUT, **kwargs)

    def post(self, path: str, **kwargs) -> requests.Response:
        return self.session.post(url(path), headers=self.auth_headers(), timeout=TIMEOUT, **kwargs)

    def put(self, path: str, **kwargs) -> requests.Response:
        return self.session.put(url(path), headers=self.auth_headers(), timeout=TIMEOUT, **kwargs)

    def delete(self, path: str, **kwargs) -> requests.Response:
        return self.session.delete(url(path), headers=self.auth_headers(), timeout=TIMEOUT, **kwargs)
