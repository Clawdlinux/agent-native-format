"""Provider-specific chat-completion wrappers.

Both wrappers expose the same interface:

    call(messages, max_tokens, temperature) -> CallResult

so the harness loop is provider-agnostic. Each wrapper handles its
own SDK quirks (Anthropic's separate `system` parameter, OpenAI's
`response_format`, etc.) and reports usage in a unified shape.
"""

from __future__ import annotations

import os
import time
from dataclasses import dataclass


@dataclass(frozen=True)
class CallResult:
    response_text: str
    prompt_tokens: int
    completion_tokens: int
    latency_ms: float


class _MissingKey(RuntimeError):
    pass


# ─── OpenAI ─────────────────────────────────────────────────────────────────


class OpenAIClient:
    """Thin wrapper around the openai SDK chat-completions endpoint.

    Supports a custom base_url so it doubles as the Ollama client
    (Ollama exposes an OpenAI-compatible /v1/chat/completions endpoint
    on http://localhost:11434/v1).
    """

    def __init__(self, model: str, base_url: str | None = None, api_key: str | None = None):
        if base_url is None:
            api_key = os.environ.get("OPENAI_API_KEY")
            if not api_key:
                raise _MissingKey("OPENAI_API_KEY not set")
        else:
            # Local endpoints (Ollama) take any non-empty key.
            api_key = api_key or "local"
        from openai import OpenAI

        self.model = model
        self._client = OpenAI(api_key=api_key, base_url=base_url)

    def call(self, messages: list[dict], max_tokens: int, temperature: float) -> CallResult:
        t0 = time.perf_counter()
        resp = self._client.chat.completions.create(
            model=self.model,
            messages=messages,
            max_tokens=max_tokens,
            temperature=temperature,
        )
        latency_ms = (time.perf_counter() - t0) * 1000.0
        # The SDK returns a Pydantic-shaped object; pulling .content is safe
        # for chat completions with a single choice.
        text = resp.choices[0].message.content or ""
        usage = resp.usage
        return CallResult(
            response_text=text,
            prompt_tokens=usage.prompt_tokens,
            completion_tokens=usage.completion_tokens,
            latency_ms=latency_ms,
        )


# ─── Anthropic ──────────────────────────────────────────────────────────────


class AnthropicClient:
    def __init__(self, model: str):
        api_key = os.environ.get("ANTHROPIC_API_KEY")
        if not api_key:
            raise _MissingKey("ANTHROPIC_API_KEY not set")
        import anthropic

        self.model = model
        self._client = anthropic.Anthropic(api_key=api_key)

    def call(self, messages: list[dict], max_tokens: int, temperature: float) -> CallResult:
        # The harness builds OpenAI-shaped messages with role=system at
        # index 0; Anthropic wants `system=` separately and the rest
        # under `messages=`.
        system_msg = ""
        user_msgs: list[dict] = []
        for m in messages:
            if m["role"] == "system":
                system_msg = m["content"]
            else:
                user_msgs.append({"role": m["role"], "content": m["content"]})
        t0 = time.perf_counter()
        resp = self._client.messages.create(
            model=self.model,
            system=system_msg,
            messages=user_msgs,
            max_tokens=max_tokens,
            temperature=temperature,
        )
        latency_ms = (time.perf_counter() - t0) * 1000.0
        # Anthropic returns content as a list of blocks; for a plain
        # text reply it's a single TextBlock.
        text_parts = [block.text for block in resp.content if getattr(block, "type", None) == "text"]
        text = "".join(text_parts)
        return CallResult(
            response_text=text,
            prompt_tokens=resp.usage.input_tokens,
            completion_tokens=resp.usage.output_tokens,
            latency_ms=latency_ms,
        )


# ─── factory ────────────────────────────────────────────────────────────────


def make_client(model: str):
    """Return the right wrapper for `model`, or raise _MissingKey if
    the corresponding API key is unset.

    `ollama:<name>` is treated as a local model served by Ollama on
    its default port (set OLLAMA_HOST to override).
    """
    if model.startswith("ollama:"):
        ollama_host = os.environ.get("OLLAMA_HOST", "http://localhost:11434")
        return OpenAIClient(model.removeprefix("ollama:"), base_url=f"{ollama_host}/v1")
    if model.startswith(("gpt-", "o1", "o3", "o4")):
        return OpenAIClient(model)
    if model.startswith("claude-"):
        return AnthropicClient(model)
    raise ValueError(f"unknown model family: {model!r}")


def is_missing_key(exc: BaseException) -> bool:
    return isinstance(exc, _MissingKey)
