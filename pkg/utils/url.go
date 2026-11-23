package utils

import (
	"errors"
	"net/url"
	"strings"
)

// ValidateHTTPURL accepts only absolute http(s) URLs, rejecting schemes such as
// javascript: or data: that would be dangerous if rendered as a link.
func ValidateHTTPURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("url must be an absolute http or https URL")
	}
	return nil
}
