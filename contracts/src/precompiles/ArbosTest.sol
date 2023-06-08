// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/fogr/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity >=0.4.21 <0.9.0;

/// @title Deprecated - Provides a method of burning fogitrary amounts of gas,
/// @notice This exists for historical reasons. Pre-FOGR, `fogosTest` had additional methods only the zero address could call.
/// These have been removed since users don't use them and calls to missing methods revert.
/// Precompiled contract that exists in every FOGR chain at 0x0000000000000000000000000000000000000069.
interface fogosTest {
    /// @notice Unproductively burns the amount of L2 fogGas
    function burnfogGas(uint256 gasAmount) external pure;
}
