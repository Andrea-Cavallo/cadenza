package session

import (
	"crypto/sha1"
	"encoding/hex"
)

// MessageHash calcola SHA1 della sequenza di messaggi.
// Due sessioni con la stessa history LLM producono lo stesso hash → stesso nome file.
func MessageHash(msgs []Message) string {
	h := sha1.New()
	for _, m := range msgs {
		h.Write([]byte(m.Role))
		h.Write([]byte{0})
		h.Write([]byte(m.Content))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
