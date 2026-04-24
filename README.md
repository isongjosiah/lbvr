# LBVR-Med

**Latency-Bounded Verifiable Retrieval fabric for safety-critical federated systems.**

IEEE ICUFN 2026 submission — deadline May 8, 2026.

For the authoritative project brief, architecture, scope decisions, threat model, evaluation protocol, and execution plan, **read [`CLAUDE.md`](./CLAUDE.md)**. Companion design specs live in [`docs/`](./docs/).

## Quickstart

```bash
# Prerequisites: Go 1.22+, Foundry, Java 17 (Synthea), Python 3.11+ (eval scripts)
cp .env.example .env           # fill in tier API keys + Cardona RPC
git submodule update --init    # pulls forge-std + openzeppelin-contracts into contracts/lib/
./scripts/install-hooks.sh     # wires pre-commit hooks (go vet, golangci-lint, forge test)

make ingest-1k                 # generate 1K Synthea corpus + ingest across 3 tiers
make bench-E2                  # run retrieval latency CDF experiment (see CLAUDE.md §8)
```

## Layout

See [`CLAUDE.md` §6](./CLAUDE.md#6-repo-layout).
