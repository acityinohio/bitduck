package main

import (
	"encoding/hex"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/wire"
)

func signTX(priv, msg string) (sig string, err error) {
	// Decode a hex-encoded private key.
	pkBytes, err := hex.DecodeString(priv)
	if err != nil {
		return
	}
	privKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), pkBytes)
	// Sign a message using the private key.
	messageHash, err := hex.DecodeString(msg)
	signature, err := privKey.Sign(messageHash)
	if err != nil {
		return
	}
	sig = hex.EncodeToString(signature.Serialize())
	return
}

func signMsg(priv, msg string) (sig string, err error) {
	// Decode a hex-encoded private key.
	pkBytes, err := hex.DecodeString(priv)
	if err != nil {
		return
	}
	privKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), pkBytes)
	// Sign a message using the private key.
	messageHash := wire.DoubleSha256([]byte(msg))
	signature, err := privKey.Sign(messageHash)
	if err != nil {
		return
	}
	sig = hex.EncodeToString(signature.Serialize())
	return
}

func verifyMsg(pub, sig, msg string) (ver bool, err error) {
	// Decode a hex-encoded public key.
	pkBytes, err := hex.DecodeString(pub)
	sigBytes, err := hex.DecodeString(sig)
	if err != nil {
		return
	}
	pubKey, err := btcec.ParsePubKey(pkBytes, btcec.S256())
	signature, err := btcec.ParseDERSignature(sigBytes, btcec.S256())
	if err != nil {
		return
	}
	messageHash := wire.DoubleSha256([]byte(msg))
	ver = signature.Verify(messageHash, pubKey)
	return
}
