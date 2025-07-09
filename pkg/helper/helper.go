package helper


import (
	"runtime"
)

func GetFuncName() string {
    pc, _, _, _ := runtime.Caller(1)
    return runtime.FuncForPC(pc).Name()
}