// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package key

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"strings"

	"github.com/lasthyphen/subnet-cli/internal/codec"

	"github.com/lasthyphen/dijetsnodego/ids"
	"github.com/lasthyphen/dijetsnodego/utils/crypto"
	"github.com/lasthyphen/dijetsnodego/utils/formatting"
	"github.com/lasthyphen/dijetsnodego/vms/components/djtx"
	"github.com/lasthyphen/dijetsnodego/vms/platformvm"
	"github.com/lasthyphen/dijetsnodego/vms/secp256k1fx"
	"go.uber.org/zap"
)

var (
	ErrInvalidPrivateKey         = errors.New("invalid private key")
	ErrInvalidPrivateKeyLen      = errors.New("invalid private key length (expect 64 bytes in hex)")
	ErrInvalidPrivateKeyEnding   = errors.New("invalid private key ending")
	ErrInvalidPrivateKeyEncoding = errors.New("invalid private key encoding")
)

var _ Key = &SoftKey{}

type SoftKey struct {
	privKey        *crypto.PrivateKeySECP256K1R
	privKeyRaw     []byte
	privKeyEncoded string

	pAddr string

	keyChain *secp256k1fx.Keychain
}

const (
	privKeyEncPfx = "PrivateKey-"
	privKeySize   = 64

	rawEwoqPk      = "ewoqjP7PxY4yr3iLTpLisriqt94hdyDFNgchSxGGztUrTXtNN"
	EwoqPrivateKey = "PrivateKey-" + rawEwoqPk
)

var keyFactory = new(crypto.FactorySECP256K1R)

type SOp struct {
	privKey        *crypto.PrivateKeySECP256K1R
	privKeyEncoded string
}

type SOpOption func(*SOp)

func (sop *SOp) applyOpts(opts []SOpOption) {
	for _, opt := range opts {
		opt(sop)
	}
}

// To create a new key SoftKey with a pre-loaded private key.
func WithPrivateKey(privKey *crypto.PrivateKeySECP256K1R) SOpOption {
	return func(sop *SOp) {
		sop.privKey = privKey
	}
}

// To create a new key SoftKey with a pre-defined private key.
func WithPrivateKeyEncoded(privKey string) SOpOption {
	return func(sop *SOp) {
		sop.privKeyEncoded = privKey
	}
}

