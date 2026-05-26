// Anchor client for on-chain provenance hashes (CLAUDE.md §4.6;
// docs/provenance-spec.md §6). The gateway anchors the JCS-canonical
// SHA-256 of every retrieval's signed PROV-JSON doc so a verifier can
// later cross-check the document's hash against on-chain bytes.
//
// Two implementations:
//   - mockAnchor: in-memory map. Used by tests and by `lbvr-gateway`
//     when the operator hasn't populated CHAIN_RPC_URL +
//     CHAIN_PRIVATE_KEY + AUDITOR_LOG_ADDRESS.
//   - chainAnchor: live AuditorLog binding. Construction dials the RPC,
//     fetches the chain ID, parses the private key, and binds the
//     contract; Anchor posts a tx + waits for the receipt; Get reads
//     the on-chain struct via the view function.

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/isongjosiah/lbvr-med/internal/registry"
)

// anchorTxTimeout bounds bind.WaitMined for AnchorProvenance calls.
// Matches the registry.txTimeout convention; mismatch here would let a
// stuck tx hang gateway shutdown indefinitely.
const anchorTxTimeout = 60 * time.Second

// Sentinel errors returned by both implementations. ErrAnchorAlreadyExists
// matches the AuditorLog AlreadyAnchored custom revert; ErrAnchorNotFound
// matches AnchorNotFound. Callers branch on these via errors.Is.
var (
	ErrAnchorAlreadyExists = errors.New("anchor: already anchored")
	ErrAnchorNotFound      = errors.New("anchor: not found")
	ErrAnchorEmptyProvHash = errors.New("anchor: provHash is zero")
)

// AnchorRecord mirrors AuditorLog.ProvenanceAnchor on-chain.
type AnchorRecord struct {
	ProvHash    [32]byte
	BlockNumber uint64
	TxHash      string
	AnchoredBy  string // 0x-hex address, lowercased to match mockAnchor.
}

// AnchorClient is the surface the gateway depends on. Both impls satisfy
// it without sharing internals; chainAnchor wraps go-ethereum and the
// AuditorLog binding, mockAnchor wraps a mutex+map.
type AnchorClient interface {
	Anchor(ctx context.Context, bundleID, retrievalID, provHash [32]byte) (AnchorRecord, error)
	Get(ctx context.Context, bundleID, retrievalID [32]byte) (AnchorRecord, error)
}

// mockAnchor is the in-memory implementation. block numbers are a
// monotonic counter starting at 1 so the doc's blockNumber field is
// non-zero (otherwise verifiers might confuse "anchored at block 0"
// with "no anchor"). Concurrent-safe.
type mockAnchor struct {
	mu        sync.RWMutex
	anchors   map[[64]byte]AnchorRecord // key = bundleID || retrievalID
	nextBlock uint64
}

func newMockAnchor() *mockAnchor {
	return &mockAnchor{
		anchors:   make(map[[64]byte]AnchorRecord),
		nextBlock: 1,
	}
}

func (m *mockAnchor) Anchor(_ context.Context, bundleID, retrievalID, provHash [32]byte) (AnchorRecord, error) {
	if provHash == ([32]byte{}) {
		return AnchorRecord{}, ErrAnchorEmptyProvHash
	}
	key := concatID(bundleID, retrievalID)

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.anchors[key]; ok {
		return AnchorRecord{}, ErrAnchorAlreadyExists
	}

	rec := AnchorRecord{
		ProvHash:    provHash,
		BlockNumber: m.nextBlock,
		TxHash:      "0xmock" + hex.EncodeToString(randBytes(28)),
		AnchoredBy:  "0xmockGateway0000000000000000000000000000",
	}
	m.anchors[key] = rec
	m.nextBlock++
	return rec, nil
}

func (m *mockAnchor) Get(_ context.Context, bundleID, retrievalID [32]byte) (AnchorRecord, error) {
	key := concatID(bundleID, retrievalID)
	m.mu.RLock()
	defer m.mu.RUnlock()
	rec, ok := m.anchors[key]
	if !ok {
		return AnchorRecord{}, ErrAnchorNotFound
	}
	return rec, nil
}

func concatID(bundleID, retrievalID [32]byte) (out [64]byte) {
	copy(out[:32], bundleID[:])
	copy(out[32:], retrievalID[:])
	return
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}

// chainAnchor is the eth-client-backed AnchorClient. Construction dials
// the RPC, fetches the chain ID, parses privKey into a keyed transactor,
// and binds the AuditorLog contract via the abigen-generated session.
type chainAnchor struct {
	rpc     *ethclient.Client
	session *registry.ChainAuditorLogSession
	from    common.Address // cached signer address for AnchoredBy on writes
}

