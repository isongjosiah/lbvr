# LBVR-Med architecture

Landing page for the storage fabric. CLAUDE.md §4 is the authoritative architectural brief; this
doc fleshes out individual subsystems as they land. Companion specs take precedence within their
own scope (see §13.1 of CLAUDE.md): [`erasure-design.md`](./erasure-design.md) owns RS(2,3)
details, [`provenance-spec.md`](./provenance-spec.md) owns PROV-JSON + BLS signing.

## 1. Layered view (mirrors CLAUDE.md §4.1)

```
L5  Provenance      — PROV-JSON generation, BLS quorum signing, on-chain anchor (D10-D11)
L4  Auditor         — contracts/src/AuditorLog.sol (stub; wired in D12)
L3  Retrieval       — cmd/gateway/, internal/erasure/ (D7-D8)
L2  Placement       — cmd/orchestrator/ (emits layout + migrates; D11-D12)
L1  Ingest client   — cmd/client/, internal/{merkle, crypto, tiers, registry}  (D3-D6)
```

Current state (D8, 2026-04-25):
- L1 substrate packages (`internal/merkle`, `internal/crypto`) landed; `internal/tiers/*` in
  flight (D4 agent).
- L4/on-chain: `CIDRegistry.sol` landed (not yet deployed — user action: `foundryup` +
  `forge install` + `forge script Deploy`).
- L2, L3, L5: not started.

## 2. Ingest data flow (CLAUDE.md §4.2)

```
FHIR bundle (bytes)
  │
  ├─► merkle.Build       → Merkle tree over 16-KiB SHA-256 chunks; root = M_root, count = N
  │
  ├─► crypto.SealChunk   → per-chunk AES-256-GCM (random nonce || ct || tag)
  │
  ├─► erasure.Encode     → RS(2,3): two data shards D0,D1 + one parity P0 (D7)
  │
  ├─► tiers.{Pinata,Filebase,Irys}.Put
  │      D0 → hot (Pinata), D1 → warm (Filebase), P0 → cold (Arweave/Irys)
  │
  └─► CIDRegistry.registerBundle(bundleId, M_root, N, [(cid,tier)×3], policyId)
```

Key invariants bound on-chain at ingest (see §4 below for the *why*):
- 3 shards exactly (RS(2,3) conference-scope).
- `numChunks` stored alongside `merkleRoot` so Merkle proofs cannot be re-interpreted against
  a crafted tree width.

## 3. Retrieval data flow (CLAUDE.md §4.3)

```
GET /bundle/{bundleId}
  │
  ├─► CIDRegistry.getRecord → M_root, numChunks, shardLayout, tier_history
  │
  ├─► tiers.*.Get in parallel (D0, D1, P0)
  │      - Fast path : D0+D1 return under SLO → direct merge
  │      - Slow path : any one missing → erasure.Decode from any 2 of 3 shards
  │      - 2+ missing: error, emit breach event
  │
  ├─► Merkle.Verify chunks against (M_root, numChunks)
  │
  └─► emit retrieval receipt + PROV-JSON (L5, D10-D11)
```

## 4. Integrity invariants

**Merkle-tree width (N) is authenticated on-chain.** `internal/merkle` uses Bitcoin-style
odd-width duplication at each level — this is the CVE-2012-2459 pattern. It is safe here only
because `CIDRegistry.BundleRecord.numChunks` binds N to the bundle at registration, and
`Verify` requires N as an input. Changing the duplication rule (e.g., RFC 6962 distinguished
empty node) would require a contract schema bump.

**Shard count is exactly 3.** `CIDRegistry` rejects any other value in both `registerBundle`
and `updateShardLayout`. Relaxed in the journal extension for RS(3,5).

**Zero-byte bundles are invalid.** `internal/merkle.Build` accepts an empty reader and
returns an all-zeroes root + `NumChunks()==0`; the D6 ingest CLI must reject these explicitly
before calling `registerBundle` (which will revert on `numChunks == 0` anyway, but clearer at
the CLI layer).

## 5. Subsystem map (by package)

| Package | Status | Authoritative spec |
|---|---|---|
| `internal/merkle` | landed (D3) | CLAUDE.md §4.2 |
| `internal/crypto` | landed (D3) | CLAUDE.md §4.2 |
| `internal/config` | D4 in flight | CLAUDE.md §7 (env vars) + `.env.example` |
| `internal/tiers/*` | D4 in flight | CLAUDE.md §4.2, §4.5 placement policy |
| `internal/erasure` | not started | [`erasure-design.md`](./erasure-design.md) |
| `internal/registry` | not started (Go bindings via `abigen`) | `contracts/src/CIDRegistry.sol` |
| `internal/por` | not started | CLAUDE.md §4.4 |
| `internal/provenance` | not started | [`provenance-spec.md`](./provenance-spec.md) |
| `internal/telemetry` | not started | Prometheus + zap per §7 |
| `cmd/client` | not started | ingest CLI, D6 |
| `cmd/gateway` | not started | retrieval gateway, D8 |
| `cmd/orchestrator` | not started | placement daemon, D11-D12 |
| `cmd/verifier` | not started | standalone provenance verifier CLI, D11 |
| `cmd/bench` | not started | E1–E10 harness, D12-D15 |

## 6. On-chain layer (`contracts/src/`)

| Contract | Status | Purpose |
|---|---|---|
| `CIDRegistry.sol` | landed (D5, not deployed) | Bundle records: merkleRoot, numChunks, shards, owner, policy. Role-gated `updateShardLayout` for PoR-driven migration. |
| `PoRVerifier.sol` | D12 | Records challenge/response verdicts; holds `MIGRATOR_ROLE` on `CIDRegistry`. |
| `AuditorLog.sol` | D12 | EU AI Act Art. 12 event log + provenance-root anchor (D11). |

Deployment target is Polygon zkEVM Cardona (chain ID 2442). The deployer key is retained as the
default admin on `CIDRegistry` until `PoRVerifier` is deployed; then `MIGRATOR_ROLE` is granted
to the verifier and revoked from the deployer.

## 7. Evaluation harness

`cmd/bench/` (not yet built) exposes one `make bench-E{n}` per §8 experiment in CLAUDE.md.
Raw results land in `eval/results/E{n}/` with an `env.json` fingerprint; post-processing
scripts in `eval/scripts/` emit figures into `paper/figures/`.

Today the only eval artefacts are the Synthea D2 outputs in `eval/results/synthea-{1000,100000}/`.

## 8. Out of scope

See CLAUDE.md §11 for the complete "do-NOT-do" list. Highlights: no new consensus protocol,
no federated-learning scope in the conference paper, no Filecoin FVM deals, no C2PA, no public
IPFS gateway as a positive baseline.
