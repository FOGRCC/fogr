// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/fogr/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity >=0.4.21 <0.9.0;

/// @title Deprecated - Info about the rollup just prior to the FOGR upgrade
/// @notice Precompiled contract in every FOGR chain for retryable transaction related data retrieval and interactions. Exists at 0x000000000000000000000000000000000000006f
interface fogStatistics {
    /// @return (
    ///      Number of accounts,
    ///      Total storage allocated (includes storage that was later deallocated),
    ///      Total fogGas used,
    ///      Number of transaction receipt issued,
    ///      Number of contracts created,
    ///    )
    function getStats()
        external
        view
        returns (
            uint256,
            uint256,
            uint256,
            uint256,
            uint256,
            uint256
        );
}
