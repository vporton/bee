// Copyright 2020 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package soc

import (
	"errors"

	"github.com/ethersphere/bee/pkg/swarm"
)

// Valid checks if the chunk is a valid single-owner chunk.
func Valid(ch swarm.Chunk) bool {
	s, err := FromChunk(ch)
	if err != nil {
		return false
	}

	address, err := s.address()
	if err != nil {
		return false
	}
	return ch.Address().Equal(address)
}

func ValidE(ch swarm.Chunk) error {
	s, err := FromChunk(ch)
	if err != nil {
		return err
	}

	address, err := s.address()
	if err != nil {
		return err
	}

	if !ch.Address().Equal(address) {
		return errors.New("address mismatch")
	}

	return nil
}
