package notification

import (
	"mime"
	"strings"
)

// headerValue strips CR/LF so user-controlled text (e.g. a project title) can't
// inject extra email headers, then MIME-encodes non-ASCII text.
func headerValue(s string) string {
	s = strings.NewReplacer("\r", " ", "\n", " ").Replace(s)
	return mime.QEncoding.Encode("utf-8", s)
}
