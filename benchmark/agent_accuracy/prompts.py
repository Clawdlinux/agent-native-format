"""Prompt construction for the agent-accuracy benchmark.

Both conditions (`raw` and `acl`) use the same system prompt and the
same per-question instruction. The only thing that varies is which
representation of the namespace state is included in the user message.

This is the **fairness lever**: any difference in accuracy must come
from the data representation, not from prompt engineering. Anything
added to one prompt must be added to the other.
"""

from __future__ import annotations

from dataclasses import dataclass

SYSTEM_PROMPT = (
    "You are an SRE assistant for a Kubernetes namespace. "
    "Answer the user's question using ONLY the provided cluster state. "
    "Be terse: respond in the exact format the question requests, with "
    "no extra commentary, no markdown, and no explanation unless asked."
)

# These wrapper templates are intentionally symmetric. The label
# ("kubectl JSON" vs "ACL document") is included so the model knows
# the format, but no other hint is given.
_RAW_WRAPPER = (
    "Below is the current namespace state as kubectl JSON.\n"
    "<state format=\"kubectl-json\">\n{payload}\n</state>\n\n"
    "Question: {question}"
)

_ACL_WRAPPER = (
    "Below is the current namespace state as an Agent Context Language (ACL) document.\n"
    "<state format=\"acl-v0.1\">\n{payload}\n</state>\n\n"
    "Question: {question}"
)


@dataclass(frozen=True)
class Prompt:
    """A complete prompt ready to send to a chat-completions API."""

    system: str
    user: str

    def messages(self) -> list[dict]:
        """OpenAI-compatible messages array. Anthropic uses the same
        shape with `system` passed separately, so the harness re-uses
        this list and pulls system out for Anthropic."""
        return [
            {"role": "system", "content": self.system},
            {"role": "user", "content": self.user},
        ]


def build(condition: str, payload: str, question: str) -> Prompt:
    """Build a prompt for the given condition.

    `condition` is "raw" or "acl"; mismatches raise so a typo can't
    silently degrade a comparison.
    """
    if condition == "raw":
        wrapper = _RAW_WRAPPER
    elif condition == "acl":
        wrapper = _ACL_WRAPPER
    else:
        raise ValueError(f"unknown condition: {condition!r}")
    return Prompt(system=SYSTEM_PROMPT, user=wrapper.format(payload=payload, question=question))
