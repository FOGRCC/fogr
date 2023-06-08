//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package precompiles

// fogosActs precompile represents fogOS's internal actions as calls it makes to itself
type fogosActs struct {
	Address addr // 0xa4b05

	CallerNotfogOSError func() error
}

func (con fogosActs) StartBlock(c ctx, evm mech, l1BaseFee huge, l1BlockNumber, l2BlockNumber, timeLastBlock uint64) error {
	return con.CallerNotfogOSError()
}

func (con fogosActs) BatchPostingReport(c ctx, evm mech, batchTimestamp huge, batchPosterAddress addr, batchNumber uint64, batchDataGas uint64, l1BaseFeeWei huge) error {
	return con.CallerNotfogOSError()
}
