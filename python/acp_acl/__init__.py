"""acp-acl — pure-Python decoder for the Agent Context Language (ACL).

Lets agent runtimes (LangGraph, CrewAI, OpenAI tool-use loops, your own)
consume ACL documents without shelling out to the Go CLI. The decoder
is a faithful port of pkg/acl in <100 LOC and has no runtime
dependencies beyond the standard library.

Example:
    >>> import acp_acl
    >>> doc = acp_acl.decode(open("state.acl").read())
    >>> doc.directives["ns"]
    'payments'
    >>> doc.section("pods").rows[0].id
    'api-7f8d-aaa11'

Token counting (optional, requires tiktoken):
    >>> acp_acl.count_tokens("@ns x\\n", encoding="cl100k_base")
    5
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Iterator

__version__ = "0.1.0"
__all__ = ["Document", "Section", "Row", "decode", "count_tokens", "DecodeError"]


class DecodeError(ValueError):
    """Raised on malformed ACL input."""


# ─── data ────────────────────────────────────────────────────────────────────


@dataclass
class Row:
    """A single indented row inside a Section."""

    id: str = ""
    count: int = 0
    fields: dict[str, str] = field(default_factory=dict)
    flags: list[str] = field(default_factory=list)


@dataclass
class Section:
    """A header line plus zero or more rows."""

    name: str
    summary: str = ""
    rows: list[Row] = field(default_factory=list)


@dataclass
class Document:
    """A complete ACL document."""

    directives: dict[str, str] = field(default_factory=dict)
    sections: list[Section] = field(default_factory=list)

    def section(self, name: str) -> Section | None:
        """Return the first section with the given name, or None."""
        for s in self.sections:
            if s.name == name:
                return s
        return None

    def __iter__(self) -> Iterator[Section]:
        return iter(self.sections)


# ─── decoder ─────────────────────────────────────────────────────────────────

_INDENT = "  "


def decode(data: str) -> Document:
    """Parse a canonical ACL document.

    Strips ``# `` comment lines, treats blank lines as section
    separators, and rejects tab indentation. Raises ``DecodeError``
    with a 1-based line number on malformed input.
    """
    doc = Document()
    cur: Section | None = None
    directives_allowed = True
    for lineno, raw in enumerate(data.splitlines(), start=1):
        if "\t" in raw:
            raise DecodeError(f"line {lineno}: tab characters are forbidden")
        if raw.startswith("# ") or raw == "#":
            continue
        if raw == "":
            cur = None
            continue
        if raw.startswith("@"):
            if not directives_allowed:
                raise DecodeError(f"line {lineno}: directive after section")
            key, _, value = raw[1:].partition(" ")
            if not key:
                raise DecodeError(f"line {lineno}: empty directive key")
            doc.directives[key] = value
            continue
        directives_allowed = False
        if raw.startswith(_INDENT):
            if cur is None:
                raise DecodeError(f"line {lineno}: row outside any section")
            cur.rows.append(_parse_row(raw[len(_INDENT) :], lineno))
            continue
        if raw.startswith(" "):
            raise DecodeError(f"line {lineno}: indentation must be exactly two spaces")
        # Section header.
        name, _, summary = raw.partition(" ")
        cur = Section(name=name, summary=summary)
        doc.sections.append(cur)
    return doc


# ─── token counting (optional helper) ───────────────────────────────────────


def count_tokens(text: str, encoding: str = "cl100k_base") -> int:
    """Return the token count of ``text`` under the given OpenAI tokenizer.

    Requires the optional ``tiktoken`` extra:
        pip install acp-acl[tokens]
    """
    try:
        import tiktoken
    except ImportError as e:  # pragma: no cover
        raise ImportError(
            "count_tokens requires the optional `tiktoken` dependency. "
            "Install it with: pip install acp-acl[tokens]"
        ) from e
    enc = tiktoken.get_encoding(encoding)
    return len(enc.encode(text))


# ─── internals ───────────────────────────────────────────────────────────────


def _parse_row(body: str, lineno: int) -> Row:
    tokens = _tokenise(body, lineno)
    if not tokens:
        raise DecodeError(f"line {lineno}: empty row")
    row = Row()
    start = 0
    first = tokens[0]
    if "=" not in first and not _is_count(first) and first not in ("!", "!!", "?"):
        row.id = first
        start = 1
    if start < len(tokens) and _is_count(tokens[start]):
        try:
            row.count = int(tokens[start][1:])
        except ValueError as e:
            raise DecodeError(f"line {lineno}: bad count token {tokens[start]!r}") from e
        start += 1
    for tok in tokens[start:]:
        eq = tok.find("=")
        if eq >= 0:
            key = tok[:eq]
            value = _unquote(tok[eq + 1 :], lineno)
            row.fields[key] = value
        else:
            row.flags.append(tok)
    return row


def _is_count(t: str) -> bool:
    return len(t) >= 2 and t[0] == "x" and t[1:].isdigit()


def _tokenise(s: str, lineno: int) -> list[str]:
    """Split a row body on spaces, treating backtick-quoted spans as
    single tokens. Doubled backticks inside a quoted span unescape to
    a single backtick."""
    out: list[str] = []
    cur: list[str] = []
    in_quote = False
    i = 0
    while i < len(s):
        c = s[i]
        if in_quote:
            if c == "`":
                if i + 1 < len(s) and s[i + 1] == "`":
                    cur.append("`")
                    i += 2
                    continue
                in_quote = False
                cur.append("`")
            else:
                cur.append(c)
            i += 1
            continue
        if c == " ":
            if cur:
                out.append("".join(cur))
                cur = []
        elif c == "`":
            in_quote = True
            cur.append("`")
        else:
            cur.append(c)
        i += 1
    if in_quote:
        raise DecodeError(f"line {lineno}: unterminated backtick quote")
    if cur:
        out.append("".join(cur))
    return out


def _unquote(v: str, lineno: int) -> str:
    if len(v) >= 2 and v[0] == "`" and v[-1] == "`":
        return v[1:-1]
    if "`" in v:
        raise DecodeError(f"line {lineno}: stray backtick in value {v!r}")
    return v
