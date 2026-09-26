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
    output: list = field(repr=False)  # every line the process wrote, in order
    pump: threading.Thread = field(repr=False)
    log_path: pathlib.Path | None = None

    def stop(self) -> tuple[int, list[dict]]:
        """Send SIGTERM and return the exit status and every line after the first.

        The pump thread ends at end of file, so joining it guarantees the last
        line the process wrote has been read.
        """
        self.process.terminate()
        code = self.process.wait(timeout=START_TIMEOUT)
        self.pump.join(timeout=START_TIMEOUT)
        return code, [json.loads(line) for line in self.output[1:] if line]


def _environment() -> dict:
    # Only what the binary needs: an inherited IDENTITY_* variable would make it refuse to start.
    return {"PATH": os.environ.get("PATH", ""), "IDENTITY_HTTP_ADDR": "127.0.0.1:0"}


def start_server(log_path: pathlib.Path) -> Server:
    """Start the binary, keeping everything it writes in log_path (a CI artefact)."""
    log_path.parent.mkdir(parents=True, exist_ok=True)
    started = time.monotonic()
    process = subprocess.Popen(
        [str(BINARY), "serve"],
        env=_environment(),
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
    )
    output: list[str] = []
    first: "queue.Queue[str]" = queue.Queue()

    def pump() -> None:
        with log_path.open("w") as log:
            for line in process.stdout:
                log.write(line)
                log.flush()
                output.append(line.strip())
                if len(output) == 1:
                    first.put(output[0])

    reader = threading.Thread(target=pump, daemon=True)
    reader.start()
    line = first.get(timeout=START_TIMEOUT)
    startup = json.loads(line)
    assert startup["message"] == "serving", line
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
    return Server(process, startup, base_url, time.monotonic() - started, output, reader, log_path)


@pytest.fixture
def server(request):
    s = start_server(ARTIFACTS / f"server-{request.node.name}.log")
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
