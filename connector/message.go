package connector

import (
	"bytes"
	"github.com/cihub/seelog"
)

func FormatMsg(data []byte) [][]byte {
	var message = bytes.SplitN(data, comma, 2)

	if (len(message) != 2) {
		seelog.Errorf("received message format is error: %s", bytes.TrimRight(data, "\n"))
		return nil
	}

	return message;
}
