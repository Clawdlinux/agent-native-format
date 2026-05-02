"""Property-style tests for acp_common.

These complement the example-based tests in test_acp_common.py by checking
invariants over many randomized inputs without requiring Hypothesis.
"""

from __future__ import annotations

import random
import string

from acp_common import (
    Action,
    CycleError,
    expand_schema_field,
    topological_order,
)


def _random_atom(rng: random.Random) -> str:
    base = rng.choice(["string", "int", "float", "bool", "json", "bytes"])
    arrays = rng.randint(0, 2)
    optional = rng.random() < 0.4
    return base + ("[]" * arrays) + ("?" if optional else "")


def test_expand_schema_field_round_trip_atoms():
    """Every atom we emit should always produce a valid {type: ...} dict."""
    rng = random.Random(0xACB)
    valid_top = {"string", "integer", "number", "boolean", "object"}
    for _ in range(200):
        compact = _random_atom(rng)
        out = expand_schema_field(compact)
        # Walk into nested array items for the strict check.
        cur = out
        while cur.get("type") == "array":
            cur = cur["items"]
        assert cur["type"] in valid_top, (compact, out)


def test_expand_schema_field_enum_preserves_options():
    rng = random.Random(0xE)
    for _ in range(50):
        opts = ["".join(rng.choices(string.ascii_lowercase, k=rng.randint(1, 6))) for _ in range(rng.randint(1, 5))]
        out = expand_schema_field("enum:" + "|".join(opts))
        assert out["type"] == "string"
        assert out["enum"] == opts


def _make_chain(n: int) -> list[Action]:
    actions: list[Action] = []
    for i in range(n):
        deps: tuple[str, ...] = (f"a{i - 1}",) if i > 0 else ()
        actions.append(
            Action(f"a{i}", "http", "u", "POST", {}, "pre-injected", depends_on=deps)
        )
    return actions


def test_topological_order_chain_is_input_order():
    actions = _make_chain(20)
    order = [a.id for a in topological_order(actions)]
    assert order == [a.id for a in actions]


def test_topological_order_random_permutation_still_valid():
    """Shuffling the input must not change the partial-order constraints
    (parent always before child) in the output."""
    rng = random.Random(0x7E57)
    for trial in range(20):
        actions = _make_chain(15)
        rng.shuffle(actions)
        order = topological_order(actions)
        position = {a.id: i for i, a in enumerate(order)}
        for a in actions:
            for dep in a.depends_on:
                assert position[dep] < position[a.id], (
                    f"trial {trial}: {dep} not before {a.id} in {[x.id for x in order]}"
                )


def test_topological_order_diamond_graphs():
    """For every k>=2, a 'k-fold diamond' (root -> k middle -> sink) must
    place root first and sink last."""
    rng = random.Random(0x71D)
    for k in range(2, 10):
        actions = [Action("root", "http", "u", "POST", {}, "pre-injected", depends_on=())]
        middles = [
            Action(f"m{i}", "http", "u", "POST", {}, "pre-injected", depends_on=("root",))
            for i in range(k)
        ]
        actions.extend(middles)
        actions.append(
            Action(
                "sink",
                "http",
                "u",
                "POST",
                {},
                "pre-injected",
                depends_on=tuple(m.id for m in middles),
            )
        )
        rng.shuffle(actions)
        order = [a.id for a in topological_order(actions)]
        assert order[0] == "root"
        assert order[-1] == "sink"


def test_topological_order_detects_cycles_in_random_graphs():
    """Forcing a cycle (b -> a, a -> b) into otherwise-random graphs must
    always raise CycleError."""
    rng = random.Random(0xC1C)
    for trial in range(20):
        n = rng.randint(3, 10)
        actions = []
        for i in range(n):
            deps = tuple(f"a{j}" for j in range(i) if rng.random() < 0.3)
            actions.append(
                Action(f"a{i}", "http", "u", "POST", {}, "pre-injected", depends_on=deps)
            )
        # Inject an unambiguous back-edge between the first two nodes so
        # there is always a real cycle (a0 -> a1 -> a0).
        actions[0] = Action(
            "a0", "http", "u", "POST", {}, "pre-injected", depends_on=("a1",)
        )
        actions[1] = Action(
            "a1", "http", "u", "POST", {}, "pre-injected", depends_on=("a0",)
        )
        try:
            topological_order(actions)
        except CycleError:
            continue
        raise AssertionError(f"trial {trial}: cycle not detected (n={n})")
