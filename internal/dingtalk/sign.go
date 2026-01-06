package dingtalk

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func Sign(timestampMillis int64, secret string) string {
	toSign := fmt.Sprintf("%d\n%s", timestampMillis, secret)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(toSign))
	sum := h.Sum(nil)
	return base64.StdEncoding.EncodeToString(sum)
}
