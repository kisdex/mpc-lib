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

package accmta_test

import (
	"crypto/elliptic"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kisdex/mpc-lib/common"
	"github.com/kisdex/mpc-lib/crypto"
	"github.com/kisdex/mpc-lib/crypto/accmta"
	"github.com/kisdex/mpc-lib/crypto/paillier"
	"github.com/kisdex/mpc-lib/crypto/zkproofs"
	"github.com/kisdex/mpc-lib/ecdsa/keygen"
	"github.com/kisdex/mpc-lib/tss"
)

var (
	skA, skB *paillier.PrivateKey
	pkA, pkB *paillier.PublicKey
	rpA, rpB *zkproofs.RingPedersenParams
	ec       elliptic.Curve
	q        *big.Int
	ell      *big.Int
)

func setUp(t *testing.T) {
	ec = tss.EC()
	q = ec.Params().N
	ell = zkproofs.GetEll(ec)
	assert.NotNil(t, ell)

	var err error
	skA, pkA, rpA, err = GetSavedKeys(0)
	assert.NoError(t, err)
	skB, pkB, rpB, err = GetSavedKeys(1)
	assert.NoError(t, err)
}

func GetSavedKeys(idx int) (sk *paillier.PrivateKey, pk *paillier.PublicKey, rp *zkproofs.RingPedersenParams, err error) {
	fixtures, _, err := keygen.LoadKeygenTestFixtures(idx + 1)
	if err != nil {
		return
	}
	fixture := fixtures[idx]
	rp = &zkproofs.RingPedersenParams{
		N: fixture.NTildei,
		S: fixture.H1i,
		T: fixture.H2i,
	}
	sk = fixture.PaillierSK
	pk = &paillier.PublicKey{N: sk.N}
	return
}

func TestMTA_P(t *testing.T) {
	setUp(t)

	a := common.GetRandomPositiveInt(q)
	ra := common.GetRandomPositiveInt(pkA.N)
	Xk, err := pkA.EncryptWithRandomness(a, ra)
	assert.NoError(t, err)
	b := common.GetRandomPositiveInt(q)

	rpVs := []*zkproofs.RingPedersenParams{rpA, rpA, nil, rpB}
	cA, proofsA, err := accmta.AliceInit(ec, pkA, a, ra, rpVs)
	assert.NoError(t, err)
	assert.NotNil(t, proofsA)
	assert.NotNil(t, cA)
	assert.Equal(t, 0, Xk.Cmp(cA))
	statementA := &zkproofs.EncStatement{
		K:  cA,    // Alice's ciphertext
		N0: pkA.N, // Alice's public key
		EC: ec,    // max size of plaintext
	}
	for i, rp := range rpVs {
		if i == 2 {
			continue
		}
		assert.True(t, proofsA[i].Verify(statementA, rp))
		assert.True(t, accmta.BobVerify(ec, pkA, proofsA[i], cA, rp))
	}

	cB, err := skB.Encrypt(b)
	assert.NoError(t, err)
	beta, cAlpha, cBeta, cBetaPrm, proofs, decProofs, err := accmta.BobRespondsP(ec, pkA, skB, proofsA[3], cB, cA, rpVs, rpB)
	assert.NoError(t, err)
	assert.NotNil(t, beta)
	assert.NotNil(t, cAlpha)
	assert.NotNil(t, cBetaPrm)
	assert.NotNil(t, cBeta)
	assert.NotNil(t, cB)
	assert.NotNil(t, proofs)
	assert.NotNil(t, decProofs)
	assert.Equal(t, len(rpVs), len(proofs))

	betaPrm, err := skB.Decrypt(cBetaPrm)
	assert.NoError(t, err)
	assert.True(t, common.ModInt(q).IsAdditiveInverse(beta, betaPrm))

	for i, _ := range rpVs {
		if rpVs[i] != nil {
			assert.NotNil(t, proofs[i])
		}
		assert.True(t, accmta.AliceVerifyP(ec, &skA.PublicKey, pkB, proofs[i], cA, cAlpha, cBetaPrm, cB, rpVs[i]))
		assert.True(t, accmta.DecProofVerify(pkB, ec, decProofs[i], cBeta, cBetaPrm, rpVs[i]))
	}
	alpha, err := accmta.AliceEndP(ec, skA, pkB, proofs[0], decProofs[0], cA, cAlpha, cBeta, cBetaPrm, cB, rpA)
	assert.NotNil(t, alpha)
	assert.NoError(t, err)

	// expect: alpha + beta = ab
	right := common.ModInt(q).Add(alpha, beta)
	left := common.ModInt(q).Mul(a, b)
	assert.Equal(t, 0, left.Cmp(right))
}

func TestDecTest(t *testing.T) {
	setUp(t)
	modQ := common.ModInt(q)
	zero := big.NewInt(0)

	betaPrm := common.GetRandomPositiveInt(q)
	beta := modQ.Sub(zero, betaPrm)
	assert.True(t, modQ.IsCongruent(zero, modQ.Add(beta, betaPrm)))

	cBeta, _, _ := pkB.EncryptAndReturnRandomness(beta)
	cBetaPrm, _, _ := pkB.EncryptAndReturnRandomness(betaPrm)
	cZero, _ := pkB.HomoAdd(cBeta, cBetaPrm) // actually should be q
	dZero, rho, _ := skB.DecryptFull(cZero)
	assert.Equal(t, 0, dZero.Cmp(q))
	assert.True(t, modQ.IsCongruent(dZero, zero))

	decStatement := &zkproofs.DecStatement{
		Q:   q,
		Ell: zkproofs.GetEll(tss.EC()),
		N0:  pkB.N,
		C:   cZero,
		X:   zero,
	}
	decWitness := &zkproofs.DecWitness{
		Y:   q,
		Rho: rho,
	}
	proof := zkproofs.NewDecProof(decWitness, decStatement, rpA)
	assert.NotNil(t, proof)
	assert.True(t, proof.Verify(decStatement, rpA))

}

