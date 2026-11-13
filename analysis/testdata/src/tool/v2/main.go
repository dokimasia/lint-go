// Command tool is the command of the cases of errorprefix at a major version of its import path.
package main

import "errors"

// errFlag has the name of the command and a colon.
var errFlag = errors.New("tool: unknown flag")

// errVersion has the major version of the import path in place of the name of the command.
var errVersion = errors.New("v2: unknown flag") // want `the text "v2: unknown flag" of errors.New does not start with "tool:", the name of its command`

func main() {
	_, _ = errFlag, errVersion
}
