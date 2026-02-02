import asyncio
import os
from typing import AsyncGenerator

import pytest
from httpx import AsyncClient


@pytest.fixture(scope="session")
def event_loop() -> asyncio.AbstractEventLoop:
    loop = asyncio.new_event_loop()
    yield loop
    loop.close()


@pytest.fixture
async def async_client() -> AsyncGenerator[AsyncClient, None]:
    os.environ.setdefault(
        "DATABASE_URL", "postgresql+asyncpg://postgres:postgres@localhost:5432/clever_better"
    )
    from app.main import app

    async with AsyncClient(app=app, base_url="http://test") as client:
        yield client
