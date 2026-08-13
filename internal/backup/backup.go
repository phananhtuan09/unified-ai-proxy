package backup

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/argon2"

	"github.com/tuanp-github/unified-ai-proxy/internal/config"
)

const (
	formatVersion = 1
	kdfName       = "argon2id"
	cipherName    = "aes-256-gcm"

	argonTime    = 3
	argonMemory  = 64 * 1024 // 64 MiB
	argonThreads = 4
	keyLen       = 32
)

// Payload is the decrypted backup content.
type Payload struct {
	Config  string                     `json:"config"`
	Tokens  map[string]json.RawMessage `json:"tokens"`
	APIKeys []string                   `json:"api_keys"`
}

type kdfParams struct {
	Name    string `json:"name"`
	Salt    string `json:"salt"`
	Time    uint32 `json:"time"`
	Memory  uint32 `json:"memory"`
	Threads uint8  `json:"threads"`
	KeyLen  uint32 `json:"key_len"`
}

type cipherParams struct {
	Name  string `json:"name"`
	Nonce string `json:"nonce"`
}

type envelope struct {
	Version int          `json:"version"`
	KDF     kdfParams    `json:"kdf"`
	Cipher  cipherParams `json:"cipher"`
	Data    string       `json:"data"`
}

// Export writes an encrypted backup of config, tokens, and API keys.
func Export(cfg *config.Config, configPath, password, output string) error {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	payload := Payload{
		Config:  string(raw),
		Tokens:  map[string]json.RawMessage{},
		APIKeys: cfg.Server.APIKeys,
	}

	for _, p := range cfg.Providers {
		if !p.Enabled {
			continue
		}
		for _, a := range p.Accounts {
			if a.TokenFile == "" {
				continue
			}
			path := config.ExpandPath(a.TokenFile)
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read token file %s: %w", path, err)
			}
			payload.Tokens[path] = json.RawMessage(data)
		}
	}

	plaintext, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	env, err := seal(plaintext, password)
	if err != nil {
		return err
	}
	out, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}
	if err := os.WriteFile(output, out, 0o600); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}
	return nil
}

// Import decrypts a backup and returns its payload.
func Import(input, password string) (*Payload, error) {
	data, err := os.ReadFile(input)
	if err != nil {
		return nil, fmt.Errorf("read backup: %w", err)
	}
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("parse backup: %w", err)
	}
	if env.Version != formatVersion {
		return nil, fmt.Errorf("unsupported backup version %d", env.Version)
	}

	plaintext, err := open(&env, password)
	if err != nil {
		return nil, err
	}
	var payload Payload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}
	return &payload, nil
}

// Restore writes the payload config and token files with secure permissions.
func Restore(payload *Payload, configPath string) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(configPath, []byte(payload.Config), 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if err := os.Chmod(configPath, 0o600); err != nil {
		return fmt.Errorf("chmod config: %w", err)
	}
	for path, raw := range payload.Tokens {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return fmt.Errorf("create token directory: %w", err)
		}
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			return fmt.Errorf("write token file %s: %w", path, err)
		}
		if err := os.Chmod(path, 0o600); err != nil {
			return fmt.Errorf("chmod token file %s: %w", path, err)
		}
	}
	return nil
}

func seal(plaintext []byte, password string) (*envelope, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, keyLen)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &envelope{
		Version: formatVersion,
		KDF: kdfParams{
			Name:    kdfName,
			Salt:    base64.StdEncoding.EncodeToString(salt),
			Time:    argonTime,
			Memory:  argonMemory,
			Threads: argonThreads,
			KeyLen:  keyLen,
		},
		Cipher: cipherParams{
			Name:  cipherName,
			Nonce: base64.StdEncoding.EncodeToString(nonce),
		},
		Data: base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

func open(env *envelope, password string) ([]byte, error) {
	salt, err := base64.StdEncoding.DecodeString(env.KDF.Salt)
	if err != nil {
		return nil, fmt.Errorf("decode salt: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(env.Cipher.Nonce)
	if err != nil {
		return nil, fmt.Errorf("decode nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(env.Data)
	if err != nil {
		return nil, fmt.Errorf("decode data: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, env.KDF.Time, env.KDF.Memory, env.KDF.Threads, env.KDF.KeyLen)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt failed (wrong password?): %w", err)
	}
	return plaintext, nil
}
