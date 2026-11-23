package notification

import (
	"strings"
	"testing"
)

func TestHeaderValueBlocksInjection(t *testing.T) {
	got := headerValue("Project\r\nBcc: victim@example.com")
	if strings.ContainsAny(got, "\r\n") {
		t.Fatalf("header value still contains a line break: %q", got)
	}
}
