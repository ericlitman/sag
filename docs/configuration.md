---
title: Configuration
description: "API keys, default voices, timeouts, base URL, and player selection — everything sag reads from the environment."
---

# Configuration

`sag` reads configuration from CLI flags first, then environment variables. There is no config file: the binary stays single-purpose and friendly to ephemeral CI runners.

## API key

Required for any TTS or voice call. `sag --help`, `sag prompting`, and `sag --version` work without one.

| Flag / variable | Notes |
| --- | --- |
| `--api-key` | Inline override. Avoid in shell history; prefer env or `--api-key-file`. |
| `ELEVENLABS_API_KEY` | Primary env var for the default ElevenLabs provider. |
| `FISH_AUDIO_API_KEY` | Primary env var for `--provider fish`. |
| `FISH_API_KEY` | Fish Audio compatibility alias. |
| `SAG_API_KEY` | Provider-neutral fallback alias. |
| `--api-key-file <path>` | Read the key from a file. |
| `ELEVENLABS_API_KEY_FILE` | ElevenLabs key file. |
| `FISH_AUDIO_API_KEY_FILE` | Fish Audio key file. |
| `FISH_API_KEY_FILE` | Fish Audio compatibility key-file alias. |
| `SAG_API_KEY_FILE` | Provider-neutral key-file fallback. |

The file form is handy for agents and containers:

```bash
echo "$ELEVENLABS_API_KEY" > ~/.config/elevenlabs.key
chmod 600 ~/.config/elevenlabs.key
SAG_API_KEY_FILE=~/.config/elevenlabs.key sag voices --limit 3
```

## Provider

ElevenLabs is the default provider. Fish Audio can be selected per command or per shell:

```bash
sag --provider fish -v Sarah "Fish Audio speaking."
export SAG_PROVIDER=fish
```

Provider-specific defaults:

| Provider | Default base URL | Default model | Primary key env |
| --- | --- | --- | --- |
| `elevenlabs` | `https://api.elevenlabs.io` | `eleven_v3` | `ELEVENLABS_API_KEY` |
| `fish` | `https://api.fish.audio` | `s2-pro` | `FISH_AUDIO_API_KEY` |

## Default voice

When `--voice` / `--voice-id` is omitted, `sag` resolves in this order:

1. Provider-specific voice env: `ELEVENLABS_VOICE_ID` or `FISH_AUDIO_VOICE_ID`.
2. `SAG_VOICE_ID`
3. The first provider voice returned by the API (logged on stderr so you notice).

```bash
export SAG_VOICE_ID=21m00Tcm4TlvDq8ikWAM
sag "Default voice locked in."
```

Pass `?` to force the voice list and exit:

```bash
sag -v ?
```

## Timeouts

`sag` ships with **no internal timeout** so that long v3 prompts don’t get truncated by a hidden SIGTERM. Decide for yourself:

| Source | Behaviour |
| --- | --- |
| `--timeout 5m` flag | Cancels the TTS request after the given Go duration. `0` keeps the parent context. |
| `SAG_TIMEOUT=5m` env | Same effect, set per shell or per CI job. |
| Outer process timeout | Use `timeout(1)` or your scheduler if you want a hard kill. |

The flag wins over the environment variable; both accept any `time.ParseDuration` string (`30s`, `2m`, `1h30m`).

```bash
SAG_TIMEOUT=10m sag --no-play -o long.mp3 "$(<chapter.txt)"
```

When sag is the bottleneck and the shell aborts the request, you’ll get a partial file. Use `ffprobe` to sanity-check duration before publishing.

## Player backend

| Value | Behaviour |
| --- | --- |
| `auto` (default) | `afplay` on macOS, `oto` everywhere else. |
| `afplay` | macOS only; routes through CoreAudio so AirPlay and Bluetooth zones work. |
| `oto` | Cross-platform Go decoder (`go-mp3` + `oto`). |

Pick a backend explicitly via `--player oto` or `SAG_PLAYER=oto`. See [Streaming & playback](streaming.md) for trade-offs.

## API base URL

Override the provider endpoint when you’re routing through a proxy or talking to a regional/staging deployment:

```bash
sag --base-url https://api.elevenlabs.io "Default."
sag --provider fish --base-url https://api.fish.audio "Default."
sag --base-url https://your-proxy.internal "Routed."
```

Provider defaults are shown above. You can also set `ELEVENLABS_BASE_URL` or `FISH_AUDIO_BASE_URL` for a shell, but the flag wins.

## Voice metadata cache

`sag voices --query` and `--label` need full voice descriptors. Metadata is cached in your platform-default config directory (`$XDG_CONFIG_HOME/sag/voices.json` on Linux, `~/Library/Application Support/sag/voices.json` on macOS) for 24 hours. Delete the file or pass `--limit 0` after a voice update to force a refresh.

## Compatibility flags (no-ops)

These are accepted for `say` parity and silently ignored:

`--progress`, `--audio-device`, `--network-send`, `--interactive`, `--file-format`, `--data-format`, `--channels`, `--bit-rate`, `--quality`.

If you rely on these in a script, sag won’t error. They simply have no effect because the synthesis happens server-side.

## Putting it together

A typical agent profile looks like this:

```bash
export ELEVENLABS_API_KEY_FILE=~/.config/elevenlabs.key
export SAG_VOICE_ID=21m00Tcm4TlvDq8ikWAM
export SAG_TIMEOUT=5m
export SAG_PLAYER=oto

sag --no-play -o "$artifact" "$prompt"
```

For Fish Audio:

```bash
export SAG_PROVIDER=fish
export FISH_AUDIO_API_KEY_FILE=~/.config/fish-audio.key
export FISH_AUDIO_VOICE_ID=<voice-model-id>
export SAG_TIMEOUT=2m

sag --no-play -o "$artifact" "$prompt"
```

## Related pages

- [Quickstart](quickstart.md) — the minimal setup walkthrough.
- [Streaming & playback](streaming.md) — when to use which backend.
- [Output & formats](formats.md) — picking a codec and format string.
- [Models](models.md) — model-specific pricing and latency.
