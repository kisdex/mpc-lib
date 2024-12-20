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
	"errors"
	"sync"

	"github.com/kisdex/mpc-lib/crypto"
	"github.com/kisdex/mpc-lib/crypto/accmta"
	"github.com/kisdex/mpc-lib/crypto/zkproofs"
	"github.com/kisdex/mpc-lib/tss"
)

func (round *round2) Start() *tss.Error {
	if round.started {
		return round.WrapError(errors.New("round already started"))
	}
	round.number = 2
	round.started = true
	round.resetOK()

	i := round.PartyID().Index
	round.ok[i] = true

	partyCount := len(round.Parties().IDs())
	psi := Make2DSlice[*zkproofs.AffGInvProof](partyCount)
	psiHat := Make2DSlice[*zkproofs.AffGInvProof](partyCount)
	psiPrime := make([]*zkproofs.LogStarProof, partyCount)
	ec := round.Params().EC()
	round.temp.pointGamma[i] = crypto.ScalarBaseMult(ec, round.temp.gamma)

	errChs := make(chan *tss.Error, (len(round.Parties().IDs())-1)*3)
	round.VerifyRound1Messages(errChs)

	wg := sync.WaitGroup{}
	for j, Pj := range round.Parties().IDs() {
		if j == i {
			continue
		}

		wg.Add(3)
		go round.BobRespondsGamma(j, Pj, psi, &wg, errChs)
		go round.BobRespondsW(j, Pj, psiHat, &wg, errChs)
		go round.ComputeProofPsiPrime(j, Pj, psiPrime, &wg, errChs)
	}
	wg.Wait()
	close(errChs)
	err := round.WrapErrorChs(round.PartyID(), errChs, "Failed to process round 1 messages.")
	if err != nil {
		return err
	}

	for j, Pj := range round.Parties().IDs() {
		if j == i {
			continue
		}

		r2msg1 := NewSignRound2Message1(
			Pj, round.PartyID(),
			round.temp.bigD[i][j],
			round.temp.bigDHat[i][j],
			round.temp.bigF[i][j],
			round.temp.bigFHat[i][j],
			psi[j],
			psiHat[j],
		)
		round.temp.signRound2Message1s[i][j] = r2msg1
		round.out <- r2msg1
	}
	r2msg2 := NewSignRound2Message2(
		round.PartyID(),
		round.temp.pointGamma[i],
		psiPrime)
	round.temp.signRound2Message2s[i] = r2msg2
	round.out <- r2msg2

	return nil
}

func (round *round2) VerifyRound1Messages(errChs chan *tss.Error) {
	i := round.PartyID().Index
	for j, _ := range round.Parties().IDs() {
		if i == j {
			continue
		}
		r1msg := round.temp.signRound1Messages[j].Content().(*SignRound1Message)

		bigG := r1msg.UnmarshalBigG()
		round.temp.bigG[j] = bigG

		bigK := r1msg.UnmarshalBigK()
		round.temp.bigK[j] = bigK
	}
}

func (round *round2) BobRespondsW(j int, Pj *tss.PartyID, proofs [][]*zkproofs.AffGInvProof, wg *sync.WaitGroup, errChs chan *tss.Error) {
	defer wg.Done()
	i := round.PartyID().Index

	r1msg := round.temp.signRound1Messages[j].Content().(*SignRound1Message)
	psiAlice, err := r1msg.UnmarshalPsi()
	if err != nil {
		errChs <- round.WrapError(errors.New("UnmarshalPsi failed"), Pj)
		return
	}

	ringPedersenBobI := round.key.GetRingPedersen(i)
	rpVs := round.key.GetAllRingPedersen()
	rpVs[i] = nil
	betaHat, bigDHat, bigFHat, pf, err := accmta.BobRespondsG(
		round.Params().EC(),
		round.key.PaillierPKs[j],
		round.key.PaillierSK,
		psiAlice[i],
		round.temp.w,
		round.temp.bigK[j],
		rpVs,
		ringPedersenBobI,
	)
	if err != nil {
		errChs <- round.WrapError(errors.New("BobResponds(w) failed"), Pj)
		return
	}

	round.temp.betaHat[j] = betaHat
	round.temp.bigDHat[i][j] = bigDHat
	round.temp.bigFHat[i][j] = bigFHat
	proofs[j] = pf
}

func (round *round2) BobRespondsGamma(j int, Pj *tss.PartyID, proofs [][]*zkproofs.AffGInvProof, wg *sync.WaitGroup, errChs chan *tss.Error) {
	defer wg.Done()
	i := round.PartyID().Index

	r1msg := round.temp.signRound1Messages[j].Content().(*SignRound1Message)
	psiAlice, err := r1msg.UnmarshalPsi()
	if err != nil {
		errChs <- round.WrapError(errors.New("UnmarshalPsi failed"), Pj)
		return
	}

	ringPedersenBobI := round.key.GetRingPedersen(i)
	rpVs := round.key.GetAllRingPedersen()
	rpVs[i] = nil
	beta, bigD, bigF, pf, err := accmta.BobRespondsG(
		round.Params().EC(),
		round.key.PaillierPKs[j],
		round.key.PaillierSK,
		psiAlice[i],
		round.temp.gamma,
		round.temp.bigK[j],
		rpVs,
		ringPedersenBobI,
	)
	if err != nil {
		errChs <- round.WrapError(errors.New("BobResponds(gamma) failed"), Pj)
		return
	}

	round.temp.beta[j] = beta
	round.temp.bigD[i][j] = bigD
	round.temp.bigF[i][j] = bigF
	proofs[j] = pf
}

func (round *round2) ComputeProofPsiPrime(j int, Pj *tss.PartyID, proofs []*zkproofs.LogStarProof, wg *sync.WaitGroup, errChs chan *tss.Error) {
	defer wg.Done()
	i := round.PartyID().Index
	ec := round.Params().EC()

	_, rho, err := round.key.PaillierSK.DecryptFull(round.temp.bigG[i])
	if err != nil {
		errChs <- round.WrapError(errors.New("Error decrypting bigG"), Pj)
		return
	}
	witness := &zkproofs.LogStarWitness{
		X:   round.temp.gamma,
		Rho: rho,
	}
	statement := &zkproofs.LogStarStatement{
		Ell: zkproofs.GetEll(ec),
		N0:  round.key.PaillierSK.PublicKey.N,
		C:   round.temp.bigG[i],
		X:   round.temp.pointGamma[i],
	}

	rp := round.key.GetRingPedersen(j)
	proofs[j] = zkproofs.NewLogStarProof(witness, statement, rp)
}

func (round *round2) Update() (bool, *tss.Error) {
	for i, msgArray := range round.temp.signRound2Message1s {
		if i == round.PartyID().Index || round.ok[i] {
			continue
		}
		for j, msg := range msgArray {
			if i == j {
				continue
			}
			if msg == nil || !round.CanAccept(msg) {
				return false, nil
			}
		}
		msg2 := round.temp.signRound2Message2s[i]
		if msg2 == nil || !round.CanAccept(msg2) {
			return false, nil
		}
		round.ok[i] = true
	}
	return true, nil
}

func (round *round2) CanAccept(msg tss.ParsedMessage) bool {
	if _, ok := msg.Content().(*SignRound2Message1); ok {
		return msg.IsBroadcast()
	}
	if _, ok := msg.Content().(*SignRound2Message2); ok {
		return msg.IsBroadcast()
	}
	return false
}

func (round *round2) NextRound() tss.Round {
	round.started = false
	return &round3{round}
}