func TestMTA_DL(t *testing.T) {
	setUp(t)

	a := common.GetRandomPositiveInt(q)
	ra := common.GetRandomPositiveInt(pkA.N)
	Xk, err := pkA.EncryptWithRandomness(a, ra)
	assert.NoError(t, err)
	b := common.GetRandomPositiveInt(q)

	rpVs := []*zkproofs.RingPedersenParams{rpA, rpA, nil, rpB}
	cA, proofsA, err := accmta.AliceInit(ec, pkA, a, ra, rpVs)
	assert.NoError(t, err)
	assert.NotNil(t, proofsA)
	assert.NotNil(t, cA)
	assert.Equal(t, 0, Xk.Cmp(cA))
	statementA := &zkproofs.EncStatement{
		K:  cA,    // Alice's ciphertext
		N0: pkA.N, // Alice's public key
		EC: ec,    // max size of plaintext
	}
	for i, rp := range rpVs {
		if i == 2 {
			continue
		}
		assert.True(t, proofsA[i].Verify(statementA, rp))
		assert.True(t, accmta.BobVerify(ec, pkA, proofsA[i], cA, rp))
	}

	B := crypto.ScalarBaseMult(ec, b)
	assert.NoError(t, err)
	beta, cAlpha, cBeta, cBetaPrm, proofs, decProofs, err := accmta.BobRespondsDL(ec, pkA, skB, proofsA[3], b, cA, rpVs, rpB, B)
	assert.NoError(t, err)
	assert.NotNil(t, beta)
	assert.NotNil(t, cAlpha)
	assert.NotNil(t, cBetaPrm)
	assert.NotNil(t, proofs)
	assert.Equal(t, len(rpVs), len(proofs))

	for i, _ := range rpVs {
		if rpVs[i] != nil {
			assert.NotNil(t, proofs[i])
		}
		assert.True(t, accmta.AliceVerifyDL(ec, &skA.PublicKey, pkB, proofs[i], cA, cAlpha, cBetaPrm, B, rpVs[i]))
		assert.True(t, accmta.DecProofVerify(pkB, ec, decProofs[i], cBeta, cBetaPrm, rpVs[i]))
	}
	alpha, err := accmta.AliceEndDL(ec, skA, pkB, proofs[0], decProofs[0], cA, cAlpha, cBeta, cBetaPrm, B, rpA)
	assert.NotNil(t, alpha)
	assert.NoError(t, err)

	// expect: alpha + beta = ab
	right := common.ModInt(q).Add(alpha, beta)
	left := common.ModInt(q).Mul(a, b)
	assert.Equal(t, 0, left.Cmp(right))
}

func TestMTA_G(t *testing.T) {
	setUp(t)

	a := common.GetRandomPositiveInt(q)
	ra := common.GetRandomPositiveInt(pkA.N)
	Xk, err := pkA.EncryptWithRandomness(a, ra)
	assert.NoError(t, err)
	b := common.GetRandomPositiveInt(q)

	rpVs := []*zkproofs.RingPedersenParams{rpA, rpA, nil, rpB}
	cA, proofsA, err := accmta.AliceInit(ec, pkA, a, ra, rpVs)
	assert.NoError(t, err)
	assert.NotNil(t, proofsA)
	assert.NotNil(t, cA)
	assert.Equal(t, 0, Xk.Cmp(cA))
	statementA := &zkproofs.EncStatement{
		K:  cA,    // Alice's ciphertext
		N0: pkA.N, // Alice's public key
		EC: ec,    // max size of plaintext
	}
	for i, rp := range rpVs {
		if i == 2 {
			continue
		}
		assert.True(t, proofsA[i].Verify(statementA, rp))
		assert.True(t, accmta.BobVerify(ec, pkA, proofsA[i], cA, rp))
	}

	B := crypto.ScalarBaseMult(ec, b)
	assert.NoError(t, err)
	beta, cAlpha, cBeta, proofs, err := accmta.BobRespondsG(ec, pkA, skB, proofsA[3], b, cA, rpVs, rpB)
	assert.NoError(t, err)
	assert.NotNil(t, beta)
	assert.NotNil(t, cAlpha)
	assert.NotNil(t, proofs)
	assert.Equal(t, len(rpVs), len(proofs))

	for i, _ := range rpVs {
		if rpVs[i] != nil {
			assert.NotNil(t, proofs[i])
		}
		assert.True(t, accmta.AliceVerifyG(ec, &skA.PublicKey, pkB, proofs[i], cA, cAlpha, cBeta, B, rpVs[i]))
	}
	alpha, err := accmta.AliceEndG(ec, skA, pkB, proofs[0], cA, cAlpha, cBeta, B, rpA)
	assert.NotNil(t, alpha)
	assert.NoError(t, err)

	// expect: alpha + beta = ab
	right := common.ModInt(q).Add(alpha, beta)
	left := common.ModInt(q).Mul(a, b)
	assert.Equal(t, 0, left.Cmp(right))
}
