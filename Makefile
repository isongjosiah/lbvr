# LBVR-Med — top-level Makefile
# See CLAUDE.md §8 for the experiment catalog (E1–E10, E6b, E9-multi, E-PROV).

SHELL := /usr/bin/env bash
.SHELLFLAGS := -euo pipefail -c

# JAVA_HOME must point at a JDK (with javac), not a JRE. Synthea's Gradle build
# rejects JRE-only installs. Resolution order: user override > ~/.local/opt/jdk-21
# (portable install) > system default > empty. See README for install instructions.
#
# macOS Homebrew example: JAVA_HOME=/opt/homebrew/opt/openjdk@17 make synthea-1k
LOCAL_JDK_DIR := $(HOME)/.local/opt/jdk-21
JAVA_HOME ?= $(if $(wildcard $(LOCAL_JDK_DIR)/bin/javac),$(LOCAL_JDK_DIR),$(if $(wildcard /usr/lib/jvm/default-java/bin/javac),/usr/lib/jvm/default-java,))
JAVA_BIN  ?= $(if $(JAVA_HOME),$(JAVA_HOME)/bin,)
SYNTHEA_DIR := eval/synthea/upstream
RESULTS_DIR := eval/results
VENV_DIR    := eval/.venv
VENV_PY     := $(VENV_DIR)/bin/python

.PHONY: help test test-go test-merkle test-crypto test-tiers test-tiers-integration \
        test-registry test-client test-gateway test-provenance \
        test-sol fmt lint hooks eval-deps \
        synthea-1k-bg synthea-10k-bg synthea-100k-bg synthea-status-% \
        validate-synthea-% plot-synthea-% \
        bench-E1 bench-E2 bench-E3 bench-E4 bench-E5 bench-E6 bench-E6b \
        bench-E7 bench-E8 bench-E9 bench-E9-multi bench-E10 bench-E-PROV

help:
	@echo "LBVR-Med targets"
	@echo "  make test                   — go test ./... + forge test"
	@echo "  make test-go                — go test -race ./..."
	@echo "  make test-merkle            — go test -race ./internal/merkle/..."
	@echo "  make test-crypto            — go test -race ./internal/crypto/..."
	@echo "  make test-tiers             — go test -race ./internal/tiers/... (no network)"
	@echo "  make test-tiers-integration — go test -tags=integration ./internal/tiers/... (real keys)"
	@echo "  make test-registry          — go test -race ./internal/registry/..."
	@echo "  make test-client            — go test -race ./cmd/client/... (D6 ingest CLI)"
	@echo "  make test-gateway           — go test -race ./cmd/gateway/... (D8 retrieval gateway)"
	@echo "  make test-provenance        — go test -race ./internal/provenance/... ./cmd/verifier/... (D10/D11)"
	@echo "  make synthea-1k             — generate 1K patient FHIR corpus"
	@echo "  make synthea-10k            — 10K corpus"
	@echo "  make synthea-100k           — 100K corpus (foreground, ~3h)"
	@echo "  make synthea-100k-bg        — 100K corpus, detached (survives shell exit)"
	@echo "  make synthea-status-100k    — check status of a detached run"
	@echo "  make validate-synthea-1k    — validate + size-stats on 1K corpus"
	@echo "  make validate-synthea-100k  — validate + size-stats on 100K corpus"
	@echo "  make plot-synthea-1k        — plot size-distribution PDF for 1K corpus"
	@echo "  make plot-synthea-100k      — plot size-distribution PDF for 100K corpus"
	@echo "  make eval-deps              — set up eval/.venv with matplotlib + numpy"
	@echo "  make bench-E{n}             — run experiment En (see CLAUDE.md §8)"
	@echo "  make hooks                  — install git pre-commit hook"

hooks:
	./scripts/install-hooks.sh

test: test-go test-sol

test-go:
	@if find . -name '*.go' -not -path './contracts/lib/*' | grep -q .; then \
	  go test -race ./... ; \
	else \
	  echo "no Go files yet" ; \
	fi

test-merkle:
	go test -race ./internal/merkle/...

test-crypto:
	go test -race ./internal/crypto/...

# tier clients use httptest mocks only — never hits real Pinata/Filebase/Irys.
test-tiers:
	go test -race -count=1 ./internal/tiers/...

# integration tier tests hit live APIs; requires .env with real keys.
# Kept out of the default `test-go` target so CI stays offline.
test-tiers-integration:
	go test -race -count=1 -tags=integration ./internal/tiers/...

test-registry:
	go test -race -count=1 ./internal/registry/...

# D6 ingest CLI end-to-end (in-memory tiers + Mock registry, no network).
test-client:
	go test -race -count=1 ./cmd/client/...

# D8 retrieval gateway (in-memory tiers + Mock registry + sidecar, no network).
test-gateway:
	go test -race -count=1 ./cmd/gateway/...

# D10/D11 cryptographic provenance (PROV-JSON + JCS + BLS quorum + verifier CLI).
test-provenance:
	go test -race -count=1 ./internal/provenance/... ./cmd/verifier/...

test-sol:
	@if find contracts/src contracts/test -name '*.sol' 2>/dev/null | grep -q .; then \
	  forge test --root contracts ; \
	else \
	  echo "no Solidity files yet" ; \
	fi

