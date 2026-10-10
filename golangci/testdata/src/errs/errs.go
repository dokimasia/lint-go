// Package errs breaks the rule of errorprefix.
package errs

import "errors"

// ErrBare has no name of the package.
var ErrBare = errors.New("bare") // want `the text "bare" of errors.New does not start with "errs:", the name of its package`
