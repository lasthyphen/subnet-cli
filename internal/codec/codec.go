// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package codec

import (
	"github.com/lasthyphen/dijetsnodego/codec"
	"github.com/lasthyphen/dijetsnodego/codec/linearcodec"
	"github.com/lasthyphen/dijetsnodego/utils/wrappers"
	"github.com/lasthyphen/dijetsnodego/vms/platformvm"
	"github.com/lasthyphen/dijetsnodego/vms/secp256k1fx"
)

var PCodecManager codec.Manager

func init() {
	pc := linearcodec.NewDefault()
	PCodecManager = codec.NewDefaultManager()
	errs := wrappers.Errs{}
	errs.Add(
		pc.RegisterType(&platformvm.ProposalBlock{}),
		pc.RegisterType(&platformvm.AbortBlock{}),
		pc.RegisterType(&platformvm.CommitBlock{}),
		pc.RegisterType(&platformvm.StandardBlock{}),
		pc.RegisterType(&platformvm.AtomicBlock{}),
		pc.RegisterType(&secp256k1fx.TransferInput{}),
		pc.RegisterType(&secp256k1fx.MintOutput{}),
		pc.RegisterType(&secp256k1fx.TransferOutput{}),
		pc.RegisterType(&secp256k1fx.MintOperation{}),
		pc.RegisterType(&secp256k1fx.Credential{}),
		pc.RegisterType(&secp256k1fx.Input{}),
		pc.RegisterType(&secp256k1fx.OutputOwners{}),
		pc.RegisterType(&platformvm.UnsignedAddValidatorTx{}),
		pc.RegisterType(&platformvm.UnsignedAddSubnetValidatorTx{}),
		pc.RegisterType(&platformvm.UnsignedAddDelegatorTx{}),
		pc.RegisterType(&platformvm.UnsignedCreateChainTx{}),
		pc.RegisterType(&platformvm.UnsignedCreateSubnetTx{}),
		pc.RegisterType(&platformvm.UnsignedImportTx{}),
		pc.RegisterType(&platformvm.UnsignedExportTx{}),
		pc.RegisterType(&platformvm.UnsignedAdvanceTimeTx{}),
		pc.RegisterType(&platformvm.UnsignedRewardValidatorTx{}),
		pc.RegisterType(&platformvm.StakeableLockIn{}),
		pc.RegisterType(&platformvm.StakeableLockOut{}),
		PCodecManager.RegisterCodec(0, pc),
	)
	if errs.Errored() {
		panic(errs.Err)
	}
}
