// Command greet is the command of the cases of errorprefix in a package main.
package main

import "errors"

// errNoName has the name of the command and a colon.
var errNoName = errors.New("greet: no name given")

// errPackage has the name of the package main in place of the name of the command.
var errPackage = errors.New("main: no name given") // want `the text "main: no name given" of errors.New does not start with "greet:", the name of its command`

func main() {
	_, _ = errNoName, errPackage
}
