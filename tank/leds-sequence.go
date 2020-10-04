package tank

import (
	"fmt"
	"math"
	"time"
)

var DefaultLedSequence = LedSequence{
	Circle:         true,
	Bounce:         false,
	NumLeds:        15,
	PeakRadius:     4,
	SleepTime:      50 * time.Millisecond,
	PeakTravelTime: 1300 * time.Millisecond,
}

type LedSequence struct {
	Circle         bool
	Bounce         bool
	NumLeds        int
	PeakRadius     int           // Number of LEDs around the brightness peak, that are not dark
	SleepTime      time.Duration // Time resolution for LED updates
	PeakTravelTime time.Duration // Time for one brightness peak to travel all LEDs
}

func (s *LedSequence) Run(numRounds int, callback func(sleepTime time.Duration, values []float64) error) error {
	stepsPerRound := float64(s.PeakTravelTime / s.SleepTime)
	timeStep := float64(s.NumLeds) / stepsPerRound

	values := make([]float64, s.NumLeds)
	numSteps := stepsPerRound * float64(numRounds)
	for i := float64(0); i < numSteps; i++ {
		if s.Circle {
			s.setLedValuesCircling(timeStep, s.Bounce, i, values)
		} else {
			s.setLedValuesBouncing(timeStep, i, values)
		}

		// Reorder the middle 5 leds
		values[5], values[6], values[7], values[8], values[9] = values[9], values[8], values[7], values[6], values[5]

		if err := callback(s.SleepTime, values); err != nil {
			return fmt.Errorf("Error during LED sequence, step %v of %v: %v", i, numSteps, err)
		}
	}
	return nil
}

func (s *LedSequence) setLedValuesBouncing(timeStep float64, x float64, values []float64) {
	t := x * timeStep
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

		if math.Abs(x) > float64(s.PeakRadius) {
			values[i] = 0
		} else {
			v := math.Cos(x / float64(s.PeakRadius) * math.Pi)
			values[i] = (v + 1) / 2 // Map to 0..1
		}
	}
}

func (s *LedSequence) setLedValuesCircling(timeStep float64, bounce bool, x float64, values []float64) {
	if bounce {
		max := 3.2 * float64(len(values))
		x = x - math.Floor(x/max)*max
		if x > max/2 {
			x = max - x
		}
	}

	t := x * timeStep
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

		if math.Abs(x) > float64(s.PeakRadius) {
			values[i] = 0
		} else {
			v := math.Cos(x / float64(s.PeakRadius) * math.Pi)
			values[i] = (v + 1) / 2 // Map to 0..1
		}
	}
}
