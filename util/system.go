package util

import (
	"runtime/debug"
	"flag"
	"github.com/cihub/seelog"
	"os"
	"encoding/json"
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

func JsonEncode(data interface{}) []byte {
	res, err := json.Marshal(data)
	if (err != nil) {
		seelog.Errorf("json_encode error: %v", err.Error())
	}
	return res
}
