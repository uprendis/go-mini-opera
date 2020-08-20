package parentscheck

import (
	"errors"

	base "github.com/Fantom-foundation/lachesis-base/eventcheck/parentscheck"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
)

var (
	ErrPastTime = errors.New("event has lower claimed time than self-parent")
)

// Checker which require only parents list + current epoch info
type Checker struct {
	base *base.Checker
}

// New validator which performs checks, which require known the parents
func New() *Checker {
	return &Checker{
		base: &base.Checker{},
	}
}

// Validate event
func (v *Checker) Validate(e dag.Event, parents dag.Events) error {
	if err := v.base.Validate(e, parents); err != nil {
		return err
	}

	if e.SelfParent() != nil {
		selfParent := parents[0]
		if !e.IsSelfParent(selfParent.ID()) {
			// sanity check, self-parent is always first, it's how it's stored
			return base.ErrWrongSelfParent
		}
		// selfParent time
		if e.RawTime() <= selfParent.RawTime() {
			return ErrPastTime
		}
	}

	return nil
}
