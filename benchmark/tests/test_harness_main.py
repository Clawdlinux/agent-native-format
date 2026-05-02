"""Tests for benchmark/harness.py beyond the determinism cases.

Covers:
- The mock-mode harness path (request_acp_manifest with a fake server).
- main() argument plumbing.
"""

from __future__ import annotations

import json
import sys
from pathlib import Path
from urllib.error import HTTPError, URLError

import pytest

from harness import (
    SCENARIOS,
    Scenario,
    measure_run,
    request_acp_manifest,
)


SAMPLE_MANIFEST = {
    "manifest_id": "m-fake",
    "version": "acp/v1",
    "ttl": "300s",
    "actions": [
        {
            "id": "a1",
            "type": "http",
            "endpoint": "grpc://db",
            "method": "POST",
            "schema": {"sql": "string"},
            "auth": "pre-injected",
        }
    ],
    "boundaries": {"egress": ["db"], "max_tokens_per_action": 15000, "audit_level": "full"},
    "feedback_endpoint": "/v1/feedback",
}


def _patch_urlopen(monkeypatch, status: int, body: dict | str):
    """Patch urllib.request.urlopen to return a canned response."""
    raw = json.dumps(body).encode("utf-8") if isinstance(body, dict) else body.encode("utf-8")

    class _Resp:
        def __enter__(self):
            return self

        def __exit__(self, *_a):
            return False

        def read(self):
            return raw

        def getcode(self):
            return status

    monkeypatch.setattr("urllib.request.urlopen", lambda req, timeout: _Resp())


def test_request_acp_manifest_returns_body_and_wall(monkeypatch):
    _patch_urlopen(monkeypatch, 200, SAMPLE_MANIFEST)
    body, wall_ms = request_acp_manifest("http://acp.local", "tok", SCENARIOS[0])
    assert json.loads(body)["manifest_id"] == "m-fake"
    assert wall_ms >= 0


def test_request_acp_manifest_propagates_http_error(monkeypatch):
    def boom(req, timeout):
        raise HTTPError(req.full_url, 401, "no auth", hdrs=None, fp=None)

    monkeypatch.setattr("urllib.request.urlopen", boom)
    with pytest.raises(HTTPError):
        request_acp_manifest("http://acp.local", None, SCENARIOS[0])


def test_measure_run_aggregates_acp_and_mcp(monkeypatch):
    _patch_urlopen(monkeypatch, 200, SAMPLE_MANIFEST)
    result = measure_run("http://acp.local", None, SCENARIOS[0])
    assert result.scenario_id == "S1"
    assert result.acp_tokens > 0
    assert result.mcp_tokens > 0
    assert result.mcp_tokens > result.acp_tokens
    assert result.acp_round_trips == 1
    assert result.mcp_round_trips >= 1


def test_main_writes_results_file(monkeypatch, tmp_path: Path, capsys):
    import harness

    _patch_urlopen(monkeypatch, 200, SAMPLE_MANIFEST)
    out = tmp_path / "out.json"
    rc = harness.main(["--acp-url", "http://acp.local", "--runs", "2", "--out", str(out)])
    assert rc == 0
    assert out.exists()
    payload = json.loads(out.read_text())
    assert payload["scenarios"]
    assert payload["raw"]


def test_main_returns_2_on_connect_error(monkeypatch, tmp_path: Path):
    import harness

    def boom(req, timeout):
        raise URLError("connect refused")

    monkeypatch.setattr("urllib.request.urlopen", boom)
    out = tmp_path / "out.json"
    rc = harness.main(["--acp-url", "http://acp.local", "--runs", "1", "--out", str(out)])
    assert rc == 2


def test_scenario_default_noise_is_zero():
    s = Scenario(
        id="X", title="t", intent="i", capability_hints=[], tool_ids=["db.query"], mcp_servers=1
    )
    assert s.mcp_noise_tools == 0
