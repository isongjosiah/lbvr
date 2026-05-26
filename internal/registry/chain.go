// chain.go is the eth-client-backed implementation of the registry.Client
// interface, talking to the deployed CIDRegistry contract via the
// abigen-generated bindings in contract_binding_cidregistry.go.
//
// Wiring lives here so the rest of the package (Mock, value types,
// sentinels, ValidateShards) stays free of any go-ethereum dependency.
// Construction dials the RPC, fetches the chain ID, builds a keyed
// transactor from the configured private key, and binds the contract.
// All four Client methods funnel through the cached session.

package registry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// txTimeout bounds how long a single write tx may sit in the mempool
// before chain.go gives up waiting for inclusion. PureChain produces
// ~2-3s blocks; 60s comfortably covers both inclusion and one round of
// re-org chop. Callers can still cancel earlier via ctx.
const txTimeout = 60 * time.Second

// ErrChainNotImplemented retained as an exported sentinel for any
// transitional caller that still branches on it. New code should rely
// on the wrapped sentinels (ErrAlreadyRegistered, ErrNotFound, etc.)
// returned by the real implementation below.
var ErrChainNotImplemented = errors.New(
	"registry: chain client not implemented; run forge build + abigen first",
)

// chainClient is the eth-client-backed Client. The session bundles the
// abigen contract instance with the per-call CallOpts/TransactOpts so
// methods can copy and customise per-invocation (ctx, value, gas) without
// mutating shared state.
type chainClient struct {
	rpc     *ethclient.Client
	session *ChainCIDRegistrySession
}

// NewChain dials rpcURL, parses privKey, fetches the chain ID, and binds
// the CIDRegistry contract at contractAddr. The chain ID is read from
// the RPC rather than threaded through the signature so callers don't
// need to know it — PureChain reports 900520900520, Cardona reports 2442,
// etc., and the keyed transactor adapts automatically.
//
// Returns the existing sentinel string errors on empty args. Wraps RPC,
// key-parse, and bind failures with %w so errors.Is/As keeps working.
func NewChain(ctx context.Context, rpcURL, contractAddr, privKey string) (Client, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if rpcURL == "" {
		return nil, errors.New("registry: rpcURL is empty")
	}
	if contractAddr == "" {
		return nil, errors.New("registry: contractAddr is empty")
	}
	if privKey == "" {
		return nil, errors.New("registry: privKey is empty")
	}
	if !common.IsHexAddress(contractAddr) {
		return nil, fmt.Errorf("registry: contractAddr %q is not a valid hex address", contractAddr)
	}

	rpc, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("registry: dial RPC %q: %w", rpcURL, err)
	}

	chainID, err := rpc.ChainID(ctx)
	if err != nil {
		rpc.Close()
		return nil, fmt.Errorf("registry: fetch chain ID: %w", err)
	}

	pk, err := crypto.HexToECDSA(strings.TrimPrefix(privKey, "0x"))
	if err != nil {
		rpc.Close()
		return nil, fmt.Errorf("registry: parse private key: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(pk, chainID)
	if err != nil {
		rpc.Close()
		return nil, fmt.Errorf("registry: build keyed transactor: %w", err)
	}

	contract, err := NewChainCIDRegistry(common.HexToAddress(contractAddr), rpc)
	if err != nil {
		rpc.Close()
		return nil, fmt.Errorf("registry: bind CIDRegistry at %s: %w", contractAddr, err)
	}

	session := &ChainCIDRegistrySession{
		Contract:     contract,
		CallOpts:     bind.CallOpts{Pending: false},
		TransactOpts: *auth,
	}

	return &chainClient{rpc: rpc, session: session}, nil
}

// Close releases the underlying RPC connection. Not on the Client
// interface — Mock has no resources to release — but exposed so call
// sites that hold a *chainClient directly can clean up. Callers obtain
// the concrete type via a type assertion when they need it.
func (c *chainClient) Close() {
	if c.rpc != nil {
		c.rpc.Close()
	}
}

// RegisterBundle posts registerBundle(...) and blocks on receipt. Same
// validation as Mock runs locally first so callers get the cheap
// pre-flight error path before paying for a revert.
func (c *chainClient) RegisterBundle(ctx context.Context, bundleID [32]byte, rec BundleRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if rec.NumChunks == 0 {
		return ErrNumChunksZero
	}
	if err := ValidateShards(rec.Shards); err != nil {
		return err
	}

	opts := c.session.TransactOpts
	opts.Context = ctx

	tx, err := c.session.Contract.RegisterBundle(
		&opts,
		bundleID,
		rec.MerkleRoot,
		rec.NumChunks,
		goShardsToContract(rec.Shards),
		rec.PolicyID,
	)
	if err != nil {
		return mapContractError(err)
	}
	if err := c.waitForTx(ctx, tx); err != nil {
		return fmt.Errorf("registry: registerBundle tx %s: %w", tx.Hash().Hex(), err)
	}
	return nil
}

// GetRecord reads the full BundleRecord via the contract's view function
// and converts it to the Go-side shape. Owner is lowercased to match
// what the Mock writes (msg.sender.Hex() returns mixed case).
func (c *chainClient) GetRecord(ctx context.Context, bundleID [32]byte) (*BundleRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	opts := c.session.CallOpts
	opts.Context = ctx

	raw, err := c.session.Contract.GetRecord(&opts, bundleID)
	if err != nil {
		return nil, mapContractError(err)
	}
	shards, err := contractShardsToGo(raw.Shards)
	if err != nil {
		return nil, fmt.Errorf("registry: convert shards for bundle %x: %w", bundleID, err)
	}
	return &BundleRecord{
		MerkleRoot:     raw.MerkleRoot,
		NumChunks:      raw.NumChunks,
		Shards:         shards,
		Owner:          strings.ToLower(raw.Owner.Hex()),
		PolicyID:       raw.PolicyId,
		RegisteredAt:   time.Unix(int64(raw.RegisteredAt), 0).UTC(),
		LastMigratedAt: time.Unix(int64(raw.LastMigratedAt), 0).UTC(),
	}, nil
}

// GetShardLayout is the cheap read used by the retrieval gateway —
// avoids the extra storage hit of pulling the full BundleRecord when
// only the shard array is needed.
func (c *chainClient) GetShardLayout(ctx context.Context, bundleID [32]byte) ([ShardCount]ShardPlacement, error) {
	if err := ctx.Err(); err != nil {
		return [ShardCount]ShardPlacement{}, err
	}
	opts := c.session.CallOpts
	opts.Context = ctx

	raw, err := c.session.Contract.GetShardLayout(&opts, bundleID)
	if err != nil {
		return [ShardCount]ShardPlacement{}, mapContractError(err)
	}
	return contractShardsToGo(raw)
}

// UpdateShardLayout posts updateShardLayout(...) and blocks on receipt.
// Caller must hold MIGRATOR_ROLE on the contract; an unauthorised caller
// surfaces as a generic revert (we don't fold it into ErrNotFound
// because the operator needs to see the AccessControlUnauthorizedAccount
// detail to know they're missing the role grant).
func (c *chainClient) UpdateShardLayout(ctx context.Context, bundleID [32]byte, shards [ShardCount]ShardPlacement) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ValidateShards(shards); err != nil {
		return err
	}
	opts := c.session.TransactOpts
	opts.Context = ctx

	tx, err := c.session.Contract.UpdateShardLayout(&opts, bundleID, goShardsToContract(shards))
	if err != nil {
		return mapContractError(err)
	}
	if err := c.waitForTx(ctx, tx); err != nil {
		return fmt.Errorf("registry: updateShardLayout tx %s: %w", tx.Hash().Hex(), err)
	}
	return nil
}

