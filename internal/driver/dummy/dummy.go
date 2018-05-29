package dummy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/CCI-MOC/obmd/internal/driver"
)

var Driver driver.Driver = dummyDriver{}

// A "dummy" Driver, that rather than actually talking to an OBM,
// Connects to the address it is passed via tcp, sends the info to
// the destination address, and then returns that connection.
// This is useful for experimentation.
type dummyDriver struct {
}

func (dummyDriver) GetOBM(info []byte) (driver.OBM, error) {
	ret := dummyOBM{}
	err := json.Unmarshal(info, &ret)
	ret.PwrStatus = "off"
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

type dummyOBM struct {
	Addr      string   `json:"addr"`
	conn      net.Conn `json:"-"`
	PwrStatus string
}

func (d *dummyOBM) Serve(ctx context.Context) {
	<-ctx.Done()
	d.DropConsole()
}

func (d *dummyOBM) DropConsole() error {
	if d.conn != nil {
		return d.conn.Close()
		d.conn = nil
	}
	return nil
}

func (d *dummyOBM) DialConsole() (io.ReadCloser, error) {
	conn, err := net.Dial("tcp", d.Addr)
	if err != nil {
		return nil, err
	}
	_, err = fmt.Fprintln(conn, d)
	if err != nil {
		conn.Close()
		return nil, err
	}
	d.conn = conn
	return conn, nil
}

func (d *dummyOBM) PowerOn() error {
	log.Println("Powering on:", d)
	d.PwrStatus = "on"
	return nil
}

func (d *dummyOBM) PowerOff() error {
	log.Println("Powering off:", d)
	d.PwrStatus = "off"
	return nil
}

func (d *dummyOBM) PowerCycle(force bool) error {
	log.Printf("Power cycling: %v (force = %v)\n", d, force)
	d.PwrStatus = "on"
	return nil
}

func (d *dummyOBM) SetBootdev(dev string) error {
	log.Printf("Setting bootdev = %v: %v\n", dev, d)
	return nil
}

func (d *dummyOBM) GetPowerStatus() (string, error) {
	log.Printf("Status = %v: %v\n", d.PwrStatus, d)
	return d.PwrStatus, nil
}
