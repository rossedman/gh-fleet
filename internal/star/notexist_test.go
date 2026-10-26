package star

import "runtime"

// noSuchFile is how the OS words a missing file in an error.
var noSuchFile = func() string {
	if runtime.GOOS == "windows" {
		return "cannot find the"
	}
	return "no such file"
}()
