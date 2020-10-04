package ft260

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_i2c_split_transactions(t *testing.T) {
	a := assert.New(t)
	test := func(stop bool, data []byte, expectedPayload [][]byte, expectedConditions []byte) {
		payload, conditions := i2cSplitTransaction(stop, data)
		a.Equal(expectedPayload, payload, "Payload differs")
		a.Equal(expectedConditions, conditions, "Conditions differ")
	}

	test(true, nil, nil, nil)
	test(false, nil, nil, nil)
	test(true, []byte{}, nil, nil)
	test(false, []byte{}, nil, nil)

	test(true, []byte{44}, [][]byte{[]byte{44}}, []byte{I2C_MasterStartStop})
	test(false, []byte{44}, [][]byte{[]byte{44}}, []byte{I2C_MasterStart})

	data := make([]byte, 130)
	for i := byte(0); i < byte(len(data)); i++ {
		data[i] = i + 10
	}

	// 59 byte
	test(true, data[:59], [][]byte{data[:59]}, []byte{I2C_MasterStartStop})
	test(false, data[:59], [][]byte{data[:59]}, []byte{I2C_MasterStart})

	// 60 byte
	test(true, data[:60], [][]byte{data[:60]}, []byte{I2C_MasterStartStop})
	test(false, data[:60], [][]byte{data[:60]}, []byte{I2C_MasterStart})

	// 61 byte
	test(true, data[:61], [][]byte{data[:60], data[60:61]}, []byte{I2C_MasterStart, I2C_MasterStop})
	test(false, data[:61], [][]byte{data[:60], data[60:61]}, []byte{I2C_MasterStart, I2C_MasterNone})

	// 119 byte
	test(true, data[:119], [][]byte{data[:60], data[60:119]}, []byte{I2C_MasterStart, I2C_MasterStop})
	test(false, data[:119], [][]byte{data[:60], data[60:119]}, []byte{I2C_MasterStart, I2C_MasterNone})

	// 120 byte
	test(true, data[:120], [][]byte{data[:60], data[60:120]}, []byte{I2C_MasterStart, I2C_MasterStop})
	test(false, data[:120], [][]byte{data[:60], data[60:120]}, []byte{I2C_MasterStart, I2C_MasterNone})

	// 121 byte
	test(true, data[:121], [][]byte{data[:60], data[60:120], data[120:121]}, []byte{I2C_MasterStart, I2C_MasterNone, I2C_MasterStop})
	test(false, data[:121], [][]byte{data[:60], data[60:120], data[120:121]}, []byte{I2C_MasterStart, I2C_MasterNone, I2C_MasterNone})

	// 130 byte
	test(true, data, [][]byte{data[:60], data[60:120], data[120:]}, []byte{I2C_MasterStart, I2C_MasterNone, I2C_MasterStop})
	test(false, data, [][]byte{data[:60], data[60:120], data[120:]}, []byte{I2C_MasterStart, I2C_MasterNone, I2C_MasterNone})
}
