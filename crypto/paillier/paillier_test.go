// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.
//
// Portions Copyright (c) 2023, Circle Internet Financial, LTD.  All rights reserved
// Circle contributions are licensed under the Apache 2.0 License.
//
// SPDX-License-Identifier: Apache-2.0 AND MIT

package paillier_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kisdex/mpc-lib/common"
	"github.com/kisdex/mpc-lib/crypto"
	. "github.com/kisdex/mpc-lib/crypto/paillier"
	"github.com/kisdex/mpc-lib/tss"
)

// Using a modulus length of 2048 is recommended in the GG18 spec
const (
	testPaillierKeyLength = 2048
)

var (
	privateKey *PrivateKey
	publicKey  *PublicKey
)

func setUp(t *testing.T) {
	if privateKey != nil && publicKey != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var err error
	privateKey, publicKey, err = GenerateKeyPair(ctx, testPaillierKeyLength)
	assert.NoError(t, err)
}

func TestGenerateKeyPair(t *testing.T) {
	setUp(t)
	assert.NotZero(t, publicKey)
	assert.NotZero(t, privateKey)
}

func TestEncrypt(t *testing.T) {
	setUp(t)
	cipher, err := publicKey.Encrypt(big.NewInt(1))
	assert.NoError(t, err, "must not error")
	assert.NotZero(t, cipher)
}

func TestEncryptDecrypt(t *testing.T) {
	setUp(t)
	exp := big.NewInt(100)
	cypher, err := privateKey.Encrypt(exp)
	if err != nil {
		t.Error(err)
	}
	ret, err := privateKey.Decrypt(cypher)
	assert.NoError(t, err)
	assert.Equal(t, 0, exp.Cmp(ret),
		"wrong decryption ", ret, " is not ", exp)

	cypher = new(big.Int).Set(privateKey.N)
	_, err = privateKey.Decrypt(cypher)
	assert.Error(t, err)
}

func TestDecryptFull(t *testing.T) {
	setUp(t)
	exp := big.NewInt(100)
	cypher, rho, err := privateKey.EncryptAndReturnRandomness(exp)
	if err != nil {
		t.Error(err)
	}
	ret, retRho, err := privateKey.DecryptFull(cypher)
	assert.NoError(t, err)
	assert.Equal(t, 0, exp.Cmp(ret),
		"wrong decryption ", ret, " is not ", exp)
	assert.Equal(t, 0, rho.Cmp(retRho),
		"wrong decryption of rho ", retRho, " is not ", ret)

	cypher = new(big.Int).Set(privateKey.N)
	_, _, err = privateKey.DecryptFull(cypher)
	assert.Error(t, err)
}

func TestHomoMul(t *testing.T) {
	setUp(t)
	three, err := privateKey.Encrypt(big.NewInt(3))
	assert.NoError(t, err)

	// for HomoMul, the first argument `m` is not ciphered
	six := big.NewInt(6)

	cm, err := privateKey.HomoMult(six, three)
	assert.NoError(t, err)
	multiple, err := privateKey.Decrypt(cm)
	assert.NoError(t, err)

	// 3 * 6 = 18
	exp := int64(18)
	assert.Equal(t, 0, multiple.Cmp(big.NewInt(exp)))
}

func TestHomoMulAndReturnRandomness(t *testing.T) {
	setUp(t)
	three, err := privateKey.Encrypt(big.NewInt(3))
	assert.NoError(t, err)

	// for HomoMul, the first argument `m` is not ciphered
	six := big.NewInt(6)

	cm, rho, err := privateKey.HomoMultAndReturnRandomness(six, three)
	assert.NoError(t, err)
	multiple, err := privateKey.Decrypt(cm)
	assert.NoError(t, err)

	// 3 * 6 = 18
	exp := int64(18)
	assert.Equal(t, 0, multiple.Cmp(big.NewInt(exp)))

	// check randomness
	N2 := new(big.Int).Mul(publicKey.N, publicKey.N)
	rhoN := new(big.Int).Exp(rho, publicKey.N, N2)
	eighteen, _ := publicKey.HomoMult(six, three)
	expectedcm := common.ModInt(N2).Mul(eighteen, rhoN)
	assert.Equal(t, 0, expectedcm.Cmp(cm))
}

