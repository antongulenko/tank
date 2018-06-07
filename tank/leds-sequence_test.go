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
	err := RunLedStartupSequence(numRounds, func(sleepTime time.Duration, values []float64) error {
		a.Equal(LedSleepTime, sleepTime, "wrong sleep time")
		a.Equal(NumLeds, len(values))
		actual = append(actual, values[0])
		i++
		return nil
	})
	a.Nil(err)
	a.Equal(int(LedStepsPerRound*float64(numRounds)), len(actual))
	// TODO compare values to real sine curve?}
}
