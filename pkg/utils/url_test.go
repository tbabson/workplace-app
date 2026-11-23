package utils

import "testing"

func TestValidateHTTPURL(t *testing.T) {
	for _, ok := range []string{"https://example.com/file.pdf", "http://files.local/x"} {
		if err := ValidateHTTPURL(ok); err != nil {
			t.Errorf("%q rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{"javascript:alert(1)", "data:text/html,hi", "/relative", "ftp://x.com/a", ""} {
		if err := ValidateHTTPURL(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}
