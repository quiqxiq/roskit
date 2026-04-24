"""
pytest conftest — shared fixtures available across all test files.
"""
import pytest
from config import APIClient, TEST_USERNAME, TEST_PASSWORD


@pytest.fixture(scope="session")
def api_client() -> APIClient:
    """Authenticated API client shared for the whole test session."""
    client = APIClient()
    client.login(TEST_USERNAME, TEST_PASSWORD)
    return client


@pytest.fixture(scope="session")
def access_token(api_client: APIClient) -> str:
    return api_client.access_token


@pytest.fixture(scope="session")
def refresh_token(api_client: APIClient) -> str:
    return api_client.refresh_token
