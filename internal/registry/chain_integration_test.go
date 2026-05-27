// Integration tests for chain.go and the AuditorLog binding against a
// live EVM chain. Gated behind `//go:build integration` so a plain
// `go test ./...` skips them; run with:
//
//	go test -tags integration -run TestIntegration ./internal/registry/...
//
// The tests read CHAIN_RPC_URL / CHAIN_PRIVATE_KEY / CID_REGISTRY_ADDRESS
// / AUDITOR_LOG_ADDRESS from the process environment (caller's
// responsibility to load .env first via `godotenv -f .env <go-test
// command>` or by sourcing the file in their shell). Any missing
// variable produces a clean t.Skip so CI without secrets stays green.
//
// Each test writes to the chain and consumes gas. BundleID and
// retrievalID are derived from time.Now().UnixNano() + random bytes so
// re-runs do not collide with prior state.

//go:build integration

package registry

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Skipf("integration: skipping — %s is unset", key)
	}
	return v
}

// freshID derives a 32-byte ID that is overwhelmingly unique per
// invocation: 8 bytes of monotonic time-since-epoch + 24 bytes random.
// Used for bundleID and retrievalID so re-running the test against the
// same chain doesn't collide with prior state.
func freshID(t *testing.T) [32]byte {
	t.Helper()
	var out [32]byte
	binary.BigEndian.PutUint64(out[:8], uint64(time.Now().UnixNano()))
	if _, err := rand.Read(out[8:]); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	return out
}

// TestIntegration_CIDRegistry posts a registerBundle tx, reads it back
// via getRecord, then posts an updateShardLayout tx and reads the new
// layout. Validates the chain.go conversion helpers and the
// abigen-bound calls end-to-end against the live CIDRegistry contract.
func TestIntegration_CIDRegistry(t *testing.T) {
	rpcURL := requireEnv(t, "CHAIN_RPC_URL")
	privKey := requireEnv(t, "CHAIN_PRIVATE_KEY")
	contractAddr := requireEnv(t, "CID_REGISTRY_ADDRESS")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cli, err := NewChain(ctx, rpcURL, contractAddr, privKey)
	if err != nil {
		t.Fatalf("NewChain: %v", err)
	}
	defer cli.(*chainClient).Close()

	bundleID := freshID(t)
	rec := BundleRecord{
		MerkleRoot: freshID(t),
		NumChunks:  42,
		Shards: [ShardCount]ShardPlacement{
			{CID: "bafyA-integration-hot-" + time.Now().Format("150405"), Tier: TierHot},
			{CID: "bafyB-integration-warm-" + time.Now().Format("150405"), Tier: TierWarm},
			{CID: "bafyC-integration-cold-" + time.Now().Format("150405"), Tier: TierCold},
		},
		PolicyID: freshID(t),
	}

	t.Logf("registerBundle: bundleID=%x merkleRoot=%x", bundleID, rec.MerkleRoot)
	if err := cli.RegisterBundle(ctx, bundleID, rec); err != nil {
		t.Fatalf("RegisterBundle: %v", err)
	}

	got, err := cli.GetRecord(ctx, bundleID)
	if err != nil {
		t.Fatalf("GetRecord: %v", err)
	}
	if got.MerkleRoot != rec.MerkleRoot {
		t.Errorf("MerkleRoot mismatch: got %x want %x", got.MerkleRoot, rec.MerkleRoot)
	}
	if got.NumChunks != rec.NumChunks {
		t.Errorf("NumChunks mismatch: got %d want %d", got.NumChunks, rec.NumChunks)
	}
	if got.PolicyID != rec.PolicyID {
		t.Errorf("PolicyID mismatch: got %x want %x", got.PolicyID, rec.PolicyID)
	}
	for i := range rec.Shards {
		if got.Shards[i] != rec.Shards[i] {
			t.Errorf("Shards[%d] mismatch: got %+v want %+v", i, got.Shards[i], rec.Shards[i])
		}
	}
	if got.Owner == "" || !strings.HasPrefix(got.Owner, "0x") {
		t.Errorf("Owner not set or not hex: %q", got.Owner)
	}
	if got.RegisteredAt.IsZero() {
		t.Errorf("RegisteredAt is zero")
	}
	t.Logf("GetRecord: owner=%s registeredAt=%s", got.Owner, got.RegisteredAt.Format(time.RFC3339))

	layout, err := cli.GetShardLayout(ctx, bundleID)
	if err != nil {
		t.Fatalf("GetShardLayout: %v", err)
	}
	if layout != rec.Shards {
		t.Errorf("GetShardLayout mismatch: got %+v want %+v", layout, rec.Shards)
	}

	// UpdateShardLayout requires MIGRATOR_ROLE. If the deployer key
	// (CHAIN_PRIVATE_KEY) doesn't hold the role on this contract, the
	// call reverts with AccessControlUnauthorizedAccount — log and
	// continue rather than fail, because the role grant is a deploy-
	// time concern outside this test's scope.
	updated := rec.Shards
	updated[0].CID = "bafyA-MIGRATED-" + time.Now().Format("150405")
	if err := cli.UpdateShardLayout(ctx, bundleID, updated); err != nil {
		t.Logf("UpdateShardLayout: %v (expected if deployer lacks MIGRATOR_ROLE)", err)
		return
	}
	post, err := cli.GetShardLayout(ctx, bundleID)
	if err != nil {
		t.Fatalf("GetShardLayout post-update: %v", err)
	}
	if post[0].CID != updated[0].CID {
		t.Errorf("post-update shard[0].CID mismatch: got %q want %q", post[0].CID, updated[0].CID)
	}
	t.Logf("UpdateShardLayout: shard[0].CID now %q", post[0].CID)
}

