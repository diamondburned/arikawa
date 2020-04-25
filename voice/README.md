# Voice

## Terminology
* **Discord Gateway** - The standard Discord Gateway users connect to and receive update events from
* **Discord Voice Gateway** - The Discord Voice gateway that allows voice connections to be configured
* **Voice Server** - What the Discord Voice Gateway allows connection to for sending of Opus voice packets over UDP
* **Voice Packet** - Opus encoded UDP packet that contains audio
* **Application** - Could be a custom Discord Client or Bot (nothing that is within this package)
* **Library** - Code within this package

## Connection Flow
* The **application** would get a new `*Voice` instance by calling `NewVoice()`
* When the **application** wants to connect to a voice channel they would call `JoinChannel()` on
the stored `*Voice` instance

---

* The **library** sends a [Voice State Update](https://discordapp.com/developers/docs/topics/voice-connections#retrieving-voice-server-information-gateway-voice-state-update-example)
to the **Discord Gateway**.
* The **library** waits until it receives a [Voice Server Update](https://discordapp.com/developers/docs/topics/voice-connections#retrieving-voice-server-information-example-voice-server-update-payload)
from the **Discord Gateway**.
* Once a *Voice Server Update* event is received, a new connection is opened to the **Discord Voice Gateway**.

---

* The **Discord Voice Gateway** will first send a [Hello Event](https://discordapp.com/developers/docs/topics/voice-connections#heartbeating-example-hello-payload-since-v3)
which will be used to create a new `*heart.PacemakerLoop` and start sending heartbeats to the **Discord Voice Gateway**.
* Afterwards, an [Identify Command](https://discordapp.com/developers/docs/topics/voice-connections#establishing-a-voice-websocket-connection-example-voice-identify-payload)
or [Resume Command](https://discordapp.com/developers/docs/topics/voice-connections#resuming-voice-connection-example-resume-connection-payload)
is sent to the **Discord Voice Gateway** depending on whether or not the **library** is reconnecting.

---

* The **Discord Voice Gateway** should then respond with a [Ready Event](https://discordapp.com/developers/docs/topics/voice-connections#establishing-a-voice-websocket-connection-example-voice-ready-payload)
once the connection is opened, providing the required information to connect to a **Voice Server**.
* Using the information provided in the *Ready Event*, a new UDP connection is opened to the **Voice Server**
and [IP Discovery](https://discordapp.com/developers/docs/topics/voice-connections#ip-discovery) occurs.
* After *IP Discovery* returns the **Application**'s external ip and port it connected to the **Voice Server**
with, the **library** sends a [Select Protocol Event](https://discordapp.com/developers/docs/topics/voice-connections#establishing-a-voice-udp-connection-example-select-protocol-payload)
to the **Discord Voice Gateway**.
* The **library** waits until it receives a [Session Description Event](https://discordapp.com/developers/docs/topics/voice-connections#establishing-a-voice-udp-connection-example-session-description-payload)
from the **Discord Voice Gateway**.
* Once the *Session Description Event* is received, [Speaking Events](https://discordapp.com/developers/docs/topics/voice-connections#speaking-example-speaking-payload)
and **Voice Packets** can begin to be sent to the **Discord Voice Gateway** and **Voice Server** respectively.

## Usage
* The **application** would get a new `*Voice` instance by calling `NewVoice()` and keep it
stored for when it needs to open voice connections.
* When the **application** wants to connect to a voice channel they would call `JoinChannel()` on
the stored `*Voice` instance.
* `JoinChannel()` will block as it follows the [Connection Flow](#connection-flow), returning an
`error` if one occurs and a `*voice.Session` if it was successful.
* The **application** should now call `(*voice.Session).Speaking()` with the wanted [voice flag](https://discordapp.com/developers/docs/topics/voice-connections#speaking)
(`voicegateway.Microphone`, `voicegateway.Soundshare`, or `voicegateway.Priority`).
* The **application** can now send **Voice Packets** using the `(*voice.Session).Write()` method
which will be sent to the **Voice Server**. `(*voice.Session)` also implements `io.Writer`.
* When the **application** wants to stop sending **Voice Packets** they should call
`(*voice.Session).StopSpeaking()`, then any required voice cleanup (closing streams, etc.), then
`(*voice.Session).Disconnect()`

## Examples

Check the integration tests at [voice/integration_test.go](https://github.com/diamondburned/arikawa/blob/master/voice/integration_test.go).
