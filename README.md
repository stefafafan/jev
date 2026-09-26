# stefafafan/jev

An unofficial, provider-neutral Unix client for [Jev](https://typesafe.ai/blog/introducing-system-one-models-and-jev) from [TypeSafe AI](https://typesafe.ai/). Written in Go.

Currently supported providers:
- TypeSafe AI
- Cloudflare
- Vercel

## Install

Build from this repository:

```bash
go build -o jev .
```

Or install the command from the module after a release is published:

```bash
go install github.com/stefafafan/jev@latest
```

## Quick Start

Set credentials for one provider:

```bash
export TYPESAFE_API_KEY=...
```

Then pipe state into one question:

```bash
echo "some state" | jev --provider typesafe noul "Is this safe?"
```

JSON is written by default:

```json
{"provider":"typesafe","model":"jev-1.13.0","answers":{"result":{"type":"noul","noul":0.95}},"usage":{"input_tokens":392,"output_tokens":21}}
```

The CLI reports whether evaluation succeeded through its exit code. It does not turn a judgment into policy. Put thresholds downstream:

```bash
git diff | jev noul "Could this change introduce a regression?" | jq -e '.answers.result.noul < 0.2'
```

## Questions

Each invocation asks exactly one question through a primitive subcommand.

### Noul

```bash
cat incident.txt | jev noul "Does this require immediate action?"
```

### Choice

Repeat `--option` for each candidate. Commas are ordinary label characters.

```bash
git diff |
  jev choice \
    --option safe \
    --option needs-review \
    --option unsafe \
    "How should this change be classified?"
```

### Score

Score levels are ordered from low to high.

```bash
kubectl diff -f manifest.yaml |
  jev score \
    --level low \
    --level medium \
    --level high \
    "How risky is this deployment?"
```

## State

State always comes from stdin. Use ordinary shell redirection for a file:

```bash
jev noul "Is this urgent?" < incident.txt
```

## Providers

The CLI supports TypeSafe AI, Cloudflare Workers AI, and Vercel AI Gateway.

### TypeSafe AI

```bash
export TYPESAFE_API_KEY=...
jev --provider typesafe noul "Is this safe?" < input.txt
```

### Cloudflare

```bash
export CLOUDFLARE_API_TOKEN=...
export CLOUDFLARE_ACCOUNT_ID=...
cat input.txt | jev --provider cloudflare noul "Is this safe?"
```

### Vercel

```bash
export AI_GATEWAY_API_KEY=...
cat input.txt | jev --provider vercel noul "Is this safe?"
```

## Output

JSON is the default machine-readable format.

Each answer retains the Jev primitive fields:

- Noul: `type`, `noul`
- Choice: `type`, `choice`, `confidence`, `probabilities`
- Score: `type`, `score`, `confidence`, `legend`, `probabilities`

Use `--output text` option as needed.

```bash
cat input.txt | jev --output text noul "Is this safe?"
```

stdout contains only a successful result. Diagnostics are written to stderr.

## Options

```text
jev [global options] noul QUESTION
jev [global options] choice --option LABEL --option LABEL [...] QUESTION
jev [global options] score --level LEVEL --level LEVEL [...] QUESTION

--provider NAME  provider: typesafe, cloudflare, or vercel
--output FORMAT  json (default) or text
--version        show version
-h, --help       show help
```