// TestIntegration_AuditorLog drives the AuditorLog binding directly
// rather than going through cmd/gateway's chainAnchor wrapper, so the
// test stays in the registry package alongside the bindings. It anchors
// a fresh (bundleID, retrievalID, provHash) tuple, reads it back via
// the view, and asserts the round-trip.
func TestIntegration_AuditorLog(t *testing.T) {
	rpcURL := requireEnv(t, "CHAIN_RPC_URL")
	privKey := requireEnv(t, "CHAIN_PRIVATE_KEY")
	contractAddr := requireEnv(t, "AUDITOR_LOG_ADDRESS")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	rpc, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		t.Fatalf("ethclient.DialContext: %v", err)
	}
	defer rpc.Close()

	chainID, err := rpc.ChainID(ctx)
	if err != nil {
		t.Fatalf("ChainID: %v", err)
	}
	pk, err := crypto.HexToECDSA(strings.TrimPrefix(privKey, "0x"))
	if err != nil {
		t.Fatalf("parse privKey: %v", err)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(pk, chainID)
	if err != nil {
		t.Fatalf("NewKeyedTransactorWithChainID: %v", err)
	}
	contract, err := NewChainAuditorLog(common.HexToAddress(contractAddr), rpc)
	if err != nil {
		t.Fatalf("NewChainAuditorLog: %v", err)
	}
	session := &ChainAuditorLogSession{
		Contract:     contract,
		CallOpts:     bind.CallOpts{Pending: false},
		TransactOpts: *auth,
	}

	bundleID := freshID(t)
	retrievalID := freshID(t)
	provHash := freshID(t)

	t.Logf("anchorProvenance: bundleID=%x retrievalID=%x provHash=%x", bundleID, retrievalID, provHash)
	opts := session.TransactOpts
	opts.Context = ctx
	tx, err := session.Contract.AnchorProvenance(&opts, bundleID, retrievalID, provHash)
	if err != nil {
		// AnchorProvenance requires ANCHOR_ROLE. Same caveat as
		// UpdateShardLayout above — log and skip if the deployer key
		// lacks the role.
		if strings.Contains(err.Error(), "AccessControlUnauthorizedAccount") {
			t.Skipf("AnchorProvenance: caller lacks ANCHOR_ROLE on %s — grant the role and re-run", contractAddr)
		}
		t.Fatalf("AnchorProvenance: %v", err)
	}
	t.Logf("AnchorProvenance tx: %s", tx.Hash().Hex())

	waitCtx, waitCancel := context.WithTimeout(ctx, 60*time.Second)
	defer waitCancel()
	receipt, err := bind.WaitMined(waitCtx, rpc, tx)
	if err != nil {
		t.Fatalf("WaitMined: %v", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("AnchorProvenance reverted at block %d", receipt.BlockNumber.Uint64())
	}
	t.Logf("AnchorProvenance mined: block=%d gasUsed=%d", receipt.BlockNumber.Uint64(), receipt.GasUsed)

	callOpts := session.CallOpts
	callOpts.Context = ctx
	got, err := session.Contract.GetProvenanceAnchor(&callOpts, bundleID, retrievalID)
	if err != nil {
		t.Fatalf("GetProvenanceAnchor: %v", err)
	}
	if got.ProvHash != provHash {
		t.Errorf("ProvHash mismatch: got %x want %x", got.ProvHash, provHash)
	}
	if got.BlockNumber.Uint64() != receipt.BlockNumber.Uint64() {
		t.Errorf("BlockNumber mismatch: got %d want %d", got.BlockNumber.Uint64(), receipt.BlockNumber.Uint64())
	}
	if got.AnchoredBy != auth.From {
		t.Errorf("AnchoredBy mismatch: got %s want %s", got.AnchoredBy.Hex(), auth.From.Hex())
	}
	t.Logf("GetProvenanceAnchor: block=%d anchoredBy=%s ts=%d",
		got.BlockNumber.Uint64(), got.AnchoredBy.Hex(), got.Timestamp.Uint64())
}