// newChainAnchor builds a chainAnchor against rpcURL with privKey
// signing AnchorProvenance writes to contractAddress. privKey is parsed
// with or without the "0x" prefix. Chain ID is fetched via eth_chainId
// so callers don't need to thread it through.
func newChainAnchor(ctx context.Context, rpcURL, contractAddress, privKey string) (*chainAnchor, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if rpcURL == "" {
		return nil, errors.New("chainAnchor: rpcURL is empty")
	}
	if contractAddress == "" {
		return nil, errors.New("chainAnchor: contractAddress is empty")
	}
	if privKey == "" {
		return nil, errors.New("chainAnchor: privKey is empty")
	}
	if !common.IsHexAddress(contractAddress) {
		return nil, fmt.Errorf("chainAnchor: contractAddress %q is not a valid hex address", contractAddress)
	}

	rpc, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("chainAnchor: dial RPC %q: %w", rpcURL, err)
	}

	chainID, err := rpc.ChainID(ctx)
	if err != nil {
		rpc.Close()
		return nil, fmt.Errorf("chainAnchor: fetch chain ID: %w", err)
	}

	pk, err := crypto.HexToECDSA(strings.TrimPrefix(privKey, "0x"))
	if err != nil {
		rpc.Close()
		return nil, fmt.Errorf("chainAnchor: parse private key: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(pk, chainID)
	if err != nil {
		rpc.Close()
		return nil, fmt.Errorf("chainAnchor: build keyed transactor: %w", err)
	}

	contract, err := registry.NewChainAuditorLog(common.HexToAddress(contractAddress), rpc)
	if err != nil {
		rpc.Close()
		return nil, fmt.Errorf("chainAnchor: bind AuditorLog at %s: %w", contractAddress, err)
	}

	return &chainAnchor{
		rpc: rpc,
		session: &registry.ChainAuditorLogSession{
			Contract:     contract,
			CallOpts:     bind.CallOpts{Pending: false},
			TransactOpts: *auth,
		},
		from: auth.From,
	}, nil
}

// Close releases the underlying RPC connection. The gateway should
// defer this on shutdown so the dial doesn't leak.
func (c *chainAnchor) Close() {
	if c.rpc != nil {
		c.rpc.Close()
	}
}

// Anchor posts anchorProvenance(bundleID, retrievalID, provHash) and
// blocks on the receipt. Returns the resulting AnchorRecord with
// BlockNumber + TxHash filled from the mined receipt; AnchoredBy is
// the cached signer address.
//
// Maps AlreadyAnchored / EmptyProvHash custom errors to local
// sentinels; other reverts (AccessControl, SafeCast) surface unchanged
// so the operator can see the underlying detail.
func (c *chainAnchor) Anchor(ctx context.Context, bundleID, retrievalID, provHash [32]byte) (AnchorRecord, error) {
	if err := ctx.Err(); err != nil {
		return AnchorRecord{}, err
	}
	if provHash == ([32]byte{}) {
		return AnchorRecord{}, ErrAnchorEmptyProvHash
	}

	opts := c.session.TransactOpts
	opts.Context = ctx

	tx, err := c.session.Contract.AnchorProvenance(&opts, bundleID, retrievalID, provHash)
	if err != nil {
		return AnchorRecord{}, mapAuditorError(err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, anchorTxTimeout)
	defer cancel()
	receipt, err := bind.WaitMined(waitCtx, c.rpc, tx)
	if err != nil {
		return AnchorRecord{}, fmt.Errorf("chainAnchor: wait for tx %s: %w", tx.Hash().Hex(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return AnchorRecord{}, fmt.Errorf("chainAnchor: tx %s reverted at block %d", tx.Hash().Hex(), receipt.BlockNumber.Uint64())
	}

	return AnchorRecord{
		ProvHash:    provHash,
		BlockNumber: receipt.BlockNumber.Uint64(),
		TxHash:      tx.Hash().Hex(),
		AnchoredBy:  strings.ToLower(c.from.Hex()),
	}, nil
}

// Get reads ProvenanceAnchor via the view function. The on-chain struct
// does not carry the originating TxHash (only block/timestamp/signer),
// so the returned AnchorRecord.TxHash is empty for chain-fetched
// records. Callers needing the tx hash can scan the ProvenanceAnchored
// event log filtered by (bundleID, retrievalID).
func (c *chainAnchor) Get(ctx context.Context, bundleID, retrievalID [32]byte) (AnchorRecord, error) {
	if err := ctx.Err(); err != nil {
		return AnchorRecord{}, err
	}
	opts := c.session.CallOpts
	opts.Context = ctx

	raw, err := c.session.Contract.GetProvenanceAnchor(&opts, bundleID, retrievalID)
	if err != nil {
		return AnchorRecord{}, mapAuditorError(err)
	}
	return AnchorRecord{
		ProvHash:    raw.ProvHash,
		BlockNumber: raw.BlockNumber.Uint64(),
		TxHash:      "", // unavailable from the view; see doc above.
		AnchoredBy:  strings.ToLower(raw.AnchoredBy.Hex()),
	}, nil
}

// mapAuditorError matches revert reasons against the AuditorLog custom
// errors (AuditorLog.sol lines 75–79) and rewrites them to the package
// sentinels. Same string-matching limitation as registry.mapContractError;
// the abigen v2 upgrade in Phase E is the right place to swap to
// typed-error decoding.
func mapAuditorError(err error) error {
	if err == nil {
		return nil
	}
	s := err.Error()
	switch {
	case strings.Contains(s, "AlreadyAnchored"):
		return ErrAnchorAlreadyExists
	case strings.Contains(s, "AnchorNotFound"):
		return ErrAnchorNotFound
	case strings.Contains(s, "EmptyProvHash"):
		return ErrAnchorEmptyProvHash
	}
	return err
}
