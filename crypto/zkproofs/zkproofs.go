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

package zkproofs

import (
	"crypto/elliptic"
	"fmt"
	"math/big"
	"strconv"

	"github.com/kisdex/mpc-lib/common"
)

type Proof interface {
	// returns true if nil
	IsNil() bool
	// returns byte encoding
	Bytes() [][]byte
	// length of Bytes() array
	Parts() int
	// decodes output of Bytes()
	ProofFromBytes(ec elliptic.Curve, bzs [][]byte) (Proof, error)
}

func ProofArrayToBytes[P Proof](proofs []P) [][]byte {
	if len(proofs) == 0 {
		return nil
	}
	parts := proofs[0].Parts()
	output := make([][]byte, parts*len(proofs))
	i := 0
	for _, proof := range proofs {
		if proof.IsNil() {
			for j := 0; j < parts; j += 1 {
				output[i] = nil
				i += 1
			}
		} else {
			pBytes := proof.Bytes()
			for _, ppBytes := range pBytes {
				output[i] = ppBytes
				i += 1
			}
		}
	}
	return output
}

func ProofArrayFromBytes[P Proof](ec elliptic.Curve, bzs [][]byte) ([]P, error) {
	pp := make([]P, 1)[0]
	parts := pp.Parts()
	if len(bzs)%parts != 0 {
		return nil, fmt.Errorf("Improper input length")
	}

	proofs := make([]P, len(bzs)/parts)
	for p, _ := range proofs {
		start := p * parts
		end := (p + 1) * parts
		slice := bzs[start:end]
		if common.NonEmptyMultiBytes(slice, len(slice)) {
			proof, err := pp.ProofFromBytes(ec, slice)
			if err != nil {
				return nil, err
			}
			proofs[p] = proof.(P)
		} else {
			// leave as nil
		}
	}
	return proofs, nil
}

// Constants derived from an instance of elliptic.Curve
type Ell struct {
	// bit length of ec.Params().N. Here N is the order of the base point
	//ec.Params().B, while |N| is the order of the subgroup generated by this point.
	Ell *big.Int

	// 2^ell
	TwoPowEll *big.Int

	// 3
	Epsilon *big.Int

	// ell + epislon
	EllPlusEpsilon *big.Int

	// 2^{ell+epsilon}
	TwoPowEllPlusEpsilon *big.Int
}

func NewEll(ell *big.Int) *Ell {
	two := big.NewInt(2)
	twoPowEll := new(big.Int).Exp(two, ell, nil)
	//epislon = 2*ell
	epsilon := new(big.Int).Mul(ell, two) // big.NewInt(3)
	ellPlusEpsilon := new(big.Int).Add(ell, epsilon)
	twoPowEllPlusEpsilon := new(big.Int).Exp(two, ellPlusEpsilon, nil)
	return &Ell{
		Ell:                  ell,
		TwoPowEll:            twoPowEll,
		Epsilon:              epsilon,
		EllPlusEpsilon:       ellPlusEpsilon,
		TwoPowEllPlusEpsilon: twoPowEllPlusEpsilon,
	}
}

func GetEll(ec elliptic.Curve) *big.Int {
	return big.NewInt(int64(ec.Params().BitSize))
}

func (ell *Ell) String() string {
	out := "Ell: " + ell.Ell.String()
	out += "\nEpsilon " + ell.Epsilon.String()
	out += "\n2^ell <= 2^ell+epsilon: " + strconv.FormatBool(ell.InRange(ell.TwoPowEll))
	out += "\n2^Ell " + ell.TwoPowEll.String()
	out += "\n2^Ell+Epislon " + ell.TwoPowEllPlusEpsilon.String()
	return out
}

// Returns true if val in [-2^{ell+epsilon}...+2^{ell+epsilon}]
func (ell *Ell) InRange(val *big.Int) bool {
	min := new(big.Int).Mul(big.NewInt(-1), ell.TwoPowEllPlusEpsilon)
	max := ell.TwoPowEllPlusEpsilon
	if val.Cmp(min) != 1 || val.Cmp(max) != -1 {
		return false
	}
	return true
}

// Returns true if val in [-2^{ell}...+2^{ell}]
func (ell *Ell) InRangeEll(val *big.Int) bool {
	min := new(big.Int).Mul(big.NewInt(-1), ell.TwoPowEll)
	max := ell.TwoPowEll
	if val.Cmp(min) != 1 || val.Cmp(max) != -1 {
		return false
	}
	return true
}

func Q(ec elliptic.Curve) *big.Int {
	return ec.Params().N
}

func IsZero(val *big.Int) bool {
	if val.Cmp(big.NewInt(0)) == 0 {
		return true
	}
	return false
}

// returns c = gamma^m * rho^N mod N^2
func PseudoPaillierEncrypt(gamma *big.Int, m *big.Int, rho *big.Int, N *big.Int, N2 *big.Int) *big.Int {
	// 1. Gm = gamma^m mod N2
	Gm := new(big.Int).Exp(gamma, m, N2)
	// 2. Xn = rho^N mod N2
	Xn := new(big.Int).Exp(rho, N, N2)
	// 3. (1) * (2) mod N2
	c := common.ModInt(N2).Mul(Gm, Xn)
	return c
}

// returns a Ring Pedersen commitment c = s^x * t^y mod N
type RingPedersenParams struct {
	S *big.Int
	T *big.Int
	N *big.Int
}

func (rp *RingPedersenParams) Commit(x *big.Int, y *big.Int) *big.Int {
	modNhat := common.ModInt(rp.N)
	sx := modNhat.Exp(rp.S, x)
	ty := modNhat.Exp(rp.T, y)
	return modNhat.Mul(sx, ty)
}

// returns a + bc
func APlusBC(a *big.Int, b *big.Int, c *big.Int) *big.Int {
	bc := new(big.Int).Mul(b, c)
	return new(big.Int).Add(a, bc)
}

// returns a * (b^c) mod N
func ATimesBToTheCModN(a *big.Int, b *big.Int, c *big.Int, N *big.Int) *big.Int {
	modN := common.ModInt(N)
	bc := modN.Exp(b, c)
	abc := modN.Mul(a, bc)
	return abc
}
