package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type P struct {
	l, r float64
}

func TestMotorSingleStick(t *testing.T) {
	a := assert.New(t)
	test := func(x, y, expectL, expectR float64) {
		l, r := convertStickToDirections(x, y)
		a.Equal(P{expectL, expectR}, P{l, r})
	}

	// Straight forward/backward
	test(0, 0, 0, 0)
	test(0, 1, 1, 1)
	test(0, -1, -1, -1)
	test(0, 0.5, 0.5, 0.5)
	test(0, -0.5, -0.5, -0.5)

	// Maximum turn left/right
	test(1, 0, 1, -1)
	test(0.5, 0, 0.5, -0.5)
	test(-1, 0, -1, 1)
	test(-0.5, 0, -0.5, 0.5)

	// Slight turn right
	intermediate := 0.40966552939826695
	test(1, 1, 1, 0)
	test(0.5, 1, 1, intermediate)
	test(1, 0.5, 1, -intermediate)

	// Slight turn left
	test(-1, 1, 0, 1)
	test(-0.5, 1, intermediate, 1)
	test(-1, 0.5, -intermediate, 1)

	// Slight turn backward left (actually steers backward right)
	test(-1, -1, -1, 0)
	test(-1, -0.5, -1, intermediate)
	test(-0.5, -1, -1, -intermediate)

	// Slight turn backward right (actually steers backward left)
	test(1, -1, 0, -1)
	test(1, -0.5, intermediate, -1)
	test(0.5, -1, -intermediate, -1)

	// Slight turn right with lower power
	test(0.2, 0.2, 0.28284271247461906, 0)
	test(0.1, 0.2, 0.223606797749979, 0.09160399717729635)
	test(0.2, 0.1, 0.223606797749979, -0.09160399717729635)
}