func TestMultInv(t *testing.T) {
	setUp(t)
	num := big.NewInt(2343)
	zero := big.NewInt(0)
	q := tss.EC().Params().N

	cipher, _ := publicKey.Encrypt(num)
	inv, _ := publicKey.HomoMultInv(cipher)
	negNum, _ := privateKey.Decrypt(inv)
	NMinusNum := new(big.Int).Sub(publicKey.N, num)
	actual := common.ModInt(publicKey.N).Add(num, negNum)

	assert.True(t, common.ModInt(publicKey.N).IsCongruent(zero, actual))
	assert.True(t, common.ModInt(publicKey.N).IsAdditiveInverse(num, negNum))
	assert.Equal(t, 0, negNum.Cmp(NMinusNum))
	assert.True(t, common.ModInt(q).IsCongruent(zero, actual))
	assert.False(t, common.ModInt(q).IsAdditiveInverse(num, negNum))
}

func TestHomoAdd(t *testing.T) {
	setUp(t)
	num1 := big.NewInt(10)
	num2 := big.NewInt(32)

	one, _ := publicKey.Encrypt(num1)
	two, _ := publicKey.Encrypt(num2)

	ciphered, _ := publicKey.HomoAdd(one, two)

	plain, _ := privateKey.Decrypt(ciphered)

	assert.Equal(t, new(big.Int).Add(num1, num2), plain)
}

func TestProofVerify(t *testing.T) {
	setUp(t)
	ki := common.MustGetRandomInt(256)                     // index
	ui := common.GetRandomPositiveInt(tss.EC().Params().N) // ECDSA private
	yX, yY := tss.EC().ScalarBaseMult(ui.Bytes())          // ECDSA public
	proof := privateKey.Proof(ki, crypto.NewECPointNoCurveCheck(tss.EC(), yX, yY))
	res, err := proof.Verify(publicKey.N, ki, crypto.NewECPointNoCurveCheck(tss.EC(), yX, yY))
	assert.NoError(t, err)
	assert.True(t, res, "proof verify result must be true")
}

func TestProofVerifyFail(t *testing.T) {
	setUp(t)
	ki := common.MustGetRandomInt(256)                     // index
	ui := common.GetRandomPositiveInt(tss.EC().Params().N) // ECDSA private
	yX, yY := tss.EC().ScalarBaseMult(ui.Bytes())          // ECDSA public
	proof := privateKey.Proof(ki, crypto.NewECPointNoCurveCheck(tss.EC(), yX, yY))
	last := proof[len(proof)-1]
	last.Sub(last, big.NewInt(1))
	res, err := proof.Verify(publicKey.N, ki, crypto.NewECPointNoCurveCheck(tss.EC(), yX, yY))
	assert.NoError(t, err)
	assert.False(t, res, "proof verify result must be true")
}

func TestComputeL(t *testing.T) {
	u := big.NewInt(21)
	n := big.NewInt(3)

	expected := big.NewInt(6)
	actual := L(u, n)

	assert.Equal(t, 0, expected.Cmp(actual))
}

func TestGenerateXs(t *testing.T) {
	k := common.MustGetRandomInt(256)
	sX := common.MustGetRandomInt(256)
	sY := common.MustGetRandomInt(256)
	N := common.GetRandomPrimeInt(2048)

	xs := GenerateXs(13, k, N, crypto.NewECPointNoCurveCheck(tss.EC(), sX, sY))
	assert.Equal(t, 13, len(xs))
	for _, xi := range xs {
		assert.True(t, common.IsNumberInMultiplicativeGroup(N, xi))
	}
}
