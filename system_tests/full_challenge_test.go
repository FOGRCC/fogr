// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/fogr/blob/master/LICENSE

//go:build challengetest
// +build challengetest

//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package fogtest

import (
	"testing"
)

func TestChallengeManagerFullAsserterIncorrect(t *testing.T) {
	RunChallengeTest(t, false)
}

func TestChallengeManagerFullAsserterCorrect(t *testing.T) {
	RunChallengeTest(t, true)
}
