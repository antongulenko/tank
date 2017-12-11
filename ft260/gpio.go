package ft260

const (
	ReportID_GPIO = 0xB0 // Feature
)

// ReportID_GPIO Feature In and Out
type ReportGpio struct {
	Value   byte // GPIO 0-5 bits
	Dir     byte // GPIO 0-5 direction bits
	ValueEx byte // GPIO A-H bits
	DirEx   byte // GPIO A-H direction bits
}

func (r ReportGpio) ReportID() byte {
	return ReportID_GPIO
}

func (r ReportGpio) ReportLen() int {
	return 5
}

func (r ReportGpio) Marshall(b []byte) error {
	b[1] = r.Value
	b[2] = r.Dir
	b[3] = r.ValueEx
	b[4] = r.DirEx
	return nil
}

func (r ReportGpio) Unmarshall(b []byte) error {
	r.Value = b[1]
	r.Dir = b[2]
	r.ValueEx = b[3]
	r.DirEx = b[4]
	return nil
}
