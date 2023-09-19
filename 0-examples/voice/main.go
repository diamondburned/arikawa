package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/voice"
	"github.com/diamondburned/arikawa/v3/voice/udp"
	"github.com/diamondburned/oggreader"
	"errors"
)

func main() {
	flag.Parse()

	file := flag.Arg(0)
	if file == "" {
		log.Fatalln("usage:", filepath.Base(os.Args[0]), "<audio file>")
	}

	voiceID, err := discord.ParseSnowflake(os.Getenv("VOICE_ID"))
	if err != nil {
		log.Fatalln("failed to parse $VOICE_ID:", err)
	}
	chID := discord.ChannelID(voiceID)

	state := state.New("Bot " + os.Getenv("BOT_TOKEN"))
	voice.AddIntents(state)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := state.Open(ctx); err != nil {
		log.Fatalln("failed to open:", err)
	}
	defer state.Close()

	if err := start(ctx, state, chID, file); err != nil {
		// Ignore context canceled errors as they're often intentional.
		if !errors.Is(err, context.Canceled) {
			log.Fatalln(err)
		}
	}
}

// Optional constants to tweak the Opus stream.
const (
	frameDuration = 60 // ms
	timeIncrement = 2880
)

func start(ctx context.Context, s *state.State, id discord.ChannelID, file string) error {
	v, err := voice.NewSession(s)
	if err != nil {
		return errors.Wrap(err, "cannot make new voice session")
	}

	// Optimize Opus frame duration. This step is optional, but it is
	// recommended.
	v.SetUDPDialer(udp.DialFuncWithFrequency(
		frameDuration*time.Millisecond, // correspond to -frame_duration
		timeIncrement,
	))

	ffmpeg := exec.CommandContext(ctx,
		"ffmpeg", "-hide_banner", "-loglevel", "error",
		// Streaming is slow, so a single thread is all we need.
		"-threads", "1",
		// Input file.
		"-i", file,
		// Output format; leave as "libopus".
		"-c:a", "libopus",
		// Bitrate in kilobits. This doesn't matter, but I recommend 96k as the
		// sweet spot.
		"-b:a", "96k",
		// Frame duration should be the same as what's given into
		// udp.DialFuncWithFrequency.
		"-frame_duration", strconv.Itoa(frameDuration),
		// Disable variable bitrate to keep packet sizes consistent. This is
		// optional.
		"-vbr", "off",
		// Output format, which is opus, so we need to unwrap the opus file.
		"-f", "opus",
		"-",
	)

	ffmpeg.Stderr = os.Stderr

	stdout, err := ffmpeg.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "failed to get stdout pipe")
	}

	// Kickstart FFmpeg before we join. FFmpeg will wait until we start
	// consuming the stream to process further.
	if err := ffmpeg.Start(); err != nil {
		return errors.Wrap(err, "failed to start ffmpeg")
	}

	// Join the voice channel.
	if err := v.JoinChannelAndSpeak(ctx, id, false, true); err != nil {
		return errors.Wrap(err, "failed to join channel")
	}
	defer v.Leave(ctx)

	// Start decoding FFmpeg's OGG-container output and extract the raw Opus
	// frames into the stream.
	if err := oggreader.DecodeBuffered(v, stdout); err != nil {
		return errors.Wrap(err, "failed to decode ogg")
	}

	// Wait until FFmpeg finishes writing entirely and leave.
	if err := ffmpeg.Wait(); err != nil {
		return errors.Wrap(err, "failed to finish ffmpeg")
	}

	return nil
}
