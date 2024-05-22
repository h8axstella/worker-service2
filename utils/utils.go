package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

func CreateSignature(secretKey string, method string, endpoint string, params string) string {
	timestamp := fmt.Sprintf("%d", time.Now().UnixNano()/1e6)
	message := fmt.Sprintf("%s&%s", params, "tonce="+timestamp)
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(message))
	signature := hex.EncodeToString(mac.Sum(nil))
	return signature
}
