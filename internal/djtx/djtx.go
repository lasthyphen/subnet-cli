// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package avax

import (
	"fmt"

	"github.com/lasthyphen/dijetsnodego/codec"
	"github.com/lasthyphen/dijetsnodego/vms/components/avax"
)

func ParseUTXO(ub []byte, cd codec.Manager) (*avax.UTXO, error) {
	utxo := new(avax.UTXO)
	if _, err := cd.Unmarshal(ub, utxo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal utxo bytes: %w", err)
	}
	return utxo, nil
}
