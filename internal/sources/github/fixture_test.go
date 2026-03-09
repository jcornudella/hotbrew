package github

import (
	"bytes"
	"io"
	"net/http"
	"os"
)

type fixtureRoundTripper struct {
	t        interface{ Helper(); Fatalf(string, ...any) }
	fixtures map[string]string
}

func (rt *fixtureRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.t.Helper()
	path, ok := rt.fixtures[req.URL.Scheme+"://"+req.URL.Host+req.URL.Path]
	if !ok {
		path, ok = rt.fixtures[req.URL.String()]
	}
	if !ok {
		rt.t.Fatalf("unexpected request URL: %s", req.URL.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		rt.t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(data)),
		Request:    req,
	}, nil
}
