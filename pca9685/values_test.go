package pca9685

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type testSuite struct {
	t *testing.T
	*require.Assertions
}

func (suite *testSuite) T() *testing.T {
	return suite.t
}

func (suite *testSuite) SetT(t *testing.T) {
	suite.t = t
	suite.Assertions = require.New(t)
}

func TestAll(t *testing.T) {
	suite.Run(t, new(testSuite))
}

// Examples from the PCA9685 manual page 17

func (s *testSuite) TestExample1() {
	onL, onH, offL, offH := ValuesDelayed(0.1, 0.2)
	s.Equal(byte(0x01), onH, "LED ON HIGH")
	s.Equal(byte(0x99), onL, "LED ON LOW")
	s.Equal(byte(0x04), offH, "LED OFF HIGH")
	s.Equal(byte(0xcc), offL, "LED OFF LOW")
}

func (s *testSuite) TestExample2() {
	onL, onH, offL, offH := ValuesDelayed(0.9, 0.9)
	s.Equal(byte(0x0e), onH, "LED ON HIGH")
	s.Equal(byte(0x65), onL, "LED ON LOW")
	s.Equal(byte(0x0c), offH, "LED OFF HIGH")
	s.Equal(byte(0xcb), offL, "LED OFF LOW")
}

// Example from the PCA9685 manual page 25

func (s *testSuite) TestPrescale() {
	s.Equal(FREQ_MIN_PRESALE, Prescaler(FREQ_MIN), "min freq prescale")
	s.Equal(FREQ_MAX_PRESCALE, Prescaler(FREQ_MAX), "max freq prescale")
	s.Equal(byte(0x1e), Prescaler(200), "example prescale")
}
