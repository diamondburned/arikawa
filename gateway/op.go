package gateway

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/diamondburned/arikawa/utils/json"
	"github.com/diamondburned/arikawa/utils/wsutil"
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

func (g *Gateway) HandleOP(op *wsutil.OP) error {
	switch op.Code {
	case HeartbeatAckOP:
		// Heartbeat from the server?
		g.PacerLoop.Echo()

	case HeartbeatOP:
		// Server requesting a heartbeat.
		return g.PacerLoop.Pace()

	case ReconnectOP:
		// Server requests to reconnect, die and retry.
		wsutil.WSDebug("ReconnectOP received.")
		// We must reconnect in another goroutine, as running Reconnect
		// synchronously would prevent the main event loop from exiting.
		go g.Reconnect()
		// Gracefully exit with a nil let the event handler take the signal from
		// the pacemaker.
		return nil

	case InvalidSessionOP:
		// Discord expects us to sleep for no reason
		time.Sleep(time.Duration(rand.Intn(5)+1) * time.Second)

		// Invalid session, try and Identify.
		if err := g.Identify(); err != nil {
			// Can't identify, reconnect.
			go g.Reconnect()
		}
		return nil

	case HelloOP:
		// What is this OP doing here???
		return nil

	case DispatchOP:
		// Set the sequence
		if op.Sequence > 0 {
			g.Sequence.Set(op.Sequence)
		}

		// Check if we know the event
		fn, ok := EventCreator[op.EventName]
		if !ok {
			return fmt.Errorf(
				"Unknown event %s: %s",
				op.EventName, string(op.Data),
			)
		}

		// Make a new pointer to the event
		var ev = fn()

		// Try and parse the event
		if err := json.Unmarshal(op.Data, ev); err != nil {
			return errors.Wrap(err, "Failed to parse event "+op.EventName)
		}

		// If the event is a ready, we'll want its sessionID
		if ev, ok := ev.(*ReadyEvent); ok {
			g.SessionID = ev.SessionID
		}

		// Throw the event into a channel, it's valid now.
		g.Events <- ev
		return nil

	default:
		return fmt.Errorf("Unknown OP code %d (event %s)", op.Code, op.EventName)
	}

	return nil
}
