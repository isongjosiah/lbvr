package main

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// GasResult is the parsed output of one forge invocation: a flat
// (fn,depth) → gas mapping. forge emits each measurement once per test
// run — we don't average, since EVM gas at fixed inputs is deterministic.
type GasResult struct {
	// Keyed first by function name ("post"/"respond"/"verdict"), then
	// by depth (3, 5, 7, 9, 11, 13).
	ByFn map[string]map[int]uint64
}

// gasLineRegex matches the format emitted by PoRVerifierGasTest:
//
//	E5_GAS,<fn>, <depth> , <gas>
//
// console2.log padding is irregular — there's whitespace around the
// numeric arguments because the cheatcode formats `(string, uint, string,
// uint)` with single-space separators. We tolerate any whitespace
// between fields.
var gasLineRegex = regexp.MustCompile(`E5_GAS,\s*(\w+)\s*,\s*(\d+)\s*,\s*(\d+)`)

// runForgeGas shells out to `forge test --match-contract PoRVerifierGas`,
// parses E5_GAS lines from stdout, and returns one row per (fn, depth).
//
// matchTest is an optional --match-test regex (e.g., "depth_(3|5|7)") for
// running a subset of the sweep — used by the smoke test to cap wall time.
// Empty matchTest runs the full sweep.
func runForgeGas(forgeBin, contractsRoot, matchTest string) (*GasResult, error) {
	args := []string{
		"test",
		"--root", contractsRoot,
		"--match-contract", "PoRVerifierGas",
		"-vv",
	}
	if matchTest != "" {
		args = append(args, "--match-test", matchTest)
	}

	cmd := exec.Command(forgeBin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("forge test failed: %w\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}

	res := &GasResult{ByFn: map[string]map[int]uint64{}}
	matched := 0
	for _, line := range strings.Split(stdout.String(), "\n") {
		m := gasLineRegex.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		fn := m[1]
		depth, err := strconv.Atoi(m[2])
		if err != nil {
			return nil, fmt.Errorf("parse depth in %q: %w", line, err)
		}
		gas, err := strconv.ParseUint(m[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse gas in %q: %w", line, err)
		}
		if _, ok := res.ByFn[fn]; !ok {
			res.ByFn[fn] = map[int]uint64{}
		}
		// Last write wins. forge's test isolation ensures clean state
		// between functions, but a depth could in theory be measured
		// multiple times if tests are duplicated; the deterministic
		// output makes "last wins" stable.
		res.ByFn[fn][depth] = gas
		matched++
	}

	if matched == 0 {
		return nil, errors.New("no E5_GAS lines found in forge output — gas test may have failed silently or output format drifted")
	}

	return res, nil
}

// Lookup returns the gas value for (fn, depth), with a clear error if
// missing. The bench is responsible for handling missing values; we
// don't default to 0 (which would silently corrupt the figure).
func (g *GasResult) Lookup(fn string, depth int) (uint64, error) {
	byDepth, ok := g.ByFn[fn]
	if !ok {
		return 0, fmt.Errorf("gas: function %q not measured", fn)
	}
	gas, ok := byDepth[depth]
	if !ok {
		return 0, fmt.Errorf("gas: function %q has no measurement at depth=%d", fn, depth)
	}
	return gas, nil
}
