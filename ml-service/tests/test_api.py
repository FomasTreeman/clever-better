import pytest


@pytest.mark.asyncio
async def test_health(async_client):
    response = await async_client.get("/health")
    assert response.status_code == 200
    payload = response.json()
    assert payload["status"] == "ok"


@pytest.mark.asyncio
async def test_api_health(async_client):
    response = await async_client.get("/api/v1/health")
    assert response.status_code == 200
    payload = response.json()
    assert payload["status"] == "ok"
