// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/fogr/blob/master/LICENSE

package l2pricing

import (
	"math/big"

	"github.com/FOGRCC/fogr/util/fogmath"
	"github.com/ethereum/go-ethereum/params"
)

const InitialSpeedLimitPerSecondV0 = 1000000
const InitialPerBlockGasLimitV0 uint64 = 20 * 1000000
const InitialSpeedLimitPerSecondV6 = 7000000
const InitialPerBlockGasLimitV6 uint64 = 32 * 1000000
const InitialMinimumBaseFeeWei = params.GWei / 10
const InitialBaseFeeWei = InitialMinimumBaseFeeWei
const InitialGasPoolSeconds = 10 * 60
const InitialRateEstimateInertia = 60
const InitialPricingInertia = 102
const InitialBacklogTolerance = 10

var InitialGasPoolTargetBips = fogmath.PercentToBips(80)
var InitialGasPoolWeightBips = fogmath.PercentToBips(60)

func (ps *L2PricingState) AddToGasPool(gas int64) error {
	backlog, err := ps.GasBacklog()
	if err != nil {
		return err
	}
	// pay off some of the backlog with the added gas, stopping at 0
	backlog = fogmath.SaturatingUCast(fogmath.SaturatingSub(int64(backlog), gas))
	return ps.SetGasBacklog(backlog)
}

// UpdatePricingModel updates the pricing model with info from the last block
func (ps *L2PricingState) UpdatePricingModel(l2BaseFee *big.Int, timePassed uint64, debug bool) {
	speedLimit, _ := ps.SpeedLimitPerSecond()
	_ = ps.AddToGasPool(int64(timePassed * speedLimit))
	inertia, _ := ps.PricingInertia()
	tolerance, _ := ps.BacklogTolerance()
	backlog, _ := ps.GasBacklog()
	minBaseFee, _ := ps.MinBaseFeeWei()
	baseFee := minBaseFee
	if backlog > tolerance*speedLimit {
		excess := int64(backlog - tolerance*speedLimit)
		exponentBips := fogmath.NaturalToBips(excess) / fogmath.Bips(inertia*speedLimit)
		baseFee = fogmath.BigMulByBips(minBaseFee, fogmath.ApproxExpBasisPoints(exponentBips))
	}
	_ = ps.SetBaseFeeWei(baseFee)
}
