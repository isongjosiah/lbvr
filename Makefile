# LBVR-Med — top-level Makefile
# See CLAUDE.md §8 for the experiment catalog (E1–E10, E6b, E9-multi, E-PROV).

SHELL := /usr/bin/env bash
.SHELLFLAGS := -euo pipefail -c

JAVA_BIN ?= /opt/homebrew/opt/openjdk@17/bin
SYNTHEA_DIR := eval/synthea/upstream
RESULTS_DIR := eval/results

.PHONY: help test test-go test-sol fmt lint hooks synthea synthea-% \
        bench-E1 bench-E2 bench-E3 bench-E4 bench-E5 bench-E6 bench-E6b \
        bench-E7 bench-E8 bench-E9 bench-E9-multi bench-E10 bench-E-PROV

help:
	@echo "LBVR-Med targets"
	@echo "  make test              — go test ./... + forge test"
	@echo "  make synthea-1k        — generate 1K patient FHIR corpus"
	@echo "  make synthea-10k       — 10K corpus"
	@echo "  make synthea-100k      — 100K corpus (overnight)"
	@echo "  make bench-E{n}        — run experiment En (see CLAUDE.md §8)"
	@echo "  make hooks             — install git pre-commit hook"

hooks:
	./scripts/install-hooks.sh

test: test-go test-sol

test-go:
	@if find . -name '*.go' -not -path './contracts/lib/*' | grep -q .; then \
	  go test ./... ; \
	else \
	  echo "no Go files yet" ; \
	fi

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
	@mkdir -p $(SYNTHEA_DIR)/output-$(NUM)
	cd $(SYNTHEA_DIR) && PATH="$(JAVA_BIN):$$PATH" ./run_synthea -p $(NUM) \
	    --exporter.fhir.export true \
	    --exporter.hospital.fhir.export false \
	    --exporter.practitioner.fhir.export false \
	    --exporter.baseDirectory ./output-$(NUM)
	@echo "✓ Synthea $(NUM) patients → $(SYNTHEA_DIR)/output-$(NUM)/fhir/"

# --- Experiment targets (stubs until D12) ---

define bench_stub
@mkdir -p $(RESULTS_DIR)/$(1)
@echo "E$(1) not yet implemented — see CLAUDE.md §8 and D12 plan."
@exit 1
endef

bench-E1:        ; $(call bench_stub,E1)
bench-E2:        ; $(call bench_stub,E2)
bench-E3:        ; $(call bench_stub,E3)
bench-E4:        ; $(call bench_stub,E4)
bench-E5:        ; $(call bench_stub,E5)
bench-E6:        ; $(call bench_stub,E6)
bench-E6b:       ; $(call bench_stub,E6b)
bench-E7:        ; $(call bench_stub,E7)
bench-E8:        ; $(call bench_stub,E8)
bench-E9:        ; $(call bench_stub,E9)
bench-E9-multi:  ; $(call bench_stub,E9-multi)
bench-E10:       ; $(call bench_stub,E10)
bench-E-PROV:    ; $(call bench_stub,E-PROV)
