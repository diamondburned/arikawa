package examples_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	_ "embed"

	"github.com/diamondburned/arikawa/v3/internal/testenv"
)

//go:embed integration_exclude.txt
var integrationExclude string

func TestExamples(t *testing.T) {
	// Assert that the tests only run when the environment variables are set.
	testenv.Must(t)

	// Assert that the Go compiler is available.
	_, err := exec.LookPath("go")
	if err != nil {
		t.Skip("skipping test; go compiler not found")
	}

	excluded := make(map[string]bool)
	for _, line := range strings.Split(string(integrationExclude), "\n") {
		excluded[strings.TrimSpace(line)] = true
	}

	examplePackages, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}

	// Run all examples for 10 seconds each.
	//
	// TODO(diamondburned): find a way to detect that the bot is online. Maybe
	// force all examples to print the current username?
	const exampleRunDuration = 60 * time.Second

	buildDir, err := os.MkdirTemp("", "arikawa-examples")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(buildDir); err != nil {
			t.Log("cannot remove artifacts dir:", err)
		}
	})

	for _, pkg := range examplePackages {
		if !pkg.IsDir() {
			continue
		}

		// Assert package main.
		if _, err := os.Stat(pkg.Name() + "/main.go"); err != nil {
			continue
		}

		pkg := pkg
		t.Run(pkg.Name(), func(t *testing.T) {
			t.Parallel()

			binPath := buildDir + "/" + pkg.Name()

			gobuild := exec.Command("go", "build", "-o", binPath, "./"+pkg.Name())
			gobuild.Stderr = &lineLogger{dst: func(line string) { t.Log("go build:", line) }}
			if err := gobuild.Run(); err != nil {
				t.Fatal("cannot go build:", err)
			}

			if excluded[pkg.Name()] {
				t.Skip("skipping excluded example", pkg.Name())
			}

			timer := time.NewTimer(exampleRunDuration)
			t.Cleanup(func() { timer.Stop() })

			bin := exec.Command(binPath)
			bin.Stderr = &lineLogger{dst: func(line string) { t.Log(pkg.Name()+":", line) }}
			if err := bin.Start(); err != nil {
				t.Fatal("cannot start binary:", err)
			}

			cmdDone := make(chan struct{})
			go func() {
				defer close(cmdDone)

				err := bin.Wait()
				if err == nil {
					return // all good
				}

				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) || !exitErr.Exited() {
					return
				}

				t.Error("binary exited with status", exitErr.ExitCode())
			}()

			select {
			case <-cmdDone:
				return
			case <-timer.C:
			}

			// Works well. Just exit.
			if err := bin.Process.Signal(os.Interrupt); err != nil {
				t.Log("cannot interrupt binary:", err)
				bin.Process.Kill()
			}

			exitTimer := time.NewTimer(5 * time.Second)
			t.Cleanup(func() { exitTimer.Stop() })

			select {
			case <-cmdDone:
				return
			case <-exitTimer.C:
				t.Error("example did not exit after 5 seconds")
				bin.Process.Kill()
			}
		})
	}
}

type lineLogger struct {
	dst func(string)
	buf bytes.Buffer
}

func (l *lineLogger) Write(p []byte) (n int, err error) {
	n, _ = l.buf.Write(p)
	for {
		line, err := l.buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return n, err
		}
		line = line[:len(line)-1] // remove newline
		l.dst(line)
	}
	return n, nil
}
