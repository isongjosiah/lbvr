# LBVR-Med

**Latency-Bounded Verifiable Retrieval fabric for safety-critical federated systems.**

IEEE ICUFN 2026 submission — deadline May 8, 2026.

For the authoritative project brief, architecture, scope decisions, threat model, evaluation protocol, and execution plan, **read [`CLAUDE.md`](./CLAUDE.md)**. Companion design specs live in [`docs/`](./docs/).

## Quickstart

```bash
# Prerequisites: Go 1.22+, Foundry, JDK 21 (Synthea compiles with gradle),
# Python 3.11+ (eval scripts via local venv).
cp .env.example .env           # fill in tier API keys + Cardona RPC
git submodule update --init    # pulls forge-std + openzeppelin-contracts into contracts/lib/
./scripts/install-hooks.sh     # wires pre-commit hooks (go vet, golangci-lint, forge test)
make eval-deps                 # bootstraps eval/.venv with matplotlib+numpy (no sudo)

# Generate Synthea FHIR R4 corpora (wall-clock: 1K ≈ 1m, 100K ≈ 2–3h on 12 cores):
make synthea-1k
make validate-synthea-1k       # structural + size-distribution JSON → eval/results/
make plot-synthea-1k           # size_distribution.{pdf,png} + sizes.csv

make bench-E2                  # retrieval latency CDF (stub until D12 — see CLAUDE.md §8)
```

### JDK note (Linux)

Synthea's Gradle build requires a JDK (not a JRE). The Makefile auto-picks
`~/.local/opt/jdk-21` if present. Portable install (no sudo):

```bash
mkdir -p ~/.local/opt && cd ~/.local/opt
curl -sSL "https://api.adoptium.net/v3/binary/latest/21/ga/linux/x64/jdk/hotspot/normal/eclipse?project=jdk" | tar xz
ln -sfn jdk-21.0.11+10 jdk-21   # or the current version directory
```

macOS Homebrew users can instead set `JAVA_HOME=/opt/homebrew/opt/openjdk@17 make synthea-1k`.

## Layout

See [`CLAUDE.md` §6](./CLAUDE.md#6-repo-layout).
