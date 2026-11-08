package shamcash

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

// protectedHeader the exact header the official SDK writes (DirectJwe.php).
const protectedHeader = `{"alg":"dir","enc":"A256GCM","cty":"json"}`

// ErrCrypto any token that cannot be opened or is not the expected shape.
var ErrCrypto = errors.New("shamcash: invalid encrypted token")

// jwe Direct JWE compact serialization with A256GCM, mirroring the SDK:
// header.<empty key>.iv(12).ciphertext.tag(16), AAD = the ASCII header segment.
type jwe struct {
	aead cipher.AEAD
}

func newJWE(secretB64 string) (*jwe, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(secretB64))
	if err != nil || len(key) != 32 {
		return nil, errors.New("shamcash: secret key must be a Base64-encoded 32-byte key")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &jwe{aead: aead}, nil
}

var b64 = base64.RawURLEncoding

func (j *jwe) encrypt(plaintext []byte) (string, error) {
	iv := make([]byte, 12)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}
	hdr := b64.EncodeToString([]byte(protectedHeader))
	sealed := j.aead.Seal(nil, iv, plaintext, []byte(hdr))
	ct, tag := sealed[:len(sealed)-16], sealed[len(sealed)-16:]
	return hdr + ".." + b64.EncodeToString(iv) + "." + b64.EncodeToString(ct) + "." + b64.EncodeToString(tag), nil
}

func (j *jwe) decrypt(token string) ([]byte, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 5 || parts[1] != "" {
		return nil, ErrCrypto
	}
	rawHdr, err := b64.DecodeString(parts[0])
	if err != nil {
		return nil, ErrCrypto
	}
	var h map[string]any
	if json.Unmarshal(rawHdr, &h) != nil || len(h) != 3 ||
		h["alg"] != "dir" || h["enc"] != "A256GCM" || h["cty"] != "json" {
		return nil, ErrCrypto
	}
	iv, err1 := b64.DecodeString(parts[2])
	ct, err2 := b64.DecodeString(parts[3])
	tag, err3 := b64.DecodeString(parts[4])
	if err1 != nil || err2 != nil || err3 != nil || len(iv) != 12 || len(tag) != 16 || len(ct) == 0 {
		return nil, ErrCrypto
	}
	pt, err := j.aead.Open(nil, iv, append(ct, tag...), []byte(parts[0]))
	if err != nil {
		return nil, ErrCrypto
	}
	return pt, nil
}
