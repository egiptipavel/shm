package url

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	defaultUrlScheme = "http://"
	urlRegex         = regexp.MustCompile(`^(https?:\/\/)?([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,63}/?$`)
)

func ConvertToExpectedUrl(url string) (string, error) {
	if !urlRegex.MatchString(url) {
		return url, fmt.Errorf("invalid url")
	}

	if !strings.HasPrefix(url, "http://") || !strings.HasPrefix(url, "https://") {
		url = defaultUrlScheme + url
	}

	url, _ = strings.CutSuffix(url, "/")
	return url, nil
}
