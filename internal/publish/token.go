package publish

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const publicationTokenBytes = 32

// GenerateToken 生成不可猜测的订阅访问令牌。
func GenerateToken() (string, error) {
	buf := make([]byte, publicationTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate publication token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
