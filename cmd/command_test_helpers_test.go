package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func isolateRootCommand(t *testing.T) {
	t.Helper()
	resetRootCommand(t)
	t.Cleanup(func() {
		resetRootCommand(t)
	})
}

func resetRootCommand(t *testing.T) {
	t.Helper()

	cfg = rootConfig{Provider: providerElevenLabs}
	versionFlag = false
	rootCmd.SetArgs(nil)
	rootCmd.SetOut(nil)
	rootCmd.SetErr(nil)
	rootCmd.SetIn(nil)

	resetFlags(t, rootCmd.PersistentFlags(), map[string]string{
		"provider":     providerElevenLabs,
		"api-key":      "",
		"api-key-file": "",
		"base-url":     "",
		"version":      "false",
	})

	if cmd := commandByName("speak"); cmd != nil {
		resetFlags(t, cmd.Flags(), map[string]string{
			"voice-id":         "",
			"voice":            "",
			"model-id":         "eleven_v3",
			"output":           "",
			"format":           "mp3_44100_128",
			"stream":           "true",
			"no-stream":        "false",
			"play":             "true",
			"no-play":          "false",
			"latency-tier":     "0",
			"speed":            "1",
			"rate":             "0",
			"stability":        "0",
			"similarity":       "0",
			"similarity-boost": "0",
			"style":            "0",
			"speaker-boost":    "false",
			"no-speaker-boost": "false",
			"seed":             "0",
			"normalize":        "",
			"lang":             "",
			"metrics":          "false",
			"timeout":          "0s",
			"player":           "auto",
			"input-file":       "",
			"progress":         "false",
			"network-send":     "",
			"audio-device":     "",
			"interactive":      "",
			"file-format":      "",
			"data-format":      "",
			"channels":         "0",
			"bit-rate":         "0",
			"quality":          "0",
		})
	}

	if cmd := commandByName("voices"); cmd != nil {
		resetFlags(t, cmd.Flags(), map[string]string{
			"search": "",
			"query":  "",
			"limit":  "100",
			"try":    "false",
		})
		resetStringArrayFlag(t, cmd.Flags(), "label")
	}
}

func commandByName(name string) *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}

func resetFlags(t *testing.T, flags *pflag.FlagSet, values map[string]string) {
	t.Helper()
	for name, value := range values {
		flag := flags.Lookup(name)
		if flag == nil {
			t.Fatalf("missing %s flag", name)
		}
		if err := flag.Value.Set(value); err != nil {
			t.Fatalf("reset %s flag: %v", name, err)
		}
		flag.Changed = false
	}
}

func resetStringArrayFlag(t *testing.T, flags *pflag.FlagSet, name string) {
	t.Helper()
	flag := flags.Lookup(name)
	if flag == nil {
		t.Fatalf("missing %s flag", name)
	}
	if sa, ok := flag.Value.(interface{ Replace([]string) error }); ok {
		if err := sa.Replace(nil); err != nil {
			t.Fatalf("reset %s flag: %v", name, err)
		}
		flag.Changed = false
		return
	}
	if err := flag.Value.Set(""); err != nil {
		t.Fatalf("reset %s flag: %v", name, err)
	}
	flag.Changed = false
}
