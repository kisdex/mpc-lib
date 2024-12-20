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

package zkproofs_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kisdex/mpc-lib/common"
	"github.com/kisdex/mpc-lib/crypto"
	"github.com/kisdex/mpc-lib/crypto/zkproofs"
)

func TestLogStarProof(t *testing.T) {
	setUp(t)

	// witness
	witness := &zkproofs.LogStarWitness{
		X:   common.GetRandomPositiveInt(q),
		Rho: common.GetRandomPositiveInt(publicKey.N),
	}

	// X = g^x
	X := crypto.ScalarBaseMult(ec, witness.X)

	// C = Encrypt(N0, x, rho)
	C, err := publicKey.EncryptWithRandomness(witness.X, witness.Rho)
	assert.NoError(t, err, "encrypt C not error")

	statement := &zkproofs.LogStarStatement{
		Ell: ell,
		N0:  publicKey.N,
		C:   C,
		X:   X,
	}

	// Prove that:
	// X = g^x
	// C = Encrypt(N0, x, rho)
	proof := zkproofs.NewLogStarProof(witness, statement, ringPedersen)
	assert.NoError(t, err)
	assert.NotNil(t, proof)
	assert.True(t, proof.Verify(statement, ringPedersen), "proof failed to verify")
}

func TestLogStarGProof(t *testing.T) {
	setUp(t)

	// witness
	witness := &zkproofs.LogStarWitness{
		X:   common.GetRandomPositiveInt(q),
		Rho: common.GetRandomPositiveInt(publicKey.N),
	}

	// G = g^x
	G := crypto.ScalarBaseMult(ec, witness.X)

	// X = G^x
	X := G.ScalarMult(witness.X)

	// C = Encrypt(N0, x, rho)
	C, err := publicKey.EncryptWithRandomness(witness.X, witness.Rho)
	assert.NoError(t, err, "encrypt C not error")

	statement := &zkproofs.LogStarStatement{
		Ell: ell,
		N0:  publicKey.N,
		C:   C,
		X:   X,
		G:   G,
	}

	// Prove that:
	// X = G^x
	// C = Encrypt(N0, x, rho)
	proof := zkproofs.NewLogStarProof(witness, statement, ringPedersen)
	assert.NoError(t, err)
	assert.NotNil(t, proof)
	assert.True(t, proof.Verify(statement, ringPedersen), "proof failed to verify")
}

func GenerateLogStarData(t *testing.T) (*zkproofs.LogStarWitness, *zkproofs.LogStarStatement) {
	// witness
	witness := &zkproofs.LogStarWitness{
		X:   common.GetRandomPositiveInt(q),
		Rho: common.GetRandomPositiveInt(publicKey.N),
	}

	// X = g^x
	X := crypto.ScalarBaseMult(ec, witness.X)

	// C = Encrypt(N0, x, rho)
	C, err := publicKey.EncryptWithRandomness(witness.X, witness.Rho)
	assert.NoError(t, err, "encrypt C not error")

	statement := &zkproofs.LogStarStatement{
		Ell: ell,
		N0:  publicKey.N,
		C:   C,
		X:   X,
	}

	return witness, statement
}

func TestLogStarProofBytes(t *testing.T) {
	setUp(t)
	witness, statement := GenerateLogStarData(t)

	// Prove that:
	// X = g^x
	// C = Encrypt(N0, x, rho)
	proof := zkproofs.NewLogStarProof(witness, statement, ringPedersen)
	assert.NotNil(t, proof)
	assert.True(t, proof.Verify(statement, ringPedersen), "proof failed to verify")

	proofBytes := proof.Bytes()
	var proofInBytes [][]byte = proofBytes[:]
	np, err := new(zkproofs.LogStarProof).ProofFromBytes(ec, proofInBytes)
	newProof := np.(*zkproofs.LogStarProof)
	assert.NoError(t, err)
	assert.NotNil(t, newProof)
	assert.False(t, newProof.IsNil())
	assert.True(t, newProof.Verify(statement, ringPedersen))

}

func TestLogStarProofArrayBytes(t *testing.T) {
	setUp(t)
	witness, statement := GenerateLogStarData(t)

	proof := zkproofs.NewLogStarProof(witness, statement, ringPedersen)
	array := []*zkproofs.LogStarProof{proof, proof, nil, proof}
	bzs := zkproofs.ProofArrayToBytes(array)
	out, err := zkproofs.ProofArrayFromBytes[*zkproofs.LogStarProof](ec, bzs)
	assert.NoError(t, err)
	assert.Equal(t, len(array), len(out))
	assert.NotNil(t, out[0])
	assert.NotNil(t, out[1])
	assert.NotNil(t, out[3])
	assert.True(t, out[0].Verify(statement, ringPedersen))
	assert.True(t, out[1].Verify(statement, ringPedersen))
	assert.Nil(t, out[2])
	assert.True(t, out[3].Verify(statement, ringPedersen))
}
