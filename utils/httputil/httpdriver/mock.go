package httpdriver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

// MockRequest is a mock request. It implements the Request interface.
type MockRequest struct {
	URL    url.URL
	Header http.Header
	Body   []byte

	ctx context.Context
}

// NewMockRequest creates a new mock request.
func NewMockRequest(urlStr string, header http.Header, jsonBody interface{}) *MockRequest {
	url, err := url.Parse(urlStr)
	if err != nil {
		panic(err)
	}

	var body []byte
	if jsonBody != nil {
		body, err = json.Marshal(jsonBody)
		if err != nil {
			panic(err)
		}
	}

	return &MockRequest{
		URL:    *url,
		Header: header,
		Body:   body,
		ctx:    context.Background(),
	}
}

// NewMockRequestWithContext creates a new mock request with context.
func NewMockRequestWithContext(ctx context.Context, urlStr string, header http.Header, jsonBody interface{}) *MockRequest {
	req := NewMockRequest(urlStr, header, jsonBody)
	req.ctx = ctx
	return req
}

// ToHTTPRequest converts a mock request to a http request.
func (r *MockRequest) ToHTTPRequest() *http.Request {
	req, err := http.NewRequestWithContext(r.ctx, http.MethodGet, r.URL.String(), bytes.NewReader(r.Body))
	if err != nil {
		panic(err)
	}
	req.Header = r.Header
	return req
}

func (r *MockRequest) GetPath() string {
	return r.URL.Path
}

func (r *MockRequest) GetContext() context.Context {
	return r.ctx
}

func (r *MockRequest) AddHeader(h http.Header) {
	for k, v := range h {
		r.Header[k] = append(r.Header[k], v...)
	}
}

func (r *MockRequest) AddQuery(v url.Values) {
	oldv := r.URL.Query()
	for k, v := range v {
		oldv[k] = append(oldv[k], v...)
	}
	r.URL.RawQuery = oldv.Encode()
}

func (r *MockRequest) WithBody(body io.ReadCloser) {
	r.Body, _ = io.ReadAll(body)
	body.Close()
}

// MockResponse is a mock response. It implements the Response interface.
type MockResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// NewMockResponse creates a new mock response.
func NewMockResponse(code int, h http.Header, jsonBody interface{}) *MockResponse {
	var body []byte
	if jsonBody != nil {
		var err error
		body, err = json.Marshal(jsonBody)
		if err != nil {
			panic(err)
		}
	}

	return &MockResponse{
		StatusCode: code,
		Header:     h,
		Body:       body,
	}
}

func (r *MockResponse) GetStatus() int {
	return r.StatusCode
}

func (r *MockRequest) GetHeader() http.Header {
	return r.Header
}

func (r *MockRequest) GetBody() io.ReadCloser {
	return io.NopCloser(bytes.NewReader(r.Body))
}
