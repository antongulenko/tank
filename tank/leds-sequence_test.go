package tank

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStartupSequence(t *testing.T) {
	a := assert.New(t)

	numRounds := 4
	var actual []float64
	i := 0
	seq := DefaultLedSequence
	err := seq.Run(numRounds, func(sleepTime time.Duration, values []float64) error {
		a.Equal(seq.SleepTime, sleepTime, "wrong sleep time")
		a.Equal(seq.NumLeds, len(values))
		actual = append(actual, values[0])
		i++
		return nil
	})
	a.Nil(err)

	x := float64(seq.PeakTravelTime / seq.SleepTime)
	a.Equal(int(x*float64(numRounds)), len(actual))
	// TODO compare values to real sine curve?}
}
