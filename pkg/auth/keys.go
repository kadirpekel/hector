package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// LoadOrGenerateRSAPrivateKey loads a PEM-encoded private key from disk or generates a new one.
// It uses standard crypto/rsa (which jwx builds upon) to ensure compatible PEM format.
func LoadOrGenerateRSAPrivateKey(path string) ([]byte, error) {
	// Try to read existing key
	if data, err := os.ReadFile(path); err == nil {
		slog.Debug("Loaded existing private key", "path", path)
		return data, nil
	}

	// Generate new RSA key
	slog.Info("Generating new RSA private key", "path", path)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Marshaling to PEM
	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	}
	pemData := pem.EncodeToMemory(pemBlock)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to file (restrictive permissions)
	if err := os.WriteFile(path, pemData, 0600); err != nil {
		return nil, fmt.Errorf("failed to write key file: %w", err)
	}

	return pemData, nil
}

// GenerateDeterministicEd25519KeyPEM generates an Ed25519 private key deterministically from a seed.
func GenerateDeterministicEd25519KeyPEM(seed string) ([]byte, error) {
	// Ed25519 expects a 32-byte seed. We hash the input seed to ensure it's the right length.
	h := sha256.Sum256([]byte(seed))
	privateKey := ed25519.NewKeyFromSeed(h[:])

	keyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ed25519 key: %w", err)
	}

	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	}
	return pem.EncodeToMemory(pemBlock), nil
}

// GenerateDeterministicRSAPrivateKeyPEM generates an RSA-2048 private key deterministically from a seed.
// NOTE: RSA generation in Go is not perfectly deterministic even with a fixed reader.
// This function is kept for backward compatibility but use of Ed25519 is recommended for true determinism.
func GenerateDeterministicRSAPrivateKeyPEM(seed string) ([]byte, error) {
	reader := &DeterministicReader{
		H:    sha256.New(),
		Seed: []byte(seed),
	}

	key, err := rsa.GenerateKey(reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate deterministic key: %w", err)
	}

	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	}
	return pem.EncodeToMemory(pemBlock), nil
}

// DeterministicReader is a reader that produces a deterministic stream of bytes from a seed.
type DeterministicReader struct {
	H       hash.Hash
	Seed    []byte
	Counter uint32
	Buf     []byte
}

func (r *DeterministicReader) Read(p []byte) (n int, err error) {
	for len(r.Buf) < len(p) {
		r.H.Reset()
		r.H.Write(r.Seed)
		_ = binary.Write(r.H, binary.BigEndian, r.Counter)
		r.Buf = append(r.Buf, r.H.Sum(nil)...)
		r.Counter++

		// Safety break if counter overflows (nearly impossible for RSA generation)
		if r.Counter == 0 {
			return 0, io.EOF
		}
	}
	n = copy(p, r.Buf)
	r.Buf = r.Buf[n:]
	return n, nil
}
