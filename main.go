package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"

	packr "github.com/gobuffalo/packr/v2"
	"github.com/peterbourgon/ff/ffcli"
)

var (
	sayBox       = packr.New("say", "./say")
	pronounceBox = packr.New("pronounce", "./pronounce")
)

func main() {
	log.SetFlags(0)

	var (
		sayFlags = flag.NewFlagSet("say", flag.ExitOnError)
		sayVoice = sayFlags.String("v", "RANDOM", "voice")

		pronounceFlags = flag.NewFlagSet("pronounce", flag.ExitOnError)
		pronounceVoice = pronounceFlags.String("v", "RANDOM", "voice")

		serverFlags = flag.NewFlagSet("server", flag.ExitOnError)
		serverBind  = serverFlags.String("b", ":8000", "bind address")
	)

	pronounce := &ffcli.Command{
		Name:     "pronounce",
		Usage:    "pronounce [-v VOICE] WORDS...",
		FlagSet:  pronounceFlags,
		LongHelp: fmt.Sprintf("VOICES\n  %s", strings.Join(append(voiceList(pronounceBox), "RANDOM"), "\n  ")),
		Exec: func(args []string) error {
			if len(args) < 1 {
				return flag.ErrHelp
			}

			tosay := strings.Join(args, " ")

			err := pronounceToFile(*pronounceVoice, tosay, "out.mp3")
			if err != nil {
				return fmt.Errorf("pronounce to file: %w", err)
			}

			return playFile("out.mp3")
		},
	}

	say := &ffcli.Command{
		Name:     "say",
		Usage:    "say [-v VOICE] WORDS...",
		FlagSet:  sayFlags,
		LongHelp: fmt.Sprintf("VOICES\n  %s", strings.Join(append(voiceList(sayBox), "RANDOM"), "\n  ")),
		Exec: func(args []string) error {
			if len(args) < 1 {
				return flag.ErrHelp
			}
			_ = sayVoice
			return fmt.Errorf("not implemented")
		},
	}

	server := &ffcli.Command{
		Name:    "server",
		Usage:   "server [OPTS]",
		FlagSet: serverFlags,
		Exec: func(args []string) error {
			pronounceHandler := func(w http.ResponseWriter, req *http.Request) {
				var (
					text     string
					voice    = "RANDOM"
					filename = "out.mp3" // FIXME: caching with hash
				)

				if query := req.URL.Query()["text"]; len(query) > 0 {
					text = query[0]
				}
				if query := req.URL.Query()["voice"]; len(query) > 0 {
					voice = query[0]
				}
				if text == "" {
					http.Error(w, "invalid input", 500)
					return
				}

				if err := pronounceToFile(voice, text, filename); err != nil {
					http.Error(w, fmt.Sprintf("failed to generate: %v", err), 500)
					return
				}

				content, err := ioutil.ReadFile(filename)
				if err != nil {
					http.Error(w, fmt.Sprintf("failed to read file: %w", err), 500)
					return
				}
				b := bytes.NewBuffer(content)

				w.Header().Set("Content-Type", "audio/mpeg")

				_, err = b.WriteTo(w)
				if err != nil {
					http.Error(w, fmt.Sprintf("failed to write to stream: %v", err), 500)
					return
				}

				log.Printf("mp3 sent, voice=%q, text=%q", voice, text)
			}
			http.HandleFunc("/api/pronounce", pronounceHandler)
			log.Printf("Starting server on %q", *serverBind)
			return http.ListenAndServe(*serverBind, nil)
		},
	}

	root := &ffcli.Command{
		Usage:       "speechotron <subcommand> [flags] [args...]",
		Subcommands: []*ffcli.Command{pronounce, say, server},
		Exec:        func([]string) error { return flag.ErrHelp },
	}

	if err := root.Run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		log.Fatalf("fatal: %+v", err)
	}
}

func playFile(dest string) error {
	log.Printf("+ afplay %s", dest)
	cmd := exec.Command("afplay", dest)
	return cmd.Run()
}

func pronounceToFile(voice string, text string, dest string) error {
	tosay := []rune(text)

	parts := voiceParts(pronounceBox, voice)
	selectedParts := []string{}

	for i := 0; i < len(tosay); {
		maxLen := 0
		selectedPart := ""
		for partString := range parts {
			part := []rune(partString)
			if len(part) <= maxLen {
				continue
			}
			if string(part) == string(tosay[i:i+len(part)]) {
				maxLen = len(part)
				selectedPart = string(part)
			}
		}
		if selectedPart != "" {
			i += maxLen
			selectedParts = append(selectedParts, selectedPart)
		} else {
			i++ // skip unmatched parts
		}
	}

	selectedFiles := []string{}

	for _, part := range selectedParts {
		randomFile := parts[part][rand.Intn(len(parts[part]))]
		selectedFiles = append(selectedFiles, fmt.Sprintf("./pronounce/%s", randomFile))
	}

	cmdArgs := append(selectedFiles, dest)
	log.Printf("+ sox %s", strings.Join(cmdArgs, " "))
	cmd := exec.Command("sox", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func voiceList(box *packr.Box) []string {
	voices := map[string]bool{}
	for _, voice := range box.List() {
		if !strings.HasSuffix(voice, ".wav") {
			continue
		}
		voices[strings.Split(voice, "/")[0]] = true
	}

	ret := []string{}
	for voice := range voices {
		ret = append(ret, voice)
	}
	sort.Strings(ret)

	return ret
}

func voiceParts(box *packr.Box, voice string) map[string][]string {
	ret := map[string][]string{}

	for _, file := range box.List() {
		if !strings.HasSuffix(file, ".wav") {
			continue
		}

		spl := strings.Split(file, "/")
		actualVoice := spl[0]
		if voice != "RANDOM" && voice != actualVoice {
			continue
		}

		part := strings.Replace(spl[1], ".wav", "", -1)

		if _, ok := ret[part]; !ok {
			ret[part] = []string{}
		}
		ret[part] = append(ret[part], file)
	}

	return ret
}
