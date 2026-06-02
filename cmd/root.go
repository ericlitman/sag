package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type rootConfig struct {
	APIKey     string
	APIKeyFile string
	BaseURL    string
	Provider   string
}

var (
	cfg         rootConfig
	versionFlag bool
	rootCmd     = &cobra.Command{
		Use:     "sag",
		Short:   "🗣️ Speech, mac-style ease",
		Long:    "Command-line TTS with macOS playback. ElevenLabs is the default provider, and Fish Audio is available via --provider fish. Call it like macOS 'say': if you skip the subcommand, text args are passed to 'speak' (e.g. `sag \"Hello\"`).\n\nTip: run `sag prompting` for model-specific prompting tips.\nModels: ElevenLabs defaults to `eleven_v3`; Fish Audio defaults to `s2-pro`.",
		Example: "  sag \"Hi Peter\"\n  echo 'piped input' | sag\n  sag speak -v Roger --rate 200 \"Faster speech\"\n  sag prompting",
		Version: "0.3.0",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if versionFlag {
				fmt.Println(cmd.Root().Name(), cmd.Root().Version)
				os.Exit(0)
			}
			return applyProviderEnv(cmd)
		},
	}
)

// Execute is the entry point from main.
func Execute() {
	maybeDefaultToSpeak()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cfg.Provider = providerElevenLabs
	rootCmd.PersistentFlags().StringVar(&cfg.Provider, "provider", cfg.Provider, "TTS provider: elevenlabs or fish (SAG_PROVIDER)")
	rootCmd.PersistentFlags().StringVar(&cfg.APIKey, "api-key", "", "Provider API key (or ELEVENLABS_API_KEY/FISH_AUDIO_API_KEY)")
	rootCmd.PersistentFlags().StringVar(&cfg.APIKeyFile, "api-key-file", "", "Read provider API key from file")
	rootCmd.PersistentFlags().StringVar(&cfg.BaseURL, "base-url", "", "Override provider API base URL")
	rootCmd.PersistentFlags().BoolVarP(&versionFlag, "version", "V", false, "Print version and exit")
}

// maybeDefaultToSpeak injects the "speak" subcommand when the user calls `sag` like macOS `say`.
func maybeDefaultToSpeak() {
	if len(os.Args) <= 1 {
		// Still default to speak if stdin has piped data
		if !isStdinTTY() {
			os.Args = append(os.Args, "speak")
		}
		return
	}

	// npm/pnpm pass-through typically prefixes args with "--"; drop it so flags still parse.
	if os.Args[1] == "--" {
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
		if len(os.Args) <= 1 {
			return
		}
	}

	args := os.Args[1:]
	insertAt := firstNonRootFlagIndex(args)
	if insertAt >= len(args) {
		if !isStdinTTY() {
			os.Args = append(os.Args, "speak")
		}
		return
	}
	first := args[insertAt]
	if isKnownSubcommand(first) || isCobraBuiltin(first) || first == "-h" || first == "--help" {
		return
	}
	withSpeak := make([]string, 0, len(args)+2)
	withSpeak = append(withSpeak, os.Args[0])
	withSpeak = append(withSpeak, args[:insertAt]...)
	withSpeak = append(withSpeak, "speak")
	withSpeak = append(withSpeak, args[insertAt:]...)
	os.Args = withSpeak
}

func firstNonRootFlagIndex(args []string) int {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-V" || arg == "--version" || arg == "-h" || arg == "--help":
			return len(args)
		case strings.HasPrefix(arg, "--provider=") || strings.HasPrefix(arg, "--api-key=") || strings.HasPrefix(arg, "--api-key-file=") || strings.HasPrefix(arg, "--base-url="):
			continue
		case arg == "--provider" || arg == "--api-key" || arg == "--api-key-file" || arg == "--base-url":
			i++
			continue
		default:
			return i
		}
	}
	return len(args)
}

func isCobraBuiltin(name string) bool {
	name = strings.ToLower(name)
	return name == "help" || name == "completion"
}

func isKnownSubcommand(name string) bool {
	name = strings.ToLower(name)
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == name {
			return true
		}
		for _, a := range cmd.Aliases {
			if a == name {
				return true
			}
		}
	}
	return false
}
