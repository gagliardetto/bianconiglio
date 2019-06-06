package bianconiglio

// Credit: github.com/juju/errors/path.go
import (
	"runtime"
	"strings"
)

// SetLocation records the source location of the error at callDepth stack
// frames above the call.
func (e *marshalableError) SetLocation(callDepth int) {
	_, file, line, _ := runtime.Caller(callDepth + 1)
	e.Stack["file"] = trimGoPath(file)
	e.Stack["line"] = line
}

// prefixSize is used internally to trim the user specific path from the
// front of the returned filenames from the runtime call stack.
var prefixSize int

// goPath is the deduced path based on the location of this file as compiled.
var goPath string

func init() {
	_, file, _, ok := runtime.Caller(0)
	if file == "?" {
		return
	}
	if ok {
		// We know that the end of the file should be:
		// github.com/gagliardetto/bianconiglio/stack.go
		size := len(file)
		suffix := len("github.com/gagliardetto/bianconiglio/stack.go")
		goPath = file[:size-suffix]
		prefixSize = len(goPath)
	}
}

func trimGoPath(filename string) string {
	if strings.HasPrefix(filename, goPath) {
		return filename[prefixSize:]
	}
	return filename
}
