# Build the arxiv preprint PDF.
#
# Requires: pdflatex + bibtex (texlive-latex-recommended on Debian/Ubuntu;
# `brew install --cask mactex-no-gui` on macOS).
# CI builds the PDF inside a TeX Live container; see
# .github/workflows/paper.yml.

PAPER ?= acp
LATEX ?= pdflatex -halt-on-error -interaction=nonstopmode
BIBTEX ?= bibtex

.PHONY: pdf clean

pdf: $(PAPER).pdf

$(PAPER).pdf: $(PAPER).tex references.bib figures/tokens.png figures/round_trips.png figures/reduction.png
	$(LATEX) $(PAPER)
	$(BIBTEX) $(PAPER)
	$(LATEX) $(PAPER)
	$(LATEX) $(PAPER)

clean:
	rm -f $(PAPER).aux $(PAPER).log $(PAPER).out $(PAPER).toc \
	      $(PAPER).bbl $(PAPER).blg $(PAPER).pdf
