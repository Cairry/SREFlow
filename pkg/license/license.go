package license

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"
)

// License represents a license payload
type License struct {
	Holder    string `json:"holder"`
	IssuedAt  int64  `json:"issued_at"`
	ExpiresAt int64  `json:"expires_at"`
	MachineID string `json:"machine_id,omitempty"`
}

// LicenseFile represents the signed license file on disk
type LicenseFile struct {
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
	PublicKey string `json:"public_key"`
}

// IsExpired checks if the license has expired
func (l *License) IsExpired() bool {
	if l.ExpiresAt == 0 {
		return false
	}
	return time.Now().Unix() > l.ExpiresAt
}

// GenerateKeyPair generates an RSA key pair and writes the private key to disk.
func GenerateKeyPair(privateKeyPath string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})
	if err := os.WriteFile(privateKeyPath, privPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	fmt.Printf("Public key:\n%s", pubPEM)

	return nil
}

// Sign generates a signed license file from a License using the private key.
// The public key is embedded in the resulting LicenseFile so users only need the .lic file.
func Sign(lic License, privateKeyPath string) (*LicenseFile, error) {
	privateKey, err := loadPrivateKey(privateKeyPath)
	if err != nil {
		return nil, err
	}

	payloadBytes, err := json.Marshal(lic)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal license: %w", err)
	}

	hash := sha256.Sum256(payloadBytes)
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign license: %w", err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})

	return &LicenseFile{
		Payload:   base64.StdEncoding.EncodeToString(payloadBytes),
		Signature: base64.StdEncoding.EncodeToString(signature),
		PublicKey: base64.StdEncoding.EncodeToString(pubPEM),
	}, nil
}

// Verify validates a license file using the public key embedded in the file
func Verify(licFile LicenseFile) (*License, error) {
	publicKey, err := parsePublicKey(licFile.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key from license: %w", err)
	}

	payloadBytes, err := base64.StdEncoding.DecodeString(licFile.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	signature, err := base64.StdEncoding.DecodeString(licFile.Signature)
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	hash := sha256.Sum256(payloadBytes)
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], signature); err != nil {
		return nil, fmt.Errorf("license signature verification failed: %w", err)
	}

	var lic License
	if err := json.Unmarshal(payloadBytes, &lic); err != nil {
		return nil, fmt.Errorf("failed to parse license payload: %w", err)
	}

	if lic.IsExpired() {
		return nil, errors.New("license has expired")
	}

	return &lic, nil
}

// LoadLicenseFile reads and parses a license file from disk
func LoadLicenseFile(path string) (*LicenseFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read license file: %w", err)
	}

	var licFile LicenseFile
	if err := json.Unmarshal(data, &licFile); err != nil {
		return nil, fmt.Errorf("failed to parse license file: %w", err)
	}

	return &licFile, nil
}

// VerifyFromFile validates a license file at the given path
func VerifyFromFile(licensePath string) (*License, error) {
	licFile, err := LoadLicenseFile(licensePath)
	if err != nil {
		return nil, err
	}
	return Verify(*licFile)
}

// parsePublicKey decodes a base64-encoded PEM public key
func parsePublicKey(pubKeyBase64 string) (*rsa.PublicKey, error) {
	pubPEM, err := base64.StdEncoding.DecodeString(pubKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key base64: %w", err)
	}

	block, _ := pem.Decode(pubPEM)
	if block == nil {
		return nil, errors.New("failed to decode public key PEM")
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not an RSA key")
	}

	return rsaKey, nil
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return key, nil
}
