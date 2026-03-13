package detect

import "net/url"

// allowedSchemes are the URL schemes we recognize as openable URLs.
var allowedSchemes = map[string]bool{
	"http":  true,
	"https": true,
	"ftp":   true,
}

// IsURL returns true if s is a valid URL with an http, https, or ftp scheme.
// Strings without a scheme (e.g., "example.com") are NOT considered URLs.
func IsURL(s string) bool {
	if s == "" {
		return false
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return allowedSchemes[u.Scheme] && u.Host != ""
}
