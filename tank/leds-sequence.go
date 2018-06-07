package tank

import (
	"fmt"
	"math"
	"time"
)

const (
	LedPeakRadius     = 4                       // Number of LEDs around the brightness peak, that are not dark
	LedSleepTime      = 50 * time.Millisecond   // Time resolution for LED updates
	LedPeakTravelTime = 1300 * time.Millisecond // Time for one brightness peak to travel all LEDs

	LedStepsPerRound = float64(LedPeakTravelTime / LedSleepTime)
	LedTimeStep      = NumLeds / LedStepsPerRound
)

func RunLedStartupSequence(numRounds int, callback func(sleepTime time.Duration, values []float64) error) error {
	values := make([]float64, NumLeds)
	numSteps := LedStepsPerRound * float64(numRounds)
	for i := float64(0); i < numSteps; i++ {
		_setLedValuesCircling(false, i, values)
		// _setLedValuesCircling(true, i, values)
		// _setLedValuesBouncing(i, values)

		// Reorder the middle 5 leds
		values[5], values[6], values[7], values[8], values[9] = values[9], values[8], values[7], values[6], values[5]

		if err := callback(LedSleepTime, values); err != nil {
			return fmt.Errorf("Error during LED startup sequence, step %v of %v: %v", i, numSteps, err)
		}
	}
	return nil
}

func _setLedValuesBouncing(x float64, values []float64) {
	t := x * LedTimeStep
	max := float64(len(values))
	mid := t - math.Floor(t/max)*max

	// TODO bounce fixen

	for i := range values {
		x := float64(i) - mid

		// Special cases for calculating distance wrapping around the 0 and max
		// Could probably be calculated more elegantly
		x2 := max - mid + float64(i)
		if x < 0 && x2 < math.Abs(x) {
			x = -x2
		}
		x3 := max - float64(i) + mid
		if x3 < x {
			x = -x3
		}

		if math.Abs(x) > LedPeakRadius {
			values[i] = 0
		} else {
			v := math.Cos(x / LedPeakRadius * math.Pi)
			values[i] = (v + 1) / 2 // Map to 0..1
		}
	}
}

func _setLedValuesCircling(bounce bool, x float64, values []float64) {
	if bounce {
		max := 3.2 * float64(len(values))
		x = x - math.Floor(x/max)*max
		if x > max/2 {
			x = max - x
		}
	}

	t := x * LedTimeStep
	max := float64(len(values))
	mid := t - math.Floor(t/max)*max

	for i := range values {
		x := float64(i) - mid

		// Special cases for calculating distance wrapping around the 0 and max
		// Could probably be calculated more elegantly
		x2 := max - mid + float64(i)
		if x < 0 && x2 < math.Abs(x) {
			x = -x2
		}
		x3 := max - float64(i) + mid
		if x3 < x {
			x = -x3
		}

		if math.Abs(x) > LedPeakRadius {
			values[i] = 0
		} else {
			v := math.Cos(x / LedPeakRadius * math.Pi)
			values[i] = (v + 1) / 2 // Map to 0..1
		}
	}
}
