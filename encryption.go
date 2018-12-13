// modified from
// https://www.thepolyglotdeveloper.com/2018/02/encrypt-decrypt-data-golang-application-crypto-packages/
package portwarden

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	ErrMessageAuthenticationFailed = "cipher: message authentication failed"
	ErrWrongBackupPassphrase       = "wrong backup passphrase entered"
)

// derive a key from the master password
func DeriveKey(passphrase string) []byte {
	return pbkdf2.Key([]byte(passphrase), []byte(Salt), 4096, 32, sha256.New)
}

func EncryptBytes(data []byte, passphrase string) ([]byte, error) {
	block, _ := aes.NewCipher(DeriveKey(passphrase))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return []byte{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return []byte{}, err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func DecryptBytes(data []byte, passphrase string) ([]byte, error) {
	key := DeriveKey(passphrase)
	block, err := aes.NewCipher(key)
	if err != nil {
		return []byte{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return []byte{}, err
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		if err.Error() == ErrMessageAuthenticationFailed {
			return []byte{}, errors.New(ErrWrongBackupPassphrase)
		}
		return []byte{}, err
	}
	return plaintext, nil
}
