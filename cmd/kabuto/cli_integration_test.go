package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kzcat/kabuto/internal/fetcher"
	"github.com/kzcat/kabuto/internal/render"
)

const cliFixture = `{"chart":{"result":[{"meta":{"regularMarketPrice":100,"chartPreviousClose":90,"regularMarketTime":1788739200},"indicators":{"quote":[{"close":[90,95,100]}]}}]}}`

// Run the real CLI in a subprocess so flag exits and terminal state are isolated.
func TestCLIProcess(t *testing.T) {
	if os.Getenv("KABUTO_TEST_PROCESS") != "1" {
		return
	}
	start := 0
	for i, a := range os.Args {
		if a == "--" {
			start = i + 1
			break
		}
	}
	if start == 0 {
		os.Exit(2)
	}
	os.Args = append([]string{"kabuto"}, os.Args[start:]...)
	flag.CommandLine = flag.NewFlagSet("kabuto", flag.ExitOnError)
	fetcher.BaseURLOverride = []string{os.Getenv("KABUTO_TEST_URL") + "/%s"}
	main()
	os.Exit(0)
}

func cliCommand(t *testing.T, url string, args ...string) *exec.Cmd {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	t.Cleanup(cancel)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, exe, append([]string{"-test.run=^TestCLIProcess$", "--"}, args...)...)
	cmd.Env = append(os.Environ(), "KABUTO_TEST_PROCESS=1", "KABUTO_TEST_URL="+url, "GORACE=atexit_sleep_ms=0", "COLUMNS=80", "LINES=24")
	return cmd
}

func TestCLISelectsRegisteredSections(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, cliFixture) }))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"sections":[{"key":"custom","title":"Custom Stocks","items":[{"symbol":"EXAMPLE","decimals":2}]}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		section string
		extra   []string
	}{
		{"custom", nil},
		{"watch", []string{"--add", "AAPL"}},
	} {
		t.Run(tc.section, func(t *testing.T) {
			args := []string{"--config", path, "-j", "-s", tc.section}
			args = append(args, tc.extra...)
			output, err := cliCommand(t, server.URL, args...).CombinedOutput()
			if err != nil {
				t.Fatalf("CLI failed: %v\n%s", err, output)
			}
			var out map[string]render.JSONSection
			if err := json.Unmarshal(output, &out); err != nil {
				t.Fatalf("invalid JSON: %v\n%s", err, output)
			}
			section, ok := out[tc.section]
			if !ok || len(out) != 1 || len(section.Items) != 1 || section.Items[0].Price == nil || *section.Items[0].Price != 100 {
				t.Fatalf("wrong selected data: %s", output)
			}
			if tc.section == "custom" && section.Title != "Custom Stocks" {
				t.Fatalf("custom title lost: %s", output)
			}
		})
	}
	output, err := cliCommand(t, server.URL, "--config", path, "-s", "missing").CombinedOutput()
	if err == nil || !strings.Contains(string(output), "invalid section: missing") {
		t.Fatalf("unknown section accepted: %v %s", err, output)
	}
}

func TestCLIWatchCanQuitDuringInitialFetch(t *testing.T) {
	for _, key := range []string{"q", "\x03"} {
		t.Run(fmt.Sprintf("key_%x", key), func(t *testing.T) {
			started := make(chan struct{}, 10)
			canceled := make(chan struct{}, 10)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				started <- struct{}{}
				<-r.Context().Done()
				canceled <- struct{}{}
			}))
			defer server.Close()
			cmd := cliCommand(t, server.URL, "--config", filepath.Join(t.TempDir(), "absent.json"), "--source", "yahoo", "-s", "japan", "-w", "60")
			input, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			cmd.Stdout = &output
			cmd.Stderr = &output
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				input.Close()
				if cmd.ProcessState == nil {
					cmd.Process.Kill()
				}
			})
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				cmd.Process.Kill()
				cmd.Wait()
				t.Fatal("initial fetch never started")
			}
			if _, err := input.Write([]byte(key)); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("quit failed: %v %s", err, output.String())
				}
			case <-time.After(3 * time.Second):
				cmd.Process.Kill()
				<-done
				t.Fatal("quit blocked behind initial network fetch")
			}
			if !strings.Contains(output.String(), "Loading...") || !strings.HasSuffix(output.String(), showCur+leaveAlt) {
				t.Fatalf("missing loading/restoration: %q", output.String())
			}
			select {
			case <-canceled:
			case <-time.After(3 * time.Second):
				t.Fatal("HTTP request was not canceled")
			}
		})
	}
}

func TestCLIWatchRangeChangeCancelsPendingFetch(t *testing.T) {
	started := make(chan struct{}, 20)
	canceled := make(chan struct{}, 20)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		<-r.Context().Done()
		canceled <- struct{}{}
	}))
	defer server.Close()
	cmd := cliCommand(t, server.URL, "--config", filepath.Join(t.TempDir(), "absent.json"), "--source", "yahoo", "-s", "japan", "-w", "60")
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// Kill the process before server.Close on any failed assertion.
	defer func() {
		input.Close()
		if cmd.ProcessState == nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()
	awaitBatch := func(ch <-chan struct{}, what string) {
		t.Helper()
		for i := 0; i < 5; i++ {
			select {
			case <-ch:
			case <-time.After(3 * time.Second):
				t.Fatalf("%s did not complete", what)
			}
		}
	}
	awaitBatch(started, "initial batch")
	if _, err := input.Write([]byte("]")); err != nil {
		t.Fatal(err)
	}
	awaitBatch(canceled, "cancel old range")
	awaitBatch(started, "new range fetch without waiting for refresh interval")
	if _, err := input.Write([]byte("q")); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("quit failed: %v %s", err, output.String())
	}
	if !strings.Contains(output.String(), "[5d]") {
		t.Fatalf("range label did not change: %s", output.String())
	}
}
