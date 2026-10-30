package src

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func cleanTitle(s string) string {
	return clip(strings.Join(strings.Fields(s), " "), 240)
}

func clip(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func normalizeDue(value *int64) *int64 {
	if value == nil || *value <= 0 {
		return nil
	}
	return value
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func boolIntString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
