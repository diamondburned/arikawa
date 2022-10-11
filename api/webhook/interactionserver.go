package webhook

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/pkg/errors"
)

func writeError(w http.ResponseWriter, code int, err error) {
	var resp struct {
		Error string `json:"error"`
	}

	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.Error = http.StatusText(code)
	}

	b, err := json.Marshal(resp)
	if err != nil {
		log.Panicln("cannot marshal error response:", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(b)
}

// InteractionHandler is a type whose method is called on every incoming event.
type InteractionHandler interface {
	// HandleInteraction is expected to return a response synchronously, either
	// to be followed-up later by deferring the response or to be responded
	// immediately.
	HandleInteraction(*discord.InteractionEvent) *api.InteractionResponse
}

// InteractionHandlerFunc is a function type that implements the interface.
type InteractionHandlerFunc func(*discord.InteractionEvent) *api.InteractionResponse

var _ InteractionHandler = InteractionHandlerFunc(nil)

func (f InteractionHandlerFunc) HandleInteraction(ev *discord.InteractionEvent) *api.InteractionResponse {
	return f(ev)
}

type alwaysDeferInteraction struct {
	f     func(*discord.InteractionEvent)
	flags discord.MessageFlags
}

// AlwaysDeferInteraction always returns a DeferredMessageInteractionWithSource
// then invokes f in the background. This allows f to always use the follow-up
// functions.
func AlwaysDeferInteraction(flags discord.MessageFlags, f func(*discord.InteractionEvent)) InteractionHandler {
	return alwaysDeferInteraction{f, flags}
}

func (f alwaysDeferInteraction) HandleInteraction(ev *discord.InteractionEvent) *api.InteractionResponse {
	go f.f(ev)
	return &api.InteractionResponse{
		Type: api.DeferredMessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Flags: f.flags,
		},
	}
}

// InteractionErrorFunc is called to write an error. err may be nil with a
// non-2xx code.
type InteractionErrorFunc func(w http.ResponseWriter, r *http.Request, code int, err error)

// InteractionServer provides a HTTP handler to verify and handle Interaction
// Create events sent by Discord into a HTTP endpoint..
type InteractionServer struct {
	ErrorFunc InteractionErrorFunc

	interactionHandler InteractionHandler
	httpHandler        http.Handler
	pubkey             ed25519.PublicKey
}

// NewInteractionServer creates a new InteractionServer instance. pubkey should
// be hex-encoded.
func NewInteractionServer(pubkey string, handler InteractionHandler) (*InteractionServer, error) {
	pubkeyB, err := hex.DecodeString(pubkey)
	if err != nil {
		return nil, errors.Wrap(err, "cannot decode hex pubkey")
	}

	s := InteractionServer{
		ErrorFunc: func(w http.ResponseWriter, r *http.Request, code int, err error) {
			writeError(w, code, err)
		},
		interactionHandler: handler,
		httpHandler:        nil,
		pubkey:             pubkeyB,
	}

	s.httpHandler = http.HandlerFunc(s.handle)
	if len(s.pubkey) != 0 {
		s.httpHandler = s.withVerification(s.httpHandler)
	}

	return &s, nil
}

// ServeHTTP implements http.Handler.
func (s *InteractionServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(s.pubkey) != 0 {
		s.withVerification(http.HandlerFunc(s.handle)).ServeHTTP(w, r)
	} else {
		http.HandlerFunc(s.handle).ServeHTTP(w, r)
	}
}

func (s *InteractionServer) handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var ev discord.InteractionEvent

		if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
			s.ErrorFunc(w, r, 400, errors.Wrap(err, "cannot decode interaction body"))
			return
		}

		switch ev.Data.(type) {
		case *discord.PingInteraction:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(api.InteractionResponse{
				Type: api.PongInteraction,
			})
		}

		resp := s.interactionHandler.HandleInteraction(&ev)
		if resp != nil && resp.Type != api.PongInteraction {
			if resp.NeedsMultipart() {
				body := multipart.NewWriter(w)
				w.Header().Set("Content-Type", body.FormDataContentType())
				resp.WriteMultipart(body)
			} else {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			}
		}
	default:
		s.ErrorFunc(w, r, http.StatusMethodNotAllowed, errors.New("method not allowed"))
	}
}

// withVerification was written thanks to @bsdlp and their code
// https://github.com/bsdlp/discord-interactions-go/blob/a2ba844/interactions/verify_example_test.go#L63.
func (s *InteractionServer) withVerification(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		signature := r.Header.Get("X-Signature-Ed25519")
		if signature == "" {
			s.ErrorFunc(w, r, 401, errors.New("missing header X-Signature-Ed25519"))
			return
		}

		sig, err := hex.DecodeString(signature)
		if err != nil {
			s.ErrorFunc(w, r, 400, errors.Wrap(err, "X-Signature-Ed25519 is not valid hex-encoded"))
			return
		}

		if len(sig) != ed25519.SignatureSize || sig[63]&224 != 0 {
			s.ErrorFunc(w, r, 400, errors.New("invalid X-Signature-Ed25519 data"))
			return
		}

		timestamp := r.Header.Get("X-Signature-Timestamp")
		if timestamp == "" {
			s.ErrorFunc(w, r, 401, errors.New("missing header X-Signature-Timestamp"))
			return
		}

		var msg bytes.Buffer
		msg.Grow(int(r.ContentLength+1) + len(timestamp))
		msg.WriteString(timestamp)

		if _, err := io.Copy(&msg, r.Body); err != nil {
			s.ErrorFunc(w, r, 500, errors.Wrap(err, "cannot read body"))
			return
		}

		if !ed25519.Verify(s.pubkey, msg.Bytes(), sig) {
			s.ErrorFunc(w, r, 401, errors.New("signature mismatch"))
			return
		}

		// Return the request body for use.
		body := msg.Bytes()[len(timestamp):]
		r.Body = io.NopCloser(bytes.NewReader(body))

		next.ServeHTTP(w, r)
	})
}
