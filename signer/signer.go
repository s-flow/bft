package signer

import (
	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/ed25519"
)

type Signer interface {
	PubKey() crypto.PubKey
	Address() crypto.Address
	Sign([]byte) error
}

type MockSigner struct {
	privKey crypto.PrivKey
}

func NewMockSigner() MockSigner {
	privKey := ed25519.GenPrivKey()
	return MockSigner{privKey: privKey}
}

func (s MockSigner) PubKey() crypto.PubKey {
	return s.privKey.PubKey()
}

func (s MockSigner) Address() crypto.Address {
	return s.privKey.PubKey().Address()
}

func (s MockSigner) Sign(data []byte) ([]byte, error) {
	sig, err := s.privKey.Sign(data)
	if err != nil {
		// @TODO: error logging
		return nil, err
	}

	return sig, nil
}
