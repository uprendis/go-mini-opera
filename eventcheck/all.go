package eventcheck

import (
	"github.com/Fantom-foundation/go-mini-opera/eventcheck/basiccheck"
	"github.com/Fantom-foundation/go-mini-opera/eventcheck/heavycheck"
	"github.com/Fantom-foundation/go-mini-opera/eventcheck/parentscheck"
	"github.com/Fantom-foundation/lachesis-base/eventcheck/epochcheck"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
)

// Checkers is collection of all the checkers
type Checkers struct {
	Basiccheck   *basiccheck.Checker
	Epochcheck   *epochcheck.Checker
	Parentscheck *parentscheck.Checker
	Heavycheck   *heavycheck.Checker
}

// Validate runs all the checks except Poset-related
func (v *Checkers) Validate(e dag.Event, parents dag.Events) error {
	if err := v.Basiccheck.Validate(e); err != nil {
		return err
	}
	if err := v.Epochcheck.Validate(e); err != nil {
		return err
	}
	if err := v.Parentscheck.Validate(e, parents); err != nil {
		return err
	}
	if err := v.Heavycheck.Validate(e); err != nil {
		return err
	}
	return nil
}
