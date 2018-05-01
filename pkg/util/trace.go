package util

import "runtime"

// GetCurrentFuncName get current function name
func GetCurrentFuncName() string {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	return runtime.FuncForPC(pc[0]).Name()
}
