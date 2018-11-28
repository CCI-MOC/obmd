package main

import (
	"errors"
	"io"
	"sync"

	"github.com/CCI-MOC/obmd/token"
)

var (
	ErrNodeExists = errors.New("Node already exists.")
	ErrNoSuchNode = errors.New("No such node.")
)

type Daemon struct {
	sync.Mutex
	state *State
	funcs chan func()
}

func NewDaemon(state *State) *Daemon {
	return &Daemon{
		state: state,
	}
}

func (d *Daemon) DeleteNode(label string) error {
	d.Lock()
	defer d.Unlock()
	return d.state.DeleteNode(label)
}

func (d *Daemon) SetNode(label string, info []byte) error {
	d.Lock()
	defer d.Unlock()

	d.state.check()

	_, err := d.state.GetNode(label)
	if err == nil {
		return ErrNodeExists
	}
	// Create the node.
	_, err = d.state.NewNode(label, info)

	d.state.check()
	return err
}

func (d *Daemon) GetNodeToken(label string) (token.Token, error) {
	d.Lock()
	defer d.Unlock()
	node, err := d.state.GetNode(label)
	if err != nil {
		return token.Token{}, err
	}
	tok, err := node.NewToken()
	if err != nil {
		return token.Token{}, err
	}
	return tok, nil
}

func (d *Daemon) InvalidateNodeToken(label string) error {
	d.Lock()
	defer d.Unlock()
	node, err := d.state.GetNode(label)
	if err != nil {
		return err
	}
	return node.ClearToken()
}

// Get the node with the specified label, and check that `token` is valid for it.
// Returns an error if the node does not exist or token is invalid.
func (d *Daemon) getNodeWithToken(label string, tok *token.Token) (*Node, error) {
	node, err := d.state.GetNode(label)
	if err != nil {
		return nil, err
	}
	if !node.ValidToken(*tok) {
		return nil, token.ErrInvalidToken
	}
	return node, nil
}

func (d *Daemon) usingNodeWithToken(label string, token *token.Token,
	f func(*Node) error) error {
	d.Lock()
	defer d.Unlock()
	node, err := d.getNodeWithToken(label, token)
	if err != nil {
		return err
	}
	return f(node)
}

func (d *Daemon) DialNodeConsole(label string, token *token.Token) (io.ReadCloser, error) {
	d.Lock()
	defer d.Unlock()
	node, err := d.getNodeWithToken(label, token)
	if err != nil {
		return nil, err
	}
	return node.OBM.DialConsole()
}

func (d *Daemon) PowerOnNode(label string, token *token.Token) error {
	return d.usingNodeWithToken(label, token, func(n *Node) error {
		return n.OBM.PowerOn()
	})
}

func (d *Daemon) PowerOffNode(label string, token *token.Token) error {
	return d.usingNodeWithToken(label, token, func(n *Node) error {
		return n.OBM.PowerOff()
	})
}

func (d *Daemon) PowerCycleNode(label string, force bool, token *token.Token) error {
	return d.usingNodeWithToken(label, token, func(n *Node) error {
		return n.OBM.PowerCycle(force)
	})
}

func (d *Daemon) SetNodeBootDev(label string, dev string, token *token.Token) error {
	d.Lock()
	defer d.Unlock()
	node, err := d.getNodeWithToken(label, token)
	if err != nil {
		return err
	}
	return node.OBM.SetBootdev(dev)
}

func (d *Daemon) GetNodePowerStatus(label string, token *token.Token) (string, error) {
	d.Lock()
	defer d.Unlock()
	node, err := d.getNodeWithToken(label, token)
	if err != nil {
		return "", err
	}
	return node.OBM.GetPowerStatus()
}
