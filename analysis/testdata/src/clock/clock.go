// Package clock is the package of the cases of errorprefix.
package clock

import (
	"errors"
	stderrors "errors"

	"other"
)

// text is a constant text without the name of the package.
const text = "constant text"

// ErrZero has the name of the package and a colon.
var ErrZero = errors.New("clock: instant is zero")

// ErrWrapped is a call of errors.New in the argument of another call.
var ErrWrapped = wrap(errors.New("wrapped")) // want `the text "wrapped" of errors.New does not start with "clock:", the name of its package`

// ErrNested is a call of errors.New in the argument of a call of errors.New whose text is no
// constant.
var ErrNested = errors.New(errors.New("nested").Error()) // want `the text "nested" of errors.New does not start with "clock:", the name of its package`

// size is a call of a builtin function, which is no call of errors.New.
var size = len(text)

// ErrPatch has the name of the package and the qualifier patch.
var ErrPatch = errors.New("clock.patch: malformed")

// ErrBare has no name of the package.
var ErrBare = errors.New("instant is zero") // want `the text "instant is zero" of errors.New does not start with "clock:", the name of its package`

// ErrAliased is a call of errors.New under another name.
var ErrAliased = stderrors.New("aliased") // want `the text "aliased" of errors.New does not start with "clock:", the name of its package`

// ErrConstant is a call of errors.New with a constant.
var ErrConstant = errors.New(text) // want `the text "constant text" of errors.New does not start with "clock:", the name of its package`

// ErrOther is a call of a function New of another package.
var ErrOther = other.New("other text")

// Dynamic returns an error whose text is no constant.
func Dynamic(s string) error {
	return errors.New(s)
}

// Message returns the text of err through the method Error of the interface error.
func Message(err error) string {
	return err.Error()
}

// wrap returns err.
func wrap(err error) error {
	return err
}
