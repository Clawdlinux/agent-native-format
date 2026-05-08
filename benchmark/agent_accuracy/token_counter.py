"""Tokenizer-accurate token counting for the agent-accuracy benchmark.

Each provider exposes a different way to count tokens:

- **OpenAI** uses BPE tokenizers shipped via the `tiktoken` library.
  We resolve the model name to its tokenizer (`o200k_base` for the
  `gpt-4o*` family, `cl100k_base` for older 3.5/4.x). Counting is done
  fully offline.

- **Anthropic** does not publish a public tokenizer; the official SDK
  exposes `client.messages.count_tokens(...)` which makes a single
  cheap (free) HTTP call. We use that, then cache the result on disk
  by content hash so a re-run never spends a second roundtrip.

The point of separating this from the harness is to keep the cost
estimator and the actual call accounting using the *same* counter, so
the pre-flight cost cap matches reality.

Public surface:
    count(model: str, text: str) -> int
    estimate_usd(model: str, prompt_tokens: int, completion_tokens: int) -> float
"""

from __future__ import annotations

import functools
import hashlib
import json
import os
from dataclasses import dataclass
from pathlib import Path

import tiktoken


# Cache for Anthropic counts. JSON file at the cache dir; small enough
# that we just rewrite it on every miss.
_CACHE_DIR = Path(__file__).resolve().parent / ".cache"
_CACHE_DIR.mkdir(exist_ok=True)
_ANTHROPIC_CACHE = _CACHE_DIR / "anthropic_token_counts.json"


@dataclass(frozen=True)
class ModelPricing:
    """USD per 1K tokens. Update when providers change rates."""

    input_per_1k: float
    output_per_1k: float


# Prices as of 2026-05; sourced from each provider's pricing page. If
# you re-run the benchmark months later, update these and re-derive
# total spend (the raw token counts in results/ remain authoritative).
_PRICING: dict[str, ModelPricing] = {
    # OpenAI
    "gpt-4o-mini":               ModelPricing(input_per_1k=0.00015, output_per_1k=0.0006),
    "gpt-4.1-mini":              ModelPricing(input_per_1k=0.0004,  output_per_1k=0.0016),
    "gpt-5-mini":                ModelPricing(input_per_1k=0.00025, output_per_1k=0.002),
    "gpt-5-nano":                ModelPricing(input_per_1k=0.00005, output_per_1k=0.0004),
    # Anthropic (May 2026 catalogue)
    "claude-haiku-4-5-20251001": ModelPricing(input_per_1k=0.0008,  output_per_1k=0.004),
    "claude-haiku-4-5":          ModelPricing(input_per_1k=0.0008,  output_per_1k=0.004),
    # Local / Ollama (cost is electricity, accounted as $0).
    "ollama:gemma3:1b":          ModelPricing(input_per_1k=0.0,     output_per_1k=0.0),
    "ollama:gemma3":             ModelPricing(input_per_1k=0.0,     output_per_1k=0.0),
    "ollama:llama3.1":           ModelPricing(input_per_1k=0.0,     output_per_1k=0.0),
    "ollama:llama3.2":           ModelPricing(input_per_1k=0.0,     output_per_1k=0.0),
    "ollama:qwen2.5":            ModelPricing(input_per_1k=0.0,     output_per_1k=0.0),
}


def _openai_encoding_for(model: str) -> tiktoken.Encoding:
    """Resolve a tiktoken encoding for an OpenAI model name.

    `tiktoken.encoding_for_model` already does this lookup but raises
    on unknown models; we fall back to `o200k_base` (the gpt-4o family
    encoding) which covers any current OpenAI model name.
    """
    try:
        return tiktoken.encoding_for_model(model)
    except KeyError:
        return tiktoken.get_encoding("o200k_base")


def _is_openai_model(model: str) -> bool:
    return model.startswith(("gpt-", "o1", "o3", "o4"))


def _is_anthropic_model(model: str) -> bool:
    return model.startswith("claude-")


def _is_ollama_model(model: str) -> bool:
    return model.startswith("ollama:")


def _load_anthropic_cache() -> dict[str, int]:
    if not _ANTHROPIC_CACHE.exists():
        return {}
    try:
        return json.loads(_ANTHROPIC_CACHE.read_text())
    except (json.JSONDecodeError, OSError):
        return {}


def _save_anthropic_cache(cache: dict[str, int]) -> None:
    _ANTHROPIC_CACHE.write_text(json.dumps(cache, indent=2, sort_keys=True))


def _content_hash(model: str, text: str) -> str:
    h = hashlib.sha256()
    h.update(model.encode())
    h.update(b"\x00")
    h.update(text.encode())
    return h.hexdigest()


@functools.lru_cache(maxsize=8)
def _anthropic_client():
    # Imported lazily so the module is importable without the SDK
    # (e.g. when only counting OpenAI tokens).
    import anthropic

    api_key = os.environ.get("ANTHROPIC_API_KEY")
    if not api_key:
        raise RuntimeError("ANTHROPIC_API_KEY not set")
    return anthropic.Anthropic(api_key=api_key)


def count(model: str, text: str) -> int:
    """Return the number of tokens `text` would consume on `model`.

    Routes to the right tokenizer per provider. Anthropic counts are
    cached on disk so re-runs don't re-hit the count endpoint. Ollama
    counts use cl100k_base as a portable approximation — for local
    cost it doesn't matter (cost is electricity), and the accuracy
    measurement is what the run records anyway.
    """
    if _is_openai_model(model):
        enc = _openai_encoding_for(model)
        return len(enc.encode(text))
    if _is_ollama_model(model):
        # Approximation: cl100k_base is close to most modern open-model
        # tokenizers within ~10% on natural-language workloads.
        enc = tiktoken.get_encoding("cl100k_base")
        return len(enc.encode(text))
    if _is_anthropic_model(model):
        key = _content_hash(model, text)
        cache = _load_anthropic_cache()
        if key in cache:
            return cache[key]
        client = _anthropic_client()
        # count_tokens is free and rate-limited generously; safe to
        # call from the cost estimator.
        result = client.messages.count_tokens(
            model=model,
            messages=[{"role": "user", "content": text}],
        )
        cache[key] = result.input_tokens
        _save_anthropic_cache(cache)
        return result.input_tokens
    raise ValueError(f"unknown model family for token counting: {model!r}")


def estimate_usd(model: str, prompt_tokens: int, completion_tokens: int) -> float:
    """Return the USD cost of a single call. Returns 0.0 for unknown
    models with a printed warning so a typo doesn't silently break the
    cost cap."""
    pricing = _PRICING.get(model)
    if pricing is None:
        print(f"WARNING: no pricing entry for {model!r}; cost estimate = 0")
        return 0.0
    return (
        prompt_tokens * pricing.input_per_1k / 1000.0
        + completion_tokens * pricing.output_per_1k / 1000.0
    )


def known_models() -> list[str]:
    return sorted(_PRICING.keys())
