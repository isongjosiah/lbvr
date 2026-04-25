package registry

import (
	"context"
	"errors"
)

// ErrChainNotImplemented is returned by every method on the chain client
// stub until D12 wires up abigen + ethclient. Kept as a sentinel so the
// retrieval gateway and orchestrator can branch on it cleanly.
var ErrChainNotImplemented = errors.New(
	"registry: chain client not implemented; run forge build + abigen first (D12)",
)

// chainClient is the placeholder type for the eth-client-backed Client
// implementation. The fields are spelled out so the D12 wiring is a strict
// fill-in: rpcURL → ethclient.Dial, contractAddr → abigen-bound instance,
// privKey → bind.NewKeyedTransactorWithChainID.
type chainClient struct {
	rpcURL       string
	contractAddr string
	privKey      string
}

// NewChain returns a Client whose methods all return ErrChainNotImplemented.
//
// TODO(D12): replace this stub with the abigen-generated CIDRegistry
// binding once `forge build` has emitted the ABI and the deployment has
// landed on Polygon zkEVM Cardona. The construction sequence is:
//
//  1. cd contracts && forge build
//  2. abigen --abi out/CIDRegistry.sol/CIDRegistry.json \
//     --pkg registry --type ChainCIDRegistry \
//     --out internal/registry/contract_binding.go
//  3. wire ethclient.Dial(rpcURL) + bind.NewKeyedTransactorWithChainID
//     into the four methods below.
//
// Returning the stub instead of a no-op nil keeps the Client interface
// satisfied so consumers can dependency-inject it now.
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
	return &chainClient{
		rpcURL:       rpcURL,
		contractAddr: contractAddr,
		privKey:      privKey,
	}, nil
}

func (c *chainClient) RegisterBundle(_ context.Context, _ [32]byte, _ BundleRecord) error {
	return ErrChainNotImplemented
}

func (c *chainClient) GetRecord(_ context.Context, _ [32]byte) (*BundleRecord, error) {
	return nil, ErrChainNotImplemented
}

func (c *chainClient) GetShardLayout(_ context.Context, _ [32]byte) ([ShardCount]ShardPlacement, error) {
	return [ShardCount]ShardPlacement{}, ErrChainNotImplemented
}

func (c *chainClient) UpdateShardLayout(_ context.Context, _ [32]byte, _ [ShardCount]ShardPlacement) error {
	return ErrChainNotImplemented
}
