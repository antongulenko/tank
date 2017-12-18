package ft260

import (
	"errors"
	"fmt"

	"time"

	"github.com/antongulenko/hid"
	log "github.com/sirupsen/logrus"
)

const (
	FTDIVendorId   = 0x0403
	FT260ProductId = 0x6030

	ReadReportTimeout = 500 * time.Millisecond
)

type Ft260Driver struct {
	Vendor  uint16
	Product uint16
}

func (d *Ft260Driver) Open() (*Ft260, error) {
	return d.OpenPath("")
}

func (d *Ft260Driver) OpenPath(path string) (*Ft260, error) {
	if !hid.Supported() {
		return nil, errors.New("This libray github.com/karalabe/hid is not supported on this platform")
	}
	vendor, product := d.Vendor, d.Product
	if vendor == 0 {
		vendor = FTDIVendorId
	}
	if product == 0 {
		product = FT260ProductId
	}
	devices := hid.Enumerate(vendor, product)
	if len(devices) == 0 {
		return nil, fmt.Errorf("No USB HID device found with vendorID=%04x productID=%04x", vendor, product)
	} else {
		var info hid.DeviceInfo
		if len(devices) == 1 {
			info = devices[0]
			if path != "" && info.Path != path {
				return nil, fmt.Errorf("Given device path %v does not match %v", path, info.Path)
			}
		} else {
			if path != "" {
				found := false
				for _, dev := range devices {
					if dev.Path == path {
						info = dev
						found = true
						break
					}
				}
				if !found {
					log.Warnf("%v devices connected with vendorID=%04x productID=%04x", len(devices), vendor, product)
					for i, dev := range devices {
						log.Warnf("Device %v: %v (USB %v): %v (%04x) from %v (%04x), Release %v",
							i, dev.Path, dev.Interface, dev.Product, dev.ProductID, dev.Manufacturer, dev.VendorID, dev.Release)
					}
					return nil, errors.New("No device matched the given device path " + path)
				}
			} else {
				log.Warnf("%v devices connected with vendorID=%04x productID=%04x", len(devices), vendor, product)
				for i, dev := range devices {
					log.Warnf("Device %v: %v (USB %v): %v (%04x) from %v (%04x), Release %v",
						i, dev.Path, dev.Interface, dev.Product, dev.ProductID, dev.Manufacturer, dev.VendorID, dev.Release)
				}
				return nil, errors.New("Please specify a unique device path")
			}
		}
		log.Printf("Opening USB HID device %v (USB %v): %v (%04x) from %v (%04x), Release %v",
			info.Path, info.Interface, info.Product, info.ProductID, info.Manufacturer, info.VendorID, info.Release)
		dev, err := info.Open()
		if err != nil {
			return nil, err
		}
		return &Ft260{
			Device: dev,
		}, nil
	}
}

func Open() (*Ft260, error) {
	return (&Ft260Driver{}).Open()
}

func OpenPath(path string) (*Ft260, error) {
	return (&Ft260Driver{}).OpenPath(path)
}

type Ft260 struct {
	*hid.Device
}

type ReportIn interface {
	Unmarshall(data []byte) error
	ReportID() byte
	ReportLen() int
}

type ReportOut interface {
	Marshall(data []byte) error
	ReportID() byte
	ReportLen() int
}

// Marker interface to switch from Feature in/out requests to raw data
type DataReport interface {
	IsDataReport() bool
}

// Marker interface for ReportIn implementations with variable size. The result of ReportLen() is treated as max size.
type VariableSizeReport interface {
	IsVariableSize() bool
}

// Marker interface for ReportIn implementations with variable report ID.
type VariableReportID interface {
	IsVariableReportID() bool
}

func (f *Ft260) Write(input interface{}) error {
	var data []byte
	feature := true
	switch v := input.(type) {
	case []byte:
		data = v
	case ReportOut:
		data = make([]byte, v.ReportLen()+1)
		err := v.Marshall(data[1:])
		if err != nil {
			return fmt.Errorf("Failed to marshall Ft260 write operation %T: %v", input, err)
		}
		data[0] = v.ReportID()
		if dataReport, ok := v.(DataReport); ok {
			feature = !dataReport.IsDataReport()
		}
	default:
		return fmt.Errorf("Unexpected type for writing to FT260: %T", input)
	}
	log.Debugf("Writing HID report (feature: %v). Data: %#v", feature, data)
	n, err := f.Device.DoWrite(data, feature)
	if err == nil && n != len(data) {
		err = fmt.Errorf("ft260: wrong write len (%v instead of %v)", n, len(data))
	}
	return err
}

func (f *Ft260) Read(report ReportIn) error {
	data := make([]byte, report.ReportLen()+1)
	data[0] = report.ReportID()

	feature := true
	if dataReport, ok := report.(DataReport); ok {
		feature = !dataReport.IsDataReport()
	}
	log.Debugf("Reading HID report (feature: %v). Data: %#v", feature, data)
	n, err := f.Device.DoRead(data, feature, ReadReportTimeout)
	if variableReport, ok := report.(VariableSizeReport); !ok || !variableReport.IsVariableSize() {
		if err == nil && n != len(data) {
			err = fmt.Errorf("ft260: wrong read len (%v instead of %v)", n, len(data))
		}
	}
	if variableReport, ok := report.(VariableReportID); !ok || !variableReport.IsVariableReportID() {
		if err == nil && data[0] != report.ReportID() {
			return fmt.Errorf("Unexpected report id (expected %v, received %v)", report.ReportID(), data[0])
		}
	}
	if err == nil {
		err = report.Unmarshall(data[1:n])
		if err != nil {
			err = fmt.Errorf("Failed to unmarshall ft260 read result %T: %v", report, err)
		}
	}
	return err
}

func _readBool(b []byte, index int, e *error) bool {
	if *e == nil {
		val := b[index]
		if val == 0 {
			return false
		} else if val == 1 {
			return true
		} else {
			*e = fmt.Errorf("Expected 0 or 1 for byte at index %v, but got %02x", index, val)
		}
	}
	return false
}
