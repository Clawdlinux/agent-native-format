# Makefile for the acp-acl Python package.
#
# Targets:
#   make install      Editable install of the package + test extras
#   make test         Run pytest
#   make build        Build sdist + wheel into dist/
#   make publish-test Upload to TestPyPI (needs TWINE_PASSWORD env var)
#   make publish      Upload to PyPI (needs TWINE_PASSWORD env var)
#   make clean        Remove dist/ and *.egg-info
#
# Auth: set TWINE_USERNAME=__token__ and TWINE_PASSWORD=<pypi token>
# in your environment, or in ~/.pypirc.

PYTHON ?= python3

.PHONY: install test build publish-test publish clean lint

install:
	$(PYTHON) -m pip install -e '.[test]'

test:
	$(PYTHON) -m pytest tests/ -v

build: clean
	$(PYTHON) -m pip install -q --upgrade build twine
	$(PYTHON) -m build
	$(PYTHON) -m twine check dist/*

publish-test: build
	$(PYTHON) -m twine upload --repository testpypi dist/*
	@echo
	@echo "Verify install:"
	@echo "  pip install -i https://test.pypi.org/simple/ \\"
	@echo "      --extra-index-url https://pypi.org/simple/ acp-acl"

publish: build
	$(PYTHON) -m twine upload dist/*

clean:
	rm -rf dist/ build/ *.egg-info acp_acl/__pycache__ tests/__pycache__

lint:
	$(PYTHON) -m pip install -q --upgrade ruff
	$(PYTHON) -m ruff check acp_acl tests
