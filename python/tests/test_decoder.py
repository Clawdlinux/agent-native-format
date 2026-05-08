# Copyright 2026 NineVigil / Clawdlinux.
#
# Licensed under the Apache License, Version 2.0 (the "License").

"""Tests for the pure-Python ACL decoder."""

from __future__ import annotations

import pytest

import acp_acl


CANONICAL = (
    "@cluster prod-east\n"
    "@ns payments\n"
    "@as-of 2026-05-03T12:34:56Z\n"
    "\n"
    "pods 12/12 ok\n"
    "  payment-api-7f8d x3 cpu=42 mem=61 r=0 age=3d\n"
    "  payment-worker-9a2b x2 cpu=87 mem=78 r=3 age=1d !\n"
    "\n"
    "actions\n"
    "  scale|rollout|restart|logs|describe\n"
)


def test_decode_canonical():
    doc = acp_acl.decode(CANONICAL)
    assert doc.directives["cluster"] == "prod-east"
    assert doc.directives["ns"] == "payments"
    assert len(doc.sections) == 2

    pods = doc.section("pods")
    assert pods is not None
    assert pods.summary == "12/12 ok"
    assert len(pods.rows) == 2
    assert pods.rows[0].id == "payment-api-7f8d"
    assert pods.rows[0].count == 3
    assert pods.rows[0].fields["cpu"] == "42"
    assert pods.rows[1].flags == ["!"]
    assert pods.rows[1].fields["r"] == "3"

    actions = doc.section("actions")
    assert actions is not None
    assert actions.rows[0].id == "scale|rollout|restart|logs|describe"


def test_decode_quoted_value():
    src = "logs\n  nginx cmd=`tail -n 100 /var/log/nginx.log`\n"
    doc = acp_acl.decode(src)
    row = doc.section("logs").rows[0]
    assert row.fields["cmd"] == "tail -n 100 /var/log/nginx.log"


def test_decode_strips_comments():
    src = "# generated\n@k v\n\n# inner\nfoo 1\n  a x=1\n"
    doc = acp_acl.decode(src)
    assert doc.directives["k"] == "v"
    assert doc.section("foo").rows[0].fields["x"] == "1"


def test_decode_rejects_tabs():
    with pytest.raises(acp_acl.DecodeError, match="tab"):
        acp_acl.decode("foo\n\tbad x=1\n")


def test_decode_rejects_orphan_row():
    with pytest.raises(acp_acl.DecodeError, match="outside any section"):
        acp_acl.decode("  orphan x=1\n")


def test_decode_rejects_one_space_indent():
    with pytest.raises(acp_acl.DecodeError, match="exactly two spaces"):
        acp_acl.decode("foo\n bad x=1\n")


def test_decode_rejects_directive_after_section():
    with pytest.raises(acp_acl.DecodeError, match="directive after section"):
        acp_acl.decode("foo 1\n  a x=1\n@k v\n")


def test_directive_without_value():
    doc = acp_acl.decode("@flagonly\n")
    assert doc.directives == {"flagonly": ""}


def test_count_tokens_smoke():
    pytest.importorskip("tiktoken")
    n = acp_acl.count_tokens(CANONICAL)
    assert n > 0
    assert n < len(CANONICAL)  # tokens are denser than chars


def test_iteration():
    doc = acp_acl.decode(CANONICAL)
    names = [s.name for s in doc]
    assert names == ["pods", "actions"]
