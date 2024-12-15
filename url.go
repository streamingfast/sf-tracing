package tracing

import "net/url"

type urlParts struct {
	endpoint string
	urlPath  string
	headers  map[string]string
}

func getURLParts(u *url.URL) urlParts {
	headers := make(map[string]string)
	for k, v := range u.Query() {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	return urlParts{
		endpoint: u.Host,
		urlPath:  u.Path,
		headers:  headers,
	}
}