fmt:
	@if find . -name '*.go' -not -path './contracts/lib/*' | grep -q .; then gofmt -w . ; fi
	@if find contracts/src contracts/test -name '*.sol' 2>/dev/null | grep -q .; then forge fmt --root contracts ; fi

lint:
	@command -v golangci-lint >/dev/null || { echo "install golangci-lint: brew install golangci-lint"; exit 1; }
	@if find . -name '*.go' -not -path './contracts/lib/*' | grep -q .; then golangci-lint run ./... ; fi

# --- Synthea corpus generation ---

$(SYNTHEA_DIR):
	git clone --depth 1 https://github.com/synthetichealth/synthea $(SYNTHEA_DIR)

synthea-1k:   NUM = 1000
synthea-10k:  NUM = 10000
synthea-100k: NUM = 100000

synthea-1k synthea-10k synthea-100k: $(SYNTHEA_DIR)
	@if [ -z "$(JAVA_HOME)" ] || [ ! -x "$(JAVA_HOME)/bin/javac" ]; then \
	  echo "✗ No JDK found. Install one or set JAVA_HOME. See README §Prerequisites."; \
	  echo "  Linux quick path:  curl -L adoptium.net Temurin 21 to $(LOCAL_JDK_DIR)"; \
	  exit 1; \
	fi
	@mkdir -p $(SYNTHEA_DIR)/output-$(NUM)
	@cd $(SYNTHEA_DIR) && \
	  export JAVA_HOME="$(JAVA_HOME)" PATH="$(JAVA_BIN):$$PATH" && \
	  ./run_synthea -p $(NUM) \
	    --exporter.fhir.export true \
	    --exporter.hospital.fhir.export false \
	    --exporter.practitioner.fhir.export false \
	    --exporter.pretty_print false \
	    --exporter.baseDirectory ./output-$(NUM)
	@echo "✓ Synthea $(NUM) patients → $(SYNTHEA_DIR)/output-$(NUM)/fhir/"

# --- Corpus validation + size-distribution artifacts ---

$(VENV_PY):
	python3 -m venv $(VENV_DIR)
	$(VENV_PY) -m pip install --quiet --upgrade pip
	$(VENV_PY) -m pip install --quiet matplotlib numpy

eval-deps: $(VENV_PY)
	@echo "✓ eval venv ready at $(VENV_DIR)"

validate-synthea-1k:   NUM = 1000
validate-synthea-10k:  NUM = 10000
validate-synthea-100k: NUM = 100000
plot-synthea-1k:       NUM = 1000
plot-synthea-10k:      NUM = 10000
plot-synthea-100k:     NUM = 100000

validate-synthea-1k validate-synthea-10k validate-synthea-100k:
	@mkdir -p $(RESULTS_DIR)/synthea-$(NUM)
	python3 eval/scripts/validate_synthea.py \
	  $(SYNTHEA_DIR)/output-$(NUM)/fhir/ \
	  --stats-out $(RESULTS_DIR)/synthea-$(NUM)/validation.json \
	  --corpus-size $(NUM)

synthea-1k-bg:   ; ./scripts/synthea-bg.sh 1k
synthea-10k-bg:  ; ./scripts/synthea-bg.sh 10k
synthea-100k-bg: ; ./scripts/synthea-bg.sh 100k

synthea-status-1k:   ; ./scripts/synthea-bg-status.sh 1k
synthea-status-10k:  ; ./scripts/synthea-bg-status.sh 10k
synthea-status-100k: ; ./scripts/synthea-bg-status.sh 100k

plot-synthea-1k plot-synthea-10k plot-synthea-100k: $(VENV_PY)
	@mkdir -p $(RESULTS_DIR)/synthea-$(NUM)
	$(VENV_PY) eval/scripts/plot_size_distribution.py \
	  $(SYNTHEA_DIR)/output-$(NUM)/fhir/ \
	  --out-dir $(RESULTS_DIR)/synthea-$(NUM) \
	  --corpus-size $(NUM)

# --- Experiment targets (stubs until D12) ---

define bench_stub
@mkdir -p $(RESULTS_DIR)/$(1)
@echo "E$(1) not yet implemented — see CLAUDE.md §8 and D12 plan."
@exit 1
endef

# Per-bench knobs. Override on the command line: `make bench-E5 E5_REPS=200`.
E5_REPS ?= 1000
E5_SEED ?= 42

bench-E1:        ; $(call bench_stub,E1)
bench-E2:        ; $(call bench_stub,E2)
bench-E3:        ; $(call bench_stub,E3)
bench-E4:        ; $(call bench_stub,E4)
bench-E5:
	@mkdir -p $(RESULTS_DIR)/E5
	go run ./cmd/bench/e5 -reps $(E5_REPS) -seed $(E5_SEED) -out-dir $(RESULTS_DIR)/E5
bench-E6:        ; $(call bench_stub,E6)
bench-E6b:       ; $(call bench_stub,E6b)
bench-E7:        ; $(call bench_stub,E7)
bench-E8:        ; $(call bench_stub,E8)
bench-E9:        ; $(call bench_stub,E9)
bench-E9-multi:  ; $(call bench_stub,E9-multi)
bench-E10:       ; $(call bench_stub,E10)
bench-E-PROV:    ; $(call bench_stub,E-PROV)
