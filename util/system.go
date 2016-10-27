package util

import (
	"runtime/debug"
	"flag"
	"github.com/cihub/seelog"
	"os"
)

var isDev = flag.Bool("dev", true, "Is dev model")

func IsDev() bool {
	return *isDev
}

func PanicExit() {
	if err := recover(); err != nil {
		seelog.Criticalf("%v\n%s\n======================================================\n", err, debug.Stack())
		os.Exit(1)
	}
}
