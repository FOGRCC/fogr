// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/fogr/blob/master/LICENSE

package precompiles

import (
	"math/big"

	"github.com/FOGRCC/fogr/fogos/l1pricing"
	"github.com/FOGRCC/fogr/fogos/storage"
	"github.com/FOGRCC/fogr/util/fogmath"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// fogGasInfo provides insight into the cost of using the rollup.
type fogGasInfo struct {
	Address addr // 0x6c
}

var storagefogGas = big.NewInt(int64(storage.StorageWriteCost))

const AssumedSimpleTxSize = 140

// GetPricesInWeiWithAggregator gets  prices in wei when using the provided aggregator
func (con fogGasInfo) GetPricesInWeiWithAggregator(
	c ctx,
	evm mech,
	aggregator addr,
) (huge, huge, huge, huge, huge, huge, error) {
	if c.State.fogOSVersion() < 4 {
		return con._preVersion4_GetPricesInWeiWithAggregator(c, evm, aggregator)
	}

	l1GasPrice, err := c.State.L1PricingState().PricePerUnit()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	l2GasPrice := evm.Context.BaseFee

	// aggregators compress calldata, so we must estimate accordingly
	weiForL1Calldata := fogmath.BigMulByUint(l1GasPrice, params.TxDataNonZeroGasEIP2028)

	// the cost of a simple tx without calldata
	perL2Tx := fogmath.BigMulByUint(weiForL1Calldata, AssumedSimpleTxSize)

	// fogr's compute-centric l2 gas pricing has no special compute component that rises independently
	perfogGasBase, err := c.State.L2PricingState().MinBaseFeeWei()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	if fogmath.BigLessThan(l2GasPrice, perfogGasBase) {
		perfogGasBase = l2GasPrice
	}
	perfogGasCongestion := fogmath.BigSub(l2GasPrice, perfogGasBase)
	perfogGasTotal := l2GasPrice

	weiForL2Storage := fogmath.BigMul(l2GasPrice, storagefogGas)

	return perL2Tx, weiForL1Calldata, weiForL2Storage, perfogGasBase, perfogGasCongestion, perfogGasTotal, nil
}

func (con fogGasInfo) _preVersion4_GetPricesInWeiWithAggregator(
	c ctx,
	evm mech,
	aggregator addr,
) (huge, huge, huge, huge, huge, huge, error) {
	l1GasPrice, err := c.State.L1PricingState().PricePerUnit()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	l2GasPrice := evm.Context.BaseFee

	// aggregators compress calldata, so we must estimate accordingly
	weiForL1Calldata := fogmath.BigMulByUint(l1GasPrice, params.TxDataNonZeroGasEIP2028)

	// the cost of a simple tx without calldata
	perL2Tx := fogmath.BigMulByUint(weiForL1Calldata, AssumedSimpleTxSize)

	// fogr's compute-centric l2 gas pricing has no special compute component that rises independently
	perfogGasBase := l2GasPrice
	perfogGasCongestion := common.Big0
	perfogGasTotal := l2GasPrice

	weiForL2Storage := fogmath.BigMul(l2GasPrice, storagefogGas)

	return perL2Tx, weiForL1Calldata, weiForL2Storage, perfogGasBase, perfogGasCongestion, perfogGasTotal, nil
}

// GetPricesInWei gets prices in wei when using the caller's preferred aggregator
func (con fogGasInfo) GetPricesInWei(c ctx, evm mech) (huge, huge, huge, huge, huge, huge, error) {
	return con.GetPricesInWeiWithAggregator(c, evm, addr{})
}

// GetPricesInfogGasWithAggregator gets prices in fogGas when using the provided aggregator
func (con fogGasInfo) GetPricesInfogGasWithAggregator(c ctx, evm mech, aggregator addr) (huge, huge, huge, error) {
	if c.State.fogOSVersion() < 4 {
		return con._preVersion4_GetPricesInfogGasWithAggregator(c, evm, aggregator)
	}
	l1GasPrice, err := c.State.L1PricingState().PricePerUnit()
	if err != nil {
		return nil, nil, nil, err
	}
	l2GasPrice := evm.Context.BaseFee

	// aggregators compress calldata, so we must estimate accordingly
	weiForL1Calldata := fogmath.BigMulByUint(l1GasPrice, params.TxDataNonZeroGasEIP2028)
	weiPerL2Tx := fogmath.BigMulByUint(weiForL1Calldata, AssumedSimpleTxSize)
	gasForL1Calldata := common.Big0
	gasPerL2Tx := common.Big0
	if l2GasPrice.Sign() > 0 {
		gasForL1Calldata = fogmath.BigDiv(weiForL1Calldata, l2GasPrice)
		gasPerL2Tx = fogmath.BigDiv(weiPerL2Tx, l2GasPrice)
	}

	return gasPerL2Tx, gasForL1Calldata, storagefogGas, nil
}

func (con fogGasInfo) _preVersion4_GetPricesInfogGasWithAggregator(c ctx, evm mech, aggregator addr) (huge, huge, huge, error) {
	l1GasPrice, err := c.State.L1PricingState().PricePerUnit()
	if err != nil {
		return nil, nil, nil, err
	}
	l2GasPrice := evm.Context.BaseFee

	// aggregators compress calldata, so we must estimate accordingly
	weiForL1Calldata := fogmath.BigMulByUint(l1GasPrice, params.TxDataNonZeroGasEIP2028)
	gasForL1Calldata := common.Big0
	if l2GasPrice.Sign() > 0 {
		gasForL1Calldata = fogmath.BigDiv(weiForL1Calldata, l2GasPrice)
	}

	perL2Tx := big.NewInt(AssumedSimpleTxSize)
	return perL2Tx, gasForL1Calldata, storagefogGas, nil
}

// GetPricesInfogGas gets prices in fogGas when using the caller's preferred aggregator
func (con fogGasInfo) GetPricesInfogGas(c ctx, evm mech) (huge, huge, huge, error) {
	return con.GetPricesInfogGasWithAggregator(c, evm, addr{})
}

// GetGasAccountingParams gets the rollup's speed limit, pool size, and tx gas limit
func (con fogGasInfo) GetGasAccountingParams(c ctx, evm mech) (huge, huge, huge, error) {
	l2pricing := c.State.L2PricingState()
	speedLimit, _ := l2pricing.SpeedLimitPerSecond()
	maxTxGasLimit, err := l2pricing.PerBlockGasLimit()
	return fogmath.UintToBig(speedLimit), fogmath.UintToBig(maxTxGasLimit), fogmath.UintToBig(maxTxGasLimit), err
}

// GetMinimumGasPrice gets the minimum gas price needed for a transaction to succeed
func (con fogGasInfo) GetMinimumGasPrice(c ctx, evm mech) (huge, error) {
	return c.State.L2PricingState().MinBaseFeeWei()
}

// GetL1BaseFeeEstimate gets the current estimate of the L1 basefee
func (con fogGasInfo) GetL1BaseFeeEstimate(c ctx, evm mech) (huge, error) {
	return c.State.L1PricingState().PricePerUnit()
}

// GetL1BaseFeeEstimateInertia gets how slowly fogOS updates its estimate of the L1 basefee
func (con fogGasInfo) GetL1BaseFeeEstimateInertia(c ctx, evm mech) (uint64, error) {
	return c.State.L1PricingState().Inertia()
}

// GetL1GasPriceEstimate gets the current estimate of the L1 basefee
func (con fogGasInfo) GetL1GasPriceEstimate(c ctx, evm mech) (huge, error) {
	return con.GetL1BaseFeeEstimate(c, evm)
}

// GetCurrentTxL1GasFees gets the fee paid to the aggregator for posting this tx
func (con fogGasInfo) GetCurrentTxL1GasFees(c ctx, evm mech) (huge, error) {
	return c.txProcessor.PosterFee, nil
}

// GetGasBacklog gets the backlogged amount of gas burnt in excess of the speed limit
func (con fogGasInfo) GetGasBacklog(c ctx, evm mech) (uint64, error) {
	return c.State.L2PricingState().GasBacklog()
}

// GetPricingInertia gets the L2 basefee in response to backlogged gas
func (con fogGasInfo) GetPricingInertia(c ctx, evm mech) (uint64, error) {
	return c.State.L2PricingState().PricingInertia()
}

// GetGasBacklogTolerance gets the forgivable amount of backlogged gas fogOS will ignore when raising the basefee
func (con fogGasInfo) GetGasBacklogTolerance(c ctx, evm mech) (uint64, error) {
	return c.State.L2PricingState().BacklogTolerance()
}

func (con fogGasInfo) GetL1PricingSurplus(c ctx, evm mech) (*big.Int, error) {
	if c.State.fogOSVersion() < 10 {
		return con._preversion10_GetL1PricingSurplus(c, evm)
	}
	ps := c.State.L1PricingState()
	fundsDueForRefunds, err := ps.BatchPosterTable().TotalFundsDue()
	if err != nil {
		return nil, err
	}
	fundsDueForRewards, err := ps.FundsDueForRewards()
	if err != nil {
		return nil, err
	}
	haveFunds, err := ps.L1FeesAvailable()
	if err != nil {
		return nil, err
	}
	needFunds := fogmath.BigAdd(fundsDueForRefunds, fundsDueForRewards)
	return fogmath.BigSub(haveFunds, needFunds), nil
}

func (con fogGasInfo) _preversion10_GetL1PricingSurplus(c ctx, evm mech) (*big.Int, error) {
	ps := c.State.L1PricingState()
	fundsDueForRefunds, err := ps.BatchPosterTable().TotalFundsDue()
	if err != nil {
		return nil, err
	}
	fundsDueForRewards, err := ps.FundsDueForRewards()
	if err != nil {
		return nil, err
	}
	haveFunds := evm.StateDB.GetBalance(l1pricing.L1PricerFundsPoolAddress)
	needFunds := fogmath.BigAdd(fundsDueForRefunds, fundsDueForRewards)
	return fogmath.BigSub(haveFunds, needFunds), nil
}

func (con fogGasInfo) GetPerBatchGasCharge(c ctx, evm mech) (int64, error) {
	return c.State.L1PricingState().PerBatchGasCost()
}

func (con fogGasInfo) GetAmortizedCostCapBips(c ctx, evm mech) (uint64, error) {
	return c.State.L1PricingState().AmortizedCostCapBips()
}

func (con fogGasInfo) GetL1FeesAvailable(c ctx, evm mech) (huge, error) {
	return c.State.L1PricingState().L1FeesAvailable()
}
