package tank

import (
	"sync"

	"github.com/antongulenko/tank/ft260"
	log "github.com/sirupsen/logrus"
)

const (
	I2cWrite = iota + 1
	I2cRead
	I2cWriteRead
	I2cGet
)

type I2cRequest struct {
	Type        int
	Addr        byte
	DataWrite   []byte
	DataRead    []byte
	GetRegister byte // Only for I2cGet
	GetSize     int  // Only for I2cGet
	Error       error

	done bool
	wait *sync.Cond
}

func (r *I2cRequest) init() {
	r.wait = &sync.Cond{L: new(sync.Mutex)}
}

func (r *I2cRequest) Wait() {
	r.wait.L.Lock()
	defer r.wait.L.Unlock()
	for !r.done {
		r.wait.Wait()
	}
}

func (r *I2cRequest) notifyDone() {
	r.wait.L.Lock()
	defer r.wait.L.Unlock()
	r.done = true
	r.wait.Broadcast()
}

type sequencedI2cBus struct {
	usb      *ft260.Ft260
	i2cQueue chan *I2cRequest
}

func (t *sequencedI2cBus) handleI2cRequests() {
	for req := range t.i2cQueue {
		switch req.Type {
		case I2cWrite:
			req.Error = t.usb.I2cWrite(req.Addr, req.DataWrite...)
			req.notifyDone()
		case I2cRead:
			req.Error = t.usb.I2cRead(req.Addr, req.DataRead)
			req.notifyDone()
		case I2cWriteRead:
			req.Error = t.usb.I2cWriteRead(req.Addr, req.DataWrite, req.DataRead)
			req.notifyDone()
		case I2cGet:
			req.DataRead, req.Error = t.usb.I2cGet(req.Addr, req.GetRegister, req.GetSize)
			req.notifyDone()
		default:
			log.Errorln("Ignoring invalid tank I2c request with type", req.Type)
		}
	}
}

func (t *sequencedI2cBus) QueueI2cRequest(req *I2cRequest) {
	req.init()
	t.i2cQueue <- req
}

func (t *sequencedI2cBus) I2cRequest(req *I2cRequest) {
	t.QueueI2cRequest(req)
	req.Wait()
}

func (t *sequencedI2cBus) I2cWrite(addr byte, data ...byte) error {
	req := &I2cRequest{
		Addr:      addr,
		Type:      I2cWrite,
		DataWrite: data,
	}
	t.I2cRequest(req)
	return req.Error
}

func (t *sequencedI2cBus) I2cRead(addr byte, data []byte) error {
	req := &I2cRequest{
		Addr:     addr,
		Type:     I2cRead,
		DataRead: data,
	}
	t.I2cRequest(req)
	return req.Error
}

func (t *sequencedI2cBus) I2cWriteRead(addr byte, out, in []byte) error {
	req := &I2cRequest{
		Addr:      addr,
		Type:      I2cWriteRead,
		DataRead:  in,
		DataWrite: out,
	}
	t.I2cRequest(req)
	return req.Error
}

func (t *sequencedI2cBus) I2cGet(addr byte, registerAddr byte, size int) ([]byte, error) {
	req := &I2cRequest{
		Addr:        addr,
		Type:        I2cGet,
		GetRegister: registerAddr,
		GetSize:     size,
	}
	t.I2cRequest(req)
	return req.DataRead, req.Error
}

type dummyI2cBus struct {
}

func (d *dummyI2cBus) I2cWrite(addr byte, data ...byte) error {
	return nil
}

func (d *dummyI2cBus) I2cRead(addr byte, data []byte) error {
	return nil
}

func (d *dummyI2cBus) I2cWriteRead(addr byte, out, in []byte) error {
	return nil
}

func (d *dummyI2cBus) I2cGet(addr byte, registerAddr byte, size int) ([]byte, error) {
	return nil, nil
}
