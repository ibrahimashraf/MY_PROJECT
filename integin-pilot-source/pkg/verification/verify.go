package verification

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Envelope is minimal signed payload.
type Envelope struct {
	Payload   json.RawMessage `json:"payload"`
	Signature string          `json:"signature"`
	DID       string          `json:"did"`
}

// ParseDIDKey parses did:key:z... (base58-btc multicodec ed25519-pub).
func ParseDIDKey(did string) (ed25519.PublicKey, error) {
	const prefix = "did:key:"
	if !strings.HasPrefix(did, prefix) {
		return nil, errors.New("unsupported DID method")
	}
	rest := strings.TrimPrefix(did, prefix)
	if !strings.HasPrefix(rest, "z") {
		return nil, errors.New("invalid did:key encoding")
	}
	raw := base58Decode(rest[1:])
	// multicodec 0xed 0x01 + 32-byte key
	if len(raw) != 34 || raw[0] != 0xed || raw[1] != 0x01 {
		return nil, fmt.Errorf("invalid ed25519 did:key length %d", len(raw))
	}
	return ed25519.PublicKey(raw[2:]), nil
}

var b58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

func base58Decode(s string) []byte {
	var num []byte // big-endian base256
	num = []byte{0}
	for _, c := range s {
		idx := strings.IndexRune(b58Alphabet, c)
		if idx < 0 {
			return nil
		}
		carry := idx
		for i := len(num) - 1; i >= 0; i-- {
			v := int(num[i])*58 + carry
			num[i] = byte(v & 0xff)
			carry = v >> 8
		}
		for carry > 0 {
			num = append([]byte{byte(carry & 0xff)}, num...)
			carry >>= 8
		}
	}
	// leading zeros
	nLead := 0
	for _, c := range s {
		if c == '1' {
			nLead++
		} else {
			break
		}
	}
	// strip leading zeros in num
	i := 0
	for i < len(num)-1 && num[i] == 0 {
		i++
	}
	num = num[i:]
	out := make([]byte, nLead+len(num))
	copy(out[nLead:], num)
	// if num was just zero, avoid duplicate
	if len(num) == 1 && num[0] == 0 {
		out = out[:nLead+1]
		// trim to nLead+1 already
		if nLead == 0 {
			out = []byte{0}
		}
	}
	return out
}

func base58Encode(b []byte) string {
	zeros := 0
	for zeros < len(b) && b[zeros] == 0 {
		zeros++
	}
	var num []byte
	num = append([]byte{}, b[zeros:]...)
	if len(num) == 0 {
		num = []byte{0}
	}
	var out []byte
	// convert base256 -> base58
	// repeated divmod 58
	buf := append([]byte{}, b[zeros:]...)
	if len(buf) == 0 {
		return strings.Repeat("1", zeros)
	}
	var digits []byte
	for len(buf) > 0 {
		var next []byte
		rem := 0
		for _, v := range buf {
			cur := rem*256 + int(v)
			q := cur / 58
			rem = cur % 58
			if len(next) > 0 || q > 0 {
				next = append(next, byte(q))
			}
		}
		digits = append(digits, b58Alphabet[rem])
		buf = next
	}
	for i := 0; i < zeros; i++ {
		digits = append(digits, '1')
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	_ = num
	_ = out
	return string(digits)
}

// DIDKeyFromPub builds did:key from ed25519 pubkey (stdlib-only).
func DIDKeyFromPub(pub ed25519.PublicKey) string {
	raw := append([]byte{0xed, 0x01}, pub...)
	return "did:key:z" + base58Encode(raw)
}

// VerifyEnvelope verifies env.Signature over env.Payload using DID.
func VerifyEnvelope(env Envelope) error {
	pub, err := ParseDIDKey(env.DID)
	if err != nil {
		return err
	}
	sig, err := base64.RawURLEncoding.DecodeString(env.Signature)
	if err != nil {
		// try std encoding
		sig2, err2 := base64.StdEncoding.DecodeString(env.Signature)
		if err2 != nil {
			return errors.New("invalid signature encoding")
		}
		sig = sig2
	}
	if len(sig) != ed25519.SignatureSize {
		return errors.New("invalid signature size")
	}
	if !ed25519.Verify(pub, []byte(env.Payload), sig) {
		return errors.New("invalid signature")
	}
	return nil
}
