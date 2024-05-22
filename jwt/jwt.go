package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

type Header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type Claims struct {
	Username string `json:"username"`
	Exp      int64  `json:"exp"`
}

type Token struct {
	Header    Header
	Claims    Claims
	Signature string
}

var secretKey = []byte("supersecretkey")

func encodeBase64(input []byte) string {
	return base64.RawURLEncoding.EncodeToString(input)
}

func createSignature(header, payload string) string {
	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(header + "." + payload))
	return encodeBase64(h.Sum(nil))
}

func GenerateJWT(username string, expirationTime time.Duration) string {
	header := Header{Alg: "HS256", Typ: "JWT"}
	claims := Claims{
		Username: username,
		Exp:      time.Now().Add(expirationTime).Unix(),
	}

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerBase64 := encodeBase64(headerJSON)
	claimsBase64 := encodeBase64(claimsJSON)
	signature := createSignature(headerBase64, claimsBase64)

	return headerBase64 + "." + claimsBase64 + "." + signature
}

func ValidateJWT(tokenString string) (Claims, bool) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return Claims{}, false
	}

	signature := createSignature(parts[0], parts[1])
	if signature != parts[2] {
		return Claims{}, false
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, false
	}

	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return Claims{}, false
	}

	if claims.Exp < time.Now().Unix() {
		return Claims{}, false
	}

	return claims, true
}
