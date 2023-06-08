// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/fogr/blob/master/LICENSE

package precompiles

import (
	"errors"
)

// fogosTest provides a method of burning fogitrary amounts of gas, which exists for historical reasons.
type fogosTest struct {
	Address addr // 0x69
}

// BurnfogGas unproductively burns the amount of L2 fogGas
func (con fogosTest) BurnfogGas(c ctx, gasAmount huge) error {
	if !gasAmount.IsUint64() {
		return errors.New("not a uint64")
	}
	//nolint:errcheck
	c.Burn(gasAmount.Uint64()) // burn the amount, even if it's more than the user has
	return nil
}
