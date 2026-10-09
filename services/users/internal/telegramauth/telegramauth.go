// Package telegramauth checks the signatures Telegram puts on Login Widget
// payloads and Mini App initData
// (apps/api/src/telegram-auth/telegram-verify.util.ts).
package telegramauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
)

// WidgetHashValid checks a Login Widget payload: fields holds every
// provided field except "hash", already stringified as JavaScript would.
func WidgetHashValid(fields map[string]string, hash, botToken string) bool {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, len(keys))
	for i, k := range keys {
		lines[i] = k + "=" + fields[k]
	}
	secret := sha256.Sum256([]byte(botToken))
	return hmacEqual(secret[:], strings.Join(lines, "\n"), hash)
}

// InitData verifies Mini App initData and returns its parameters, or false.
func InitData(initData, botToken string) (url.Values, bool) {
	params, err := url.ParseQuery(initData)
	if err != nil {
		return nil, false
	}
	hash := params.Get("hash")
	if hash == "" {
		return nil, false
	}
	params.Del("hash")

	var lines []string
	for k, values := range params {
		for _, v := range values {
			lines = append(lines, k+"="+v)
		}
	}
	sort.Strings(lines)

	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(botToken))
	if !hmacEqual(mac.Sum(nil), strings.Join(lines, "\n"), hash) {
		return nil, false
	}
	return params, true
}

func hmacEqual(key []byte, data, expectedHex string) bool {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	expected, err := hex.DecodeString(expectedHex)
	return err == nil && hmac.Equal(mac.Sum(nil), expected)
}
