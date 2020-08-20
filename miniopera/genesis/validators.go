package genesis

import (
	"crypto/sha256"
	"math/big"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/common"
)

type (
	// Validator is a helper structure to define genesis validators
	Validator struct {
		ID     idx.ValidatorID
		PubKey []byte
		Stake  *big.Int
	}

	Validators []Validator
)

// Build converts Validators to Validators
func (gv Validators) Build() *pos.Validators {
	builder := pos.NewBigBuilder()
	for _, validator := range gv {
		builder.Set(validator.ID, validator.Stake)
	}
	return builder.Build()
}

// TotalStake returns sum of stakes
func (gv Validators) TotalStake() *big.Int {
	totalStake := new(big.Int)
	for _, validator := range gv {
		totalStake.Add(totalStake, validator.Stake)
	}
	return totalStake
}

// Map converts Validators to map
func (gv Validators) Map() map[idx.ValidatorID]Validator {
	validators := map[idx.ValidatorID]Validator{}
	for _, validator := range gv {
		validators[validator.ID] = validator
	}
	return validators
}

// Addresses returns not sorted genesis addresses
func (gv Validators) PubKeys() [][]byte {
	res := make([][]byte, 0, len(gv))
	for _, v := range gv {
		res = append(res, v.PubKey)
	}
	return res
}

// Hash returns data digest
func (gv Validators) Hash() common.Hash {
	hasher := sha256.New()
	for _, v := range gv {
		hasher.Write(v.ID.Bytes())
		hasher.Write(v.PubKey)
		hasher.Write([]byte{uint8((len(v.Stake.Bytes())))})
		hasher.Write(v.Stake.Bytes())
	}
	return common.BytesToHash(hasher.Sum(nil))
}