func NewSoft(networkID uint32, opts ...SOpOption) (*SoftKey, error) {
	ret := &SOp{}
	ret.applyOpts(opts)

	// set via "WithPrivateKeyEncoded"
	if len(ret.privKeyEncoded) > 0 {
		privKey, err := decodePrivateKey(ret.privKeyEncoded)
		if err != nil {
			return nil, err
		}
		// to not overwrite
		if ret.privKey != nil &&
			!bytes.Equal(ret.privKey.Bytes(), privKey.Bytes()) {
			return nil, ErrInvalidPrivateKey
		}
		ret.privKey = privKey
	}

	// generate a new one
	if ret.privKey == nil {
		rpk, err := keyFactory.NewPrivateKey()
		if err != nil {
			return nil, err
		}
		var ok bool
		ret.privKey, ok = rpk.(*crypto.PrivateKeySECP256K1R)
		if !ok {
			return nil, ErrInvalidType
		}
	}

	privKey := ret.privKey
	privKeyEncoded, err := encodePrivateKey(ret.privKey)
	if err != nil {
		return nil, err
	}

	// double-check encoding is consistent
	if ret.privKeyEncoded != "" &&
		ret.privKeyEncoded != privKeyEncoded {
		return nil, ErrInvalidPrivateKeyEncoding
	}

	keyChain := secp256k1fx.NewKeychain()
	keyChain.Add(privKey)

	m := &SoftKey{
		privKey:        privKey,
		privKeyRaw:     privKey.Bytes(),
		privKeyEncoded: privKeyEncoded,

		keyChain: keyChain,
	}

	// Parse HRP to create valid address
	hrp := getHRP(networkID)
	m.pAddr, err = formatting.FormatAddress("P", hrp, m.privKey.PublicKey().Address().Bytes())
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Returns the private key.
func (m *SoftKey) Key() *crypto.PrivateKeySECP256K1R {
	return m.privKey
}

// Returns the private key in raw bytes.
func (m *SoftKey) Raw() []byte {
	return m.privKeyRaw
}

// Returns the private key encoded in CB58 and "PrivateKey-" prefix.
func (m *SoftKey) Encode() string {
	return m.privKeyEncoded
}

// Saves the private key to disk with hex encoding.
func (m *SoftKey) Save(p string) error {
	k := hex.EncodeToString(m.privKeyRaw)
	return ioutil.WriteFile(p, []byte(k), fsModeWrite)
}

func (m *SoftKey) P() string { return m.pAddr }

func (m *SoftKey) Spends(outputs []*djtx.UTXO, opts ...OpOption) (
	totalBalanceToSpend uint64,
	inputs []*djtx.TransferableInput,
) {
	ret := &Op{}
	ret.applyOpts(opts)

	for _, out := range outputs {
		input, err := m.spend(out, ret.time)
		if err != nil {
			zap.L().Warn("cannot spend with current key", zap.Error(err))
			continue
		}
		totalBalanceToSpend += input.Amount()
		inputs = append(inputs, &djtx.TransferableInput{
			UTXOID: out.UTXOID,
			Asset:  out.Asset,
			In:     input,
		})
		if ret.targetAmount > 0 &&
			totalBalanceToSpend > ret.targetAmount+ret.feeDeduct {
			break
		}
	}
	djtx.SortTransferableInputs(inputs)

	return totalBalanceToSpend, inputs
}

func (m *SoftKey) spend(output *djtx.UTXO, time uint64) (
	input djtx.TransferableIn,
	err error,
) {
	// "time" is used to check whether the key owner
	// is still within the lock time (thus can't spend).
	inputf, _, err := m.keyChain.Spend(output.Out, time)
	if err != nil {
		return nil, err
	}
	var ok bool
	input, ok = inputf.(djtx.TransferableIn)
	if !ok {
		return nil, ErrInvalidType
	}
	return input, nil
}

const fsModeWrite = 0o600

// LoadSoft loads the private key from disk and creates the corresponding SoftKey.
func LoadSoft(networkID uint32, keyPath string) (*SoftKey, error) {
	kb, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	// in case, it's already encoded
	k, err := NewSoft(networkID, WithPrivateKeyEncoded(string(kb)))
	if err == nil {
		return k, nil
	}

	r := bufio.NewReader(bytes.NewBuffer(kb))
	buf := make([]byte, privKeySize)
	n, err := readASCII(buf, r)
	if err != nil {
		return nil, err
	}
	if n != len(buf) {
		return nil, ErrInvalidPrivateKeyLen
	}
	if err := checkKeyFileEnd(r); err != nil {
		return nil, err
	}

	skBytes, err := hex.DecodeString(string(buf))
	if err != nil {
		return nil, err
	}
	rpk, err := keyFactory.ToPrivateKey(skBytes)
	if err != nil {
		return nil, err
	}
	privKey, ok := rpk.(*crypto.PrivateKeySECP256K1R)
	if !ok {
		return nil, ErrInvalidType
	}

	return NewSoft(networkID, WithPrivateKey(privKey))
}

// readASCII reads into 'buf', stopping when the buffer is full or
// when a non-printable control character is encountered.
func readASCII(buf []byte, r io.ByteReader) (n int, err error) {
	for ; n < len(buf); n++ {
		buf[n], err = r.ReadByte()
		switch {
		case errors.Is(err, io.EOF) || buf[n] < '!':
			return n, nil
		case err != nil:
			return n, err
		}
	}
	return n, nil
}

const fileEndLimit = 1

// checkKeyFileEnd skips over additional newlines at the end of a key file.
func checkKeyFileEnd(r io.ByteReader) error {
	for idx := 0; ; idx++ {
		b, err := r.ReadByte()
		switch {
		case errors.Is(err, io.EOF):
			return nil
		case err != nil:
			return err
		case b != '\n' && b != '\r':
			return ErrInvalidPrivateKeyEnding
		case idx > fileEndLimit:
			return ErrInvalidPrivateKeyLen
		}
	}
}

func encodePrivateKey(pk *crypto.PrivateKeySECP256K1R) (string, error) {
	privKeyRaw := pk.Bytes()
	enc, err := formatting.EncodeWithChecksum(formatting.CB58, privKeyRaw)
	if err != nil {
		return "", err
	}
	return privKeyEncPfx + enc, nil
}

func decodePrivateKey(enc string) (*crypto.PrivateKeySECP256K1R, error) {
	rawPk := strings.Replace(enc, privKeyEncPfx, "", 1)
	skBytes, err := formatting.Decode(formatting.CB58, rawPk)
	if err != nil {
		return nil, err
	}
	rpk, err := keyFactory.ToPrivateKey(skBytes)
	if err != nil {
		return nil, err
	}
	privKey, ok := rpk.(*crypto.PrivateKeySECP256K1R)
	if !ok {
		return nil, ErrInvalidType
	}
	return privKey, nil
}

func (m *SoftKey) Address() ids.ShortID {
	return m.privKey.PublicKey().Address()
}

func (m *SoftKey) Sign(pTx *platformvm.Tx, sigs int) error {
	signers := make([][]*crypto.PrivateKeySECP256K1R, sigs)
	for i := 0; i < sigs; i++ {
		signers[i] = []*crypto.PrivateKeySECP256K1R{m.privKey}
	}

	return pTx.Sign(codec.PCodecManager, signers)
}
