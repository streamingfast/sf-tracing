package tracing

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetURLParts(t *testing.T) {
	tests := []struct {
		url      string
		expected urlParts
	}{
		{
			url: "https://api.smith.langchain.com/otel/v1/traces?x-api-key=aa&LANGSMITH_PROJECT=bb",
			expected: urlParts{
				endpoint: "api.smith.langchain.com",
				urlPath:  "/otel/v1/traces",
				headers: map[string]string{
					"x-api-key":         "aa",
					"LANGSMITH_PROJECT": "bb",
				},
			},
		},
		{
			url: "https://api.smith.langchain.com",
			expected: urlParts{
				endpoint: "api.smith.langchain.com",
				urlPath:  "",
				headers:  map[string]string{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.url, func(t *testing.T) {
			u, err := url.Parse(test.url)
			require.NoError(t, err)
			parts := getURLParts(u)
			require.Equal(t, test.expected, parts)
		})
	}
}
