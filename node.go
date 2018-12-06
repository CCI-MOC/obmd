package main

import (
	"context"

	"github.com/CCI-MOC/obmd/internal/driver"
	"github.com/CCI-MOC/obmd/token"
)

// Information about a node
type Node struct {
	ConnInfo     []byte             // Connection info for this node's OBM.
	ObmCancel    context.CancelFunc // stop the OBM
	OBM          driver.OBM         // OBM for this node.
	CurrentToken token.Token        // Token for regular user operations.
}

// Returns a new node with the given driver information. The token will be
// freshly generated.
func NewNode(d driver.Driver, info []byte) (*Node, error) {
	tok, err := token.New()
	if err != nil {
		return nil, err
	}

	obm, err := d.GetOBM(info)
	if err != nil {
		return nil, err
	}
	ret := &Node{
		OBM:          obm,
		ConnInfo:     info,
		CurrentToken: tok,
	}
	return ret, nil
}

// Generate a new token, invaidating the old one if any, and disconnecting
// clients using it. If an error occurs, the state of the node/token will
// be unchanged.
func (n *Node) NewToken() (token.Token, error) {
	if err := n.ClearToken(); err != nil {
		return token.Token{}, err
	}
	tok, err := token.New()
	if err != nil {
		return tok, err
	}
	n.CurrentToken = tok
	return n.CurrentToken, nil
}

// Return whether a token is valid.
func (n *Node) ValidToken(tok token.Token) bool {
	return n.CurrentToken.Verify(tok) == nil
}

// Clear any existing token, and disconnect any clients
func (n *Node) ClearToken() error {
	var err error
	n.OBM.DropConsole()
	// We just overwrite the token with a new one, which we never actually
	// share with the outside world. So technically there's still a valid
	// token, but nobody knows it, so the node is effectively inaccessible.
	n.CurrentToken, err = token.New()
	return err
}

func (n *Node) StartOBM() {
	if n.ObmCancel != nil {
		panic("BUG: OBM is already started!")
	}
	ctx, cancel := context.WithCancel(context.Background())
	n.ObmCancel = cancel
	go n.OBM.Serve(ctx)
}

func (n *Node) StopOBM() {
	if n.ObmCancel == nil {
		panic("BUG: OBM is not running!")
	}
	n.ObmCancel()
	n.ObmCancel = nil
}
