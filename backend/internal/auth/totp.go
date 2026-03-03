package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"

	"github.com/pquerna/otp/totp"
)

const backupCodeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // unambiguous characters

// GenerateSecret creates a new TOTP secret for the given account.
// Returns the base32 secret and the otpauth:// URI for QR code rendering.
func GenerateSecret(email, issuer string) (secret, otpauthURL string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: email,
	})
	if err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

// ValidateCode checks a 6-digit TOTP code against the given base32 secret.
func ValidateCode(secret, code string) bool {
	return totp.Validate(code, secret)
}

// GenerateBackupCodes produces 10 random 10-character backup codes.
// Returns both the plain-text codes (to show the user once) and their sha256 hashes (to store).
func GenerateBackupCodes() (plain []string, hashed []string) {
	for i := 0; i < 10; i++ {
		code := randomCode(10)
		h := sha256.Sum256([]byte(code))
		plain = append(plain, code)
		hashed = append(hashed, hex.EncodeToString(h[:]))
	}
	return plain, hashed
}

func randomCode(length int) string {
	result := make([]byte, length)
	charLen := big.NewInt(int64(len(backupCodeChars)))
	for i := range result {
		n, _ := rand.Int(rand.Reader, charLen)
		result[i] = backupCodeChars[n.Int64()]
	}
	return string(result)
}
