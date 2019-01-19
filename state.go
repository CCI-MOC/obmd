package main

import (
	"database/sql"

	"github.com/CCI-MOC/obmd/internal/driver"
)

// Persistent store for node info, + ephemeral tracking of live OBM
// connections.
//
// This is basically a map[string]*Node, except that it (a) persists changes in
// metadata to a database, and (b) will shutdown/initialize OBMs as needed
//
// Note that this is not thread-safe.
type State struct {
	db     *sql.DB
	nodes  map[string]*Node
	driver driver.Driver
}

// Create a State from a database. This loads existent objects in immediately.
func NewState(db *sql.DB, driver driver.Driver) (*State, error) {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS nodes (
		label VARCHAR(80) PRIMARY KEY,
		obm_info TEXT NOT NULL
	)`)
	if err != nil {
		return nil, err
	}
	ret := &State{
		nodes:  make(map[string]*Node),
		db:     db,
		driver: driver,
	}
	rows, err := db.Query(`SELECT label, obm_info FROM nodes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			label string
			info  []byte
		)
		err = rows.Scan(&label, &info)
		if err != nil {
			return nil, err
		}
		node, err := NewNode(driver, info)
		if err != nil {
			return nil, err
		}
		ret.nodes[label] = node
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	for _, node := range ret.nodes {
		node.StartOBM()
	}
	ret.check()
	return ret, nil
}

func (s *State) check() {
	for label, node := range s.nodes {
		if node == nil {
			panic("Node " + label + " is nil!")
		}
	}
}

// Clean up resources used by the State. Does not close the database.
func (s *State) Close() error {
	for _, node := range s.nodes {
		node.StopOBM()
	}
	return nil
}

func (s *State) GetNode(label string) (*Node, error) {
	node, ok := s.nodes[label]
	if !ok {
		return nil, ErrNoSuchNode
	}
	return node, nil
}

func (s *State) NewNode(label string, info []byte) (*Node, error) {
	_, err := s.GetNode(label)
	if err == nil {
		return nil, ErrNodeExists
	}
	// Node doesn't exist; create it.
	node, err := NewNode(s.driver, info)
	if err != nil {
		return nil, err
	}
	_, err = s.db.Exec(
		`INSERT INTO nodes(label, obm_info)
			VALUES ($1, $2)`,
		label,
		info,
	)
	if err != nil {
		return nil, err
	}
	s.nodes[label] = node
	node.StartOBM()
	return node, nil
}

func (s *State) DeleteNode(label string) error {
	var err error
	node, ok := s.nodes[label]
	if ok {
		node.StopOBM()
		delete(s.nodes, label)
		_, err = s.db.Exec("DELETE FROM nodes WHERE label = $1", label)
	}
	return err
}
