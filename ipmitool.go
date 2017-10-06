package main

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/kr/pty"
)

var (
	ErrInvalidBootdev = errors.New("Invalid boot device.")
)

type IpmitoolDialer struct {
}

// A running ipmi process, connected to a serial console. Its Close() method
// kills the process as well as closing its attached pty.
type ipmiProcess struct {
	io.ReadCloser
	proc *os.Process
}

func (p *ipmiProcess) Close() error {
	p.proc.Signal(syscall.SIGTERM)
	p.proc.Wait()
	return p.ReadCloser.Close()
}

func (d *IpmitoolDialer) DialIpmi(info *IpmiInfo) (io.ReadCloser, error) {
	cmd := d.callIpmitool(info, "sol", "activate")
	stdio, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}
	return &ipmiProcess{
		ReadCloser: stdio,
		proc:       cmd.Process,
	}, nil
}

func (d *IpmitoolDialer) callIpmitool(info *IpmiInfo, args ...string) *exec.Cmd {
	// Annoyingly, when invoking a variadic function f(x ...Foo), you can't
	// just do Foo(x, y, z, ...more); you need either Foo(x, y, z) or
	// Foo(...more). We work around this by adding the static arguments to
	// the slice, and then doing the latter:
	args = append([]string{
		"-I", "lanplus",
		"-U", info.User,
		"-P", info.Pass,
		"-H", info.Addr,
	}, args...)
	return exec.Command("ipmitool", args...)
}

func (d *IpmitoolDialer) PowerOff(info *IpmiInfo) error {
	return d.callIpmitool(info, "chassis", "power", "off").Run()
}

func (d *IpmitoolDialer) PowerCycle(info *IpmiInfo, force bool) error {
	var op string
	if force {
		op = "reset"
	} else {
		op = "cycle"
	}

	err := d.callIpmitool(info, "chassis", "power", op).Run()
	if err == nil {
		return nil
	}
	// The above can fail if the machine is already powered off; in
	// this case we just turn it on:
	return d.callIpmitool(info, "chassis", "power", "on").Run()
}

func (d *IpmitoolDialer) SetBootdev(info *IpmiInfo, dev string) error {
	if dev != "disk" && dev != "pxe" && dev != "none" {
		return ErrInvalidBootdev
	}
	return d.callIpmitool(info,
		"chassis", "bootdev", dev, "options=persistent").Run()
}