// waitForTx blocks on bind.WaitMined and surfaces a revert as a
// non-nil error so call sites don't have to inspect the receipt status
// themselves. txTimeout caps the wait so a stuck mempool can't hang
// the gateway indefinitely.
func (c *chainClient) waitForTx(ctx context.Context, tx *types.Transaction) error {
	waitCtx, cancel := context.WithTimeout(ctx, txTimeout)
	defer cancel()
	receipt, err := bind.WaitMined(waitCtx, c.rpc, tx)
	if err != nil {
		return err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("reverted at block %d", receipt.BlockNumber.Uint64())
	}
	return nil
}

// goShardsToContract converts the fixed-size Go shard array to the
// variable-length binding slice the abigen-generated call expects.
// CIDs are stored on-chain as `bytes` (raw multibase) per the rationale
// in CIDRegistry.sol; the string→[]byte conversion is the inverse of
// what contractShardsToGo does.
func goShardsToContract(shards [ShardCount]ShardPlacement) []CIDRegistryShardPlacement {
	out := make([]CIDRegistryShardPlacement, len(shards))
	for i, s := range shards {
		out[i] = CIDRegistryShardPlacement{
			Cid:  []byte(s.CID),
			Tier: s.Tier,
		}
	}
	return out
}

// contractShardsToGo enforces the RS(2,3) shard-count invariant on
// read so callers can rely on the fixed-size return type. A contract
// that ever returns a different count is a protocol-level bug; surface
// it loudly rather than silently truncating.
func contractShardsToGo(in []CIDRegistryShardPlacement) ([ShardCount]ShardPlacement, error) {
	if len(in) != ShardCount {
		return [ShardCount]ShardPlacement{}, fmt.Errorf("registry: expected %d shards from chain, got %d", ShardCount, len(in))
	}
	var out [ShardCount]ShardPlacement
	for i, s := range in {
		out[i] = ShardPlacement{
			CID:  string(s.Cid),
			Tier: s.Tier,
		}
	}
	return out, nil
}

// mapContractError matches revert reasons against the CIDRegistry
// custom errors (CIDRegistry.sol lines 72–76) and rewrites them to the
// package's sentinel errors so callers can branch via errors.Is. Falls
// through with the original error if no match — that's the right
// behaviour for AccessControl, SafeCast, and any unforeseen revert.
//
// String-matching is fragile but matches go-ethereum's idiomatic
// pattern for abigen v1 bindings (typed error decoding lands in
// abigen v2). The journal-extension VC-PoR work in Phase E is the
// right time to upgrade to ABI-driven error decoding.
func mapContractError(err error) error {
	if err == nil {
		return nil
	}
	s := err.Error()
	switch {
	case strings.Contains(s, "BundleAlreadyRegistered"):
		return ErrAlreadyRegistered
	case strings.Contains(s, "BundleNotFound"):
		return ErrNotFound
	case strings.Contains(s, "EmptyShardCID"):
		return ErrEmptyShardCID
	case strings.Contains(s, "NumChunksZero"):
		return ErrNumChunksZero
	case strings.Contains(s, "InvalidShardCount"):
		return ErrInvalidShardCount
	}
	return err
}
