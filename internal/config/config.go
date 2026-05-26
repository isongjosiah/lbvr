// Package config loads LBVR-Med runtime secrets from a .env-style file with
// a fallback to process environment variables. The fallback order matches
// the deployment matrix in CLAUDE.md §13: local dev reads ./.env; CI and
// production inject the same variables directly into the process environment
// (no .env on disk). Callers never panic on a missing value — each tier
// client validates only the fields it actually needs in its constructor.
package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config mirrors the variable names in .env.example. Empty string means
// "not set" at this layer; downstream constructors decide whether a
// missing value is fatal for them.
type Config struct {
	// Pinata (hot tier)
	PinataJWT     string
	PinataGateway string

	// Filebase (warm tier)
	FilebaseAccessKey string
	FilebaseSecretKey string
	FilebaseBucket    string

	// Irys / Arweave (cold tier)
	IrysNodeURL    string
	IrysPrivateKey string

	// L4 chain (generic; per .env.example, deploy scripts pick a named
	// network via foundry aliases while Go reads the CHAIN_* variables).
	// Originally CardonaRPCURL/CardonaPrivateKey before the D23
	// PureChain pivot — kept generic so future chain swaps don't need
	// another rename.
	ChainRPCURL       string
	ChainPrivateKey   string
	PolygonscanAPIKey string

	// Deployed contract addresses (populated after D5 deployment)
	CIDRegistryAddress string
	PoRVerifierAddress string
	AuditorLogAddress  string

	// Gateway BLS12-381 quorum (populated after provenance keygen)
	GatewayBLSSK1 string
	GatewayBLSSK2 string
	GatewayBLSPK1 string
	GatewayBLSPK2 string
}

// Load reads path (defaulting to ".env" in the current working directory
// when path is empty) and returns the parsed Config. A missing .env file
// is NOT an error — in CI/prod the process env already holds the values.
// Any other filesystem error is returned unchanged.
func Load(path string) (*Config, error) {
	if path == "" {
		path = ".env"
	}
	if _, err := os.Stat(path); err == nil {
		if err := godotenv.Overload(path); err != nil {
			return nil, fmt.Errorf("config: load %s: %w", path, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("config: stat %s: %w", path, err)
	}

	return &Config{
		PinataJWT:     os.Getenv("PINATA_JWT"),
		PinataGateway: os.Getenv("PINATA_GATEWAY"),

		FilebaseAccessKey: os.Getenv("FILEBASE_ACCESS_KEY"),
		FilebaseSecretKey: os.Getenv("FILEBASE_SECRET_KEY"),
		FilebaseBucket:    os.Getenv("FILEBASE_BUCKET"),

		IrysNodeURL:    os.Getenv("IRYS_NODE_URL"),
		IrysPrivateKey: os.Getenv("IRYS_PRIVATE_KEY"),

		ChainRPCURL:       os.Getenv("CHAIN_RPC_URL"),
		ChainPrivateKey:   os.Getenv("CHAIN_PRIVATE_KEY"),
		PolygonscanAPIKey: os.Getenv("POLYGONSCAN_API_KEY"),

		CIDRegistryAddress: os.Getenv("CID_REGISTRY_ADDRESS"),
		PoRVerifierAddress: os.Getenv("POR_VERIFIER_ADDRESS"),
		AuditorLogAddress:  os.Getenv("AUDITOR_LOG_ADDRESS"),

		GatewayBLSSK1: os.Getenv("GATEWAY_BLS_SK_1"),
		GatewayBLSSK2: os.Getenv("GATEWAY_BLS_SK_2"),
		GatewayBLSPK1: os.Getenv("GATEWAY_BLS_PK_1"),
		GatewayBLSPK2: os.Getenv("GATEWAY_BLS_PK_2"),
	}, nil
}
