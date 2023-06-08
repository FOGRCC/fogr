// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/fogr/blob/master/LICENSE

package das

import (
	"time"

	"github.com/FOGRCC/fogr/fogstate"
	"github.com/FOGRCC/fogr/util/pretty"
	"github.com/ethereum/go-ethereum/log"
)

func logPut(store string, data []byte, timeout uint64, reader fogstate.DataAvailabilityReader, more ...interface{}) {
	if len(more) == 0 {
		log.Trace(
			store, "message", pretty.FirstFewBytes(data), "timeout", time.Unix(int64(timeout), 0),
			"this", reader,
		)
	} else {
		log.Trace(
			store, "message", pretty.FirstFewBytes(data), "timeout", time.Unix(int64(timeout), 0),
			"this", reader, more,
		)
	}
}
