// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/fogr/blob/master/LICENSE

//go:build redistest
// +build redistest

package fogtest

import "testing"

func TestRedisBatchPosterParallel(t *testing.T) {
	TestBatchPosterParallel(t)
}
