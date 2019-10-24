package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
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
	)

	pronounce := &ffcli.Command{
		Name:     "pronounce",
		Usage:    "pronounce [-v VOICE] PRONOUNCE",
		FlagSet:  pronounceFlags,
		LongHelp: fmt.Sprintf("VOICES\n  %s", strings.Join(append(voiceList(pronounceBox), "RANDOM"), "\n  ")),
		Exec: func(args []string) error {
			if len(args) < 1 {
				return flag.ErrHelp
			}

			var (
				parts         = voiceParts(pronounceBox, *pronounceVoice)
				tosay         = []rune(strings.Join(args, " "))
				selectedParts = []string{}
				selectedFiles = []string{}
			)

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
					log.Printf("unknown part: %q", tosay[i])
					i++ // skip unmatched parts
				}
			}

			for _, part := range selectedParts {
				randomFile := parts[part][rand.Intn(len(parts[part]))]
				selectedFiles = append(selectedFiles, fmt.Sprintf("./pronounce/%s", randomFile))
			}

			cmdArgs := append(selectedFiles, "out.mp3")
			log.Printf("+ sox %s", strings.Join(cmdArgs, " "))
			cmd := exec.Command("sox", cmdArgs...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return err
			}

			log.Printf("+ afplay out.mp3")
			cmd = exec.Command("afplay", "out.mp3")
			if err := cmd.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	say := &ffcli.Command{
		Name:     "say",
		Usage:    "say [-v VOICE] SAY",
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

	root := &ffcli.Command{
		Usage:       "speechotron <subcommand> [flags] [args...]",
		Subcommands: []*ffcli.Command{pronounce, say},
		Exec:        func([]string) error { return flag.ErrHelp },
	}

	if err := root.Run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		log.Fatalf("fatal: %+v", err)
	}
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
