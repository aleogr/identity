# SPDX-License-Identifier: AGPL-3.0-only
"""Fixtures that start the identity binary and a Chromium page."""

import json
import os
import pathlib
import queue
import subprocess
import threading
import time
import urllib.request
from dataclasses import dataclass, field

import pytest
from playwright.sync_api import sync_playwright

ROOT = pathlib.Path(__file__).resolve().parent.parent
BINARY = pathlib.Path(os.environ.get("E2E_BINARY", ROOT / "bin" / "identity"))
ARTIFACTS = ROOT / "e2e" / "artifacts"
START_TIMEOUT = 10.0


@dataclass
class Server:
    process: subprocess.Popen
    startup: dict
    base_url: str
    ready_seconds: float
    lines: "queue.Queue[str]" = field(repr=False)

    def stop(self) -> tuple[int, list[dict]]:
        """Send SIGTERM and return the exit status and every later log line."""
        self.process.terminate()
        code = self.process.wait(timeout=START_TIMEOUT)
        later = []
        while not self.lines.empty():
            line = self.lines.get()
            if line:
                later.append(json.loads(line))
        return code, later


def _environment() -> dict:
    # Only what the binary needs: an inherited IDENTITY_* variable would make it refuse to start.
    return {"PATH": os.environ.get("PATH", ""), "IDENTITY_HTTP_ADDR": "127.0.0.1:0"}


def start_server() -> Server:
    started = time.monotonic()
    process = subprocess.Popen(
        [str(BINARY), "serve"],
        env=_environment(),
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
    )
    lines: "queue.Queue[str]" = queue.Queue()

    def pump() -> None:
        for line in process.stdout:
            lines.put(line.strip())

    threading.Thread(target=pump, daemon=True).start()
    first = lines.get(timeout=START_TIMEOUT)
    startup = json.loads(first)
    assert startup["message"] == "serving", first
    base_url = "http://" + startup["addr"]
    while True:
        try:
            with urllib.request.urlopen(base_url + "/health", timeout=1) as response:
                if response.status == 200:
                    break
        except OSError:
            if time.monotonic() - started > START_TIMEOUT:
                raise
            time.sleep(0.01)
    return Server(process, startup, base_url, time.monotonic() - started, lines)


@pytest.fixture
def server():
    s = start_server()
    yield s
    if s.process.poll() is None:
        s.process.kill()
        s.process.wait()


@pytest.fixture(scope="session")
def browser():
    with sync_playwright() as p:
        b = p.chromium.launch()
        yield b
        b.close()


@pytest.fixture
def page(browser):
    ARTIFACTS.mkdir(parents=True, exist_ok=True)
    context = browser.new_context()
    p = context.new_page()
    yield p
    context.close()
