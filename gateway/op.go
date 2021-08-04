package gateway

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/diamondburned/arikawa/v3/utils/json"
	"github.com/diamondburned/arikawa/v3/utils/wsutil"
	"github.com/pkg/errors"
)

type OPCode = wsutil.OPCode

const (
	DispatchOP            OPCode = 0 // recv
	HeartbeatOP           OPCode = 1 // send/recv
	IdentifyOP            OPCode = 2 // send...
	StatusUpdateOP        OPCode = 3 //
	VoiceStateUpdateOP    OPCode = 4 //
	VoiceServerPingOP     OPCode = 5 //
	ResumeOP              OPCode = 6 //
	ReconnectOP           OPCode = 7 // recv
	RequestGuildMembersOP OPCode = 8 // send
	InvalidSessionOP      OPCode = 9 // recv...
	HelloOP               OPCode = 10
	HeartbeatAckOP        OPCode = 11
	CallConnectOP         OPCode = 13
	GuildSubscriptionsOP  OPCode = 14
)

// ErrReconnectRequest is returned by HandleOP if a ReconnectOP is given. This
// is used mostly internally to signal the heartbeat loop to reconnect, if
// needed. It is not a fatal error.
var ErrReconnectRequest = errors.New("ReconnectOP received")

func (g *Gateway) HandleOP(op *wsutil.OP) error {
	switch op.Code {
	case HeartbeatAckOP:
		// Heartbeat from the server?
		g.PacerLoop.Echo()

	case HeartbeatOP:
		ctx, cancel := context.WithTimeout(context.Background(), g.WSTimeout)
		defer cancel()

		// Server requesting a heartbeat.
		if err := g.PacerLoop.Pace(ctx); err != nil {
			return wsutil.ErrBrokenConnection(errors.Wrap(err, "failed to pace"))
		}

	case ReconnectOP:
		// Server requests to Reconnect, die and retry.
		wsutil.WSDebug("ReconnectOP received.")

		// Exit with the ReconnectOP error to force the heartbeat event loop to
		// Reconnect synchronously. Not really a fatal error.
		return wsutil.ErrBrokenConnection(ErrReconnectRequest)

	case InvalidSessionOP:
		// Discord expects us to sleep for no reason
		time.Sleep(time.Duration(rand.Intn(5)+1) * time.Second)

		ctx, cancel := context.WithTimeout(context.Background(), g.WSTimeout)
		defer cancel()

		// Invalid session, try and Identify.
		if err := g.IdentifyCtx(ctx); err != nil {
			// Can't identify, Reconnect.
			return wsutil.ErrBrokenConnection(ErrReconnectRequest)
		}

		return nil

	case HelloOP:
		return nil

	case DispatchOP:
		// Set the sequence
		if op.Sequence > 0 {
			g.Sequence.Set(op.Sequence)
		}

		// Check if we know the event
		fn, ok := EventCreator[op.EventName]
		if !ok {
			return &wsutil.UnknownEventError{
				Name: op.EventName,
				Data: op.Data,
			}
		}

		// Make a new pointer to the event
		var ev = fn()

		// Try and parse the event
		if err := json.Unmarshal(op.Data, ev); err != nil {
			return errors.Wrap(err, "failed to parse event "+op.EventName)
		}

		// If the event is a ready, we'll want its sessionID
		if ev, ok := ev.(*ReadyEvent); ok {
			g.sessionMu.Lock()
			g.sessionID = ev.SessionID
			g.sessionMu.Unlock()
		}

		// Throw the event into a channel; it's valid now.
		g.Events <- ev
		return nil

	default:
		return fmt.Errorf("unknown OP code %d (event %s)", op.Code, op.EventName)
	}

	return nil
}
