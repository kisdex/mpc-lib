// Copyright (c) 2023, Circle Internet Financial, LTD. All rights reserved.
//
//  SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cggplus

import (
	//	"github.com/kisdex/mpc-lib/tss"
	//	"sync"
	"testing"
)

/*
*  Uncomment test to check individual round
*

	func TestRound1(t *testing.T) {
		params, parties, outCh, _, _, _ := SetupParties(t)
		rounds := RunRound1(t, params, parties, outCh)

		wg := sync.WaitGroup{}
		partyCount := len(parties)
		errChs := make(chan *tss.Error, partyCount*partyCount*3)
		for _, round := range rounds {
			wg.Add(1)
			go func(round *round1) {
				defer wg.Done()
				nextRound := &round2{round}
				nextRound.VerifyRound1Messages(errChs)
			}(round)
		}
		wg.Wait()
		close(errChs)
		AssertNoErrors(t, errChs)
	}

	func TestRound2(t *testing.T) {
		params, parties, outCh, _, _, _ := SetupParties(t)
		t.Logf("round 1")
		round1s := RunRound1(t, params, parties, outCh)
		t.Logf("round 2")
		totalMessages := len(parties) * len(parties)
		round2s := RunRound[*round1, *round2](t, params, parties, round1s, totalMessages, outCh)

		wg := sync.WaitGroup{}
		partyCount := len(parties)
		errChs := make(chan *tss.Error, partyCount*partyCount*partyCount)
		for _, round := range round2s {
			wg.Add(1)
			go func(round *round2) {
				defer wg.Done()
				nextRound := &round3{round}
				nextRound.VerifyRound2Messages(errChs)
			}(round)
		}
		wg.Wait()
		close(errChs)
		AssertNoErrors(t, errChs)
	}

	func TestRound3(t *testing.T) {
		params, parties, outCh, _, _, _ := SetupParties(t)
		t.Logf("round 1")
		round1s := RunRound1(t, params, parties, outCh)
		t.Logf("round 2")
		totalMessages := len(parties) * len(parties)
		round2s := RunRound[*round1, *round2](t, params, parties, round1s, totalMessages, outCh)
		t.Logf("round 3")
		round3s := RunRound[*round2, *round3](t, params, parties, round2s, len(parties), outCh)

		wg := sync.WaitGroup{}
		partyCount := len(parties)
		errChs := make(chan *tss.Error, partyCount*partyCount*partyCount)
		for _, round := range round3s {
			wg.Add(1)
			go func(round *round3) {
				defer wg.Done()
				nextRound := &round4{round}
				nextRound.VerifyRound3Messages(errChs)
			}(round)
		}
		wg.Wait()
		close(errChs)
		AssertNoErrors(t, errChs)
	}

	func TestRound4(t *testing.T) {
		params, parties, outCh, _, _, _ := SetupParties(t)
		t.Logf("round 1")
		round1s := RunRound1(t, params, parties, outCh)
		t.Logf("round 2")
		totalMessages := len(parties) * len(parties)
		round2s := RunRound[*round1, *round2](t, params, parties, round1s, totalMessages, outCh)
		t.Logf("round 3")
		round3s := RunRound[*round2, *round3](t, params, parties, round2s, len(parties), outCh)
		t.Logf("round 4")
		_ = RunRound[*round3, *round4](t, params, parties, round3s, len(parties), outCh)

		// skip verification; round 4 does not output messages
	}

	func TestRound5(t *testing.T) {
		params, parties, outCh, _, _, _ := SetupParties(t)

		t.Logf("round 1")
		round1s := RunRound1(t, params, parties, outCh)
		t.Logf("round 2")
		totalMessages := len(parties) * len(parties)
		round2s := RunRound[*round1, *round2](t, params, parties, round1s, totalMessages, outCh)
		t.Logf("round 3")
		round3s := RunRound[*round2, *round3](t, params, parties, round2s, len(parties), outCh)
		t.Logf("round 4")
		round4s := RunRound[*round3, *round4](t, params, parties, round3s, len(parties), outCh)
		t.Logf("round 5")
		round5s := RunRound[*round4, *round5](t, params, parties, round4s, len(parties), outCh)

		wg := sync.WaitGroup{}
		partyCount := len(parties)
		errChs := make(chan *tss.Error, partyCount*partyCount*partyCount)
		for _, round := range round5s {
			wg.Add(1)
			go func(round *round5) {
				defer wg.Done()
				nextRound := &finalization{round}
				nextRound.VerifyRound5Messages(errChs)
			}(round)
		}
		wg.Wait()
		close(errChs)
		AssertNoErrors(t, errChs)
	}
*/
func TestRoundFinalization(t *testing.T) {
	params, parties, outCh, _, _, _ := SetupParties(t)

	t.Logf("round 1")
	round1s := RunRound1(t, params, parties, outCh)
	t.Logf("round 2")
	totalMessages := len(parties) * len(parties)
	round2s := RunRound[*round1, *round2](t, params, parties, round1s, totalMessages, outCh)
	t.Logf("round 3")
	round3s := RunRound[*round2, *round3](t, params, parties, round2s, len(parties), outCh)
	t.Logf("round 4")
	round4s := RunRound[*round3, *round4](t, params, parties, round3s, len(parties), outCh)
	t.Logf("round 5")
	round5s := RunRound[*round4, *round5](t, params, parties, round4s, len(parties), outCh)
	t.Logf("finalize")
	_ = RunRound[*round5, *finalization](t, params, parties, round5s, len(parties), outCh)
}
