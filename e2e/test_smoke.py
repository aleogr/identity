# SPDX-License-Identifier: AGPL-3.0-only
"""Smoke tests of the identity binary: it starts, answers and stops."""

import json
import subprocess

from conftest import ARTIFACTS, ROOT


def test_chromium_reads_health(server, page):
    response = page.goto(server.base_url + "/health")
    assert response is not None
    assert response.status == 200
    assert response.headers["content-type"] == "application/json"
    assert response.headers["cache-control"] == "no-store"
    assert response.headers["x-content-type-options"] == "nosniff"
    assert json.loads(response.text()) == {"status": "ok"}
    page.screenshot(path=ARTIFACTS / "health.png")


def test_cold_start_under_one_second(server):
    # Requirement §18: the binary serves requests within 1 second of starting.
    assert server.ready_seconds < 1.0, f"ready after {server.ready_seconds:.3f}s"


def test_startup_log_carries_the_build_identifier(server):
    expected = subprocess.run(
        ["git", "describe", "--tags", "--always", "--dirty", "--abbrev=12"],
        cwd=ROOT, capture_output=True, text=True, check=True,
    ).stdout.strip()
    assert server.startup["build_id"] == expected
    assert server.startup["severity"] == "INFO"
    assert server.startup["go_version"].startswith("go")


def test_sigterm_stops_gracefully(server):
    code, later = server.stop()
    assert code == 0
    assert later and later[-1]["message"] == "stopped"


def test_server_output_is_kept_as_an_artefact(server):
    server.stop()
    kept = server.log_path.read_text().splitlines()
    assert json.loads(kept[0])["message"] == "serving"
    assert json.loads(kept[-1])["message"] == "stopped"
