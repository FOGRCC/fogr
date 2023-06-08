package sharedmetrics

import (
	"github.com/FOGRCC/fogr/fogutil"
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	latestSequenceNumberGauge  = metrics.NewRegisteredGauge("fogr/sequencenumber/latest", nil)
	sequenceNumberInBlockGauge = metrics.NewRegisteredGauge("fogr/sequencenumber/inblock", nil)
)

func UpdateSequenceNumberGauge(sequenceNumber fogutil.MessageIndex) {
	latestSequenceNumberGauge.Update(int64(sequenceNumber))
}
func UpdateSequenceNumberInBlockGauge(sequenceNumber fogutil.MessageIndex) {
	sequenceNumberInBlockGauge.Update(int64(sequenceNumber))
}
