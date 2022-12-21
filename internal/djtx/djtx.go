// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package djtx

import (
	"fmt"

	"github.com/lasthyphen/dijetsnodego/codec"
	"github.com/lasthyphen/dijetsnodego/vms/components/djtx"
)

func ParseUTXO(ub []byte, cd codec.Manager) (*djtx.UTXO, error) {
	utxo := new(djtx.UTXO)
	if _, err := cd.Unmarshal(ub, utxo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal utxo bytes: %w", err)
	}
	return utxo, nil
}
