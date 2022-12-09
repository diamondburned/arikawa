package httpdriver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
)

var (
	_ Request  = (*MockRequest)(nil)
	_ Response = (*MockResponse)(nil)
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
	if r.Header == nil {
		r.Header = make(http.Header)
	}

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
// *MockRequest.
func ExpectMockRequest(expected *MockRequest, gotAny Request) error {
	got, ok := gotAny.(*MockRequest)
	if !ok {
		return fmt.Errorf("got unexpected request type %T", gotAny)
	}

	if expected.Method != got.Method {
		return fmt.Errorf("unexpected method %q, got %q", expected.Method, got.Method)
	}

	if expected.URL.String() != got.URL.String() {
		return fmt.Errorf("unexpected URL %q, got %q", expected.URL.String(), got.URL.String())
	}

	for expectK, expectV := range expected.Header {
		gotV, ok := got.Header[expectK]
		if !ok {
			return fmt.Errorf("unexpected header key %q, got none", expectK)
		}

		if !reflect.DeepEqual(expectV, gotV) {
			return fmt.Errorf("unexpected header key %q to have value %q, got %q", expectK, expectV, gotV)
		}
	}

	body1 := bytes.TrimRight(expected.Body, "\n")
	body2 := bytes.TrimRight(got.Body, "\n")
	if !bytes.Equal(body1, body2) {
		return fmt.Errorf("unexpected body:\n"+
			"expected %q\n"+
			"got      %q", expected.Body, got.Body)
	}

	return nil
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

func (r *MockResponse) GetHeader() http.Header {
	return r.Header
}

func (r *MockResponse) GetBody() io.ReadCloser {
	return io.NopCloser(bytes.NewReader(r.Body))
}
