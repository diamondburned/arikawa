package httpdriver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"reflect"
)

// MockRequest is a mock request. It implements the Request interface.
type MockRequest struct {
	Method string
	URL    url.URL
	Header http.Header
	Body   []byte

	ctx context.Context
}

// NewMockRequest creates a new mock request. If any of the given parameters
// cause an error, the function will panic.
func NewMockRequest(method, urlstr string, header http.Header, jsonBody interface{}) *MockRequest {
	u, err := url.Parse(urlstr)
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
		Method: method,
		URL:    *u,
		Header: header,
		Body:   body,
		ctx:    context.Background(),
	}
}

// NewMockRequestWithContext creates a new mock request with context. If any of
// the given parameters cause an error, the function will panic.
func NewMockRequestWithContext(ctx context.Context, method, urlstr string, header http.Header, jsonBody interface{}) *MockRequest {
	req := NewMockRequest(method, urlstr, header, jsonBody)
	req.ctx = ctx
	return req
}

// ToHTTPRequest converts a mock request to a net/http request. If an error
// occurs, the function will panic.
func (r *MockRequest) ToHTTPRequest() *http.Request {
	req, err := http.NewRequestWithContext(r.ctx, r.Method, r.URL.String(), bytes.NewReader(r.Body))
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

// ExpectMockRequest asserts that the given request is a mock request that
// matches what is expected. The given request for got must be of type
// *MockRequest. The given t function can either be (*testing.T).Errorf or
// (*testing.T).Fatalf.
func ExpectMockRequest(t func(f string, args ...interface{}), expected *MockRequest, gotAny Request) {
	got, ok := gotAny.(*MockRequest)
	if !ok {
		t("got unexpected request type %T", gotAny)
		return
	}

	if expected.Method != got.Method {
		t("unexpected method %q, got %q", expected.Method, got.Method)
		return
	}

	if expected.URL.String() != got.URL.String() {
		t("unexpected URL %q, got %q", expected.URL.String(), got.URL.String())
		return
	}

	for expectK, expectV := range expected.Header {
		gotV, ok := got.Header[expectK]
		if !ok {
			t("unexpected header key %q, got none", expectK)
			return
		}

		if !reflect.DeepEqual(expectV, gotV) {
			t("unexpected header key %q to have value %q, got %q", expectK, expectV, gotV)
			return
		}
	}

	if !bytes.Equal(expected.Body, got.Body) {
		t("unexpected body:\n"+
			"expected %q\n"+
			"got      %q", expected.Body, got.Body)
		return
	}
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
