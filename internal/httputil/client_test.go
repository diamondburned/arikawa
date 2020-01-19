package httputil

import (
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
)

func TestMeanwhileBody(t *testing.T) {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal("Can't read body:", err)
		}

		if s := string(b); s != "Hime" {
			t.Fatal("Unexpected body:", s)
		}

		w.Write([]byte("Arikawa"))
	})

	addr := startHTTP(t)
	c := NewClient()
	w := func(w io.Writer) error {
		w.Write([]byte("Hime"))
		return nil
	}

	r, err := c.MeanwhileBody(w, "GET", "http://"+addr)
	if err != nil {
		t.Fatal("Failed to send request:", err)
	}

	defer r.Body.Close()

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Fatal("Can't read body:", err)
	}

	if s := string(b); s != "Arikawa" {
		t.Fatal("Unexpected body:", s)
	}
}

func startHTTP(t *testing.T) string {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("TCP error:", err)
	}

	go func() {
		if err := http.Serve(listener, nil); err != nil {
			t.Fatal("HTTP error:", err)
		}
	}()

	return listener.Addr().(*net.TCPAddr).String()
}
