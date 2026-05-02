# Paper drafts

This directory holds the source for the arxiv preprint that accompanies the
ACP reference implementation in this repository.

## Contents

- `acp.md` — readable Markdown source. The canonical text.
- `acp.tex` — LaTeX source suitable for arxiv submission. Hand-mirrored
  from `acp.md`. Both must agree (`scripts/check_paper.py` enforces this
  at CI time).
- `figures/` — charts copied from `results/charts/` so the paper builds
  hermetically.
- `references.bib` — BibTeX bibliography.
- `Makefile` — `make pdf` invokes `pdflatex` if available locally.

## Building the PDF

```bash
make -C paper pdf       # requires pdflatex / texlive-latex-recommended
```

The CI workflow `paper.yml` builds the PDF on every PR using a
TeX Live image; the resulting `acp.pdf` is uploaded as a build artifact
so reviewers don't need a local LaTeX install.

## arxiv submission checklist

See `docs/paper-plan.md` (Phase 5 / public launch).
