package cli

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/stefafafan/jev/internal/evaluation"
	"github.com/stefafafan/jev/internal/provider"
)

type ProviderFactory func(provider.Config) (provider.Provider, error)

type App struct {
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	Getenv      provider.LookupEnv
	NewProvider ProviderFactory
	Version     string
}

func (a App) Run(ctx context.Context, args []string) int {
	options, err := parseArgs(args)
	if err != nil {
		return a.fail(2, err)
	}
	if options.helpText != "" {
		if _, err := io.WriteString(a.Stdout, options.helpText); err != nil {
			return a.fail(1, fmt.Errorf("write help: %w", err))
		}
		return 0
	}
	if options.version {
		version := a.Version
		if version == "" {
			version = "dev"
		}
		if _, err := fmt.Fprintf(a.Stdout, "jev %s\n", version); err != nil {
			return a.fail(1, fmt.Errorf("write version: %w", err))
		}
		return 0
	}
	if err := validateExplicitOptions(options); err != nil {
		return a.fail(2, err)
	}
	format, err := parseOutputFormat(options.output)
	if err != nil {
		return a.fail(2, err)
	}
	questions, err := options.questions()
	if err != nil {
		return a.fail(2, err)
	}
	state, err := a.state()
	if err != nil {
		return a.fail(2, err)
	}

	config, err := provider.Resolve(options.provider, a.Getenv)
	if err != nil {
		return a.fail(1, err)
	}
	selected, err := a.NewProvider(config)
	if err != nil {
		return a.fail(1, fmt.Errorf("create provider: %w", err))
	}
	if selected == nil {
		return a.fail(1, fmt.Errorf("create provider: provider is nil"))
	}
	response, err := selected.Evaluate(ctx, evaluation.Request{State: state, Questions: questions})
	if err != nil {
		return a.fail(1, err)
	}
	var rendered bytes.Buffer
	if err := writeOutput(&rendered, format, response); err != nil {
		return a.fail(1, err)
	}
	if _, err := io.Copy(a.Stdout, &rendered); err != nil {
		return a.fail(1, fmt.Errorf("write output: %w", err))
	}
	return 0
}

type options struct {
	provider    string
	providerSet bool
	output      string
	command     string
	question    string
	values      []string
	helpText    string
	version     bool
}

func parseArgs(args []string) (options, error) {
	var options options
	flags := flag.NewFlagSet("jev", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&options.provider, "provider", "", "provider: typesafe, cloudflare, or vercel")
	flags.StringVar(&options.output, "output", "json", "output format: json or text")
	var help bool
	flags.BoolVar(&help, "help", false, "show help")
	flags.BoolVar(&help, "h", false, "show help")
	flags.BoolVar(&options.version, "version", false, "show version")
	if err := flags.Parse(args); err != nil {
		return options, fmt.Errorf("parse arguments: %w", err)
	}
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "provider" {
			options.providerSet = true
		}
	})
	if help {
		options.helpText = usage
		return options, nil
	}
	if options.version {
		return options, nil
	}

	remaining := flags.Args()
	if len(remaining) == 0 {
		return options, fmt.Errorf("command is required; expected noul, choice, or score")
	}
	options.command = remaining[0]
	if err := parseCommand(&options, remaining[1:]); err != nil {
		return options, err
	}
	return options, nil
}

func parseCommand(options *options, args []string) error {
	flags := flag.NewFlagSet(options.command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var help bool
	flags.BoolVar(&help, "help", false, "show help")
	flags.BoolVar(&help, "h", false, "show help")

	switch options.command {
	case "noul":
		if err := flags.Parse(args); err != nil {
			return fmt.Errorf("parse noul arguments: %w", err)
		}
		if help {
			options.helpText = noulUsage
			return nil
		}
	case "choice":
		flags.Var((*stringList)(&options.values), "option", "choice option; repeat between 2 and 255 times")
		if err := flags.Parse(args); err != nil {
			return fmt.Errorf("parse choice arguments: %w", err)
		}
		if help {
			options.helpText = choiceUsage
			return nil
		}
	case "score":
		flags.Var((*stringList)(&options.values), "level", "ordered score level; repeat between 2 and 10 times")
		if err := flags.Parse(args); err != nil {
			return fmt.Errorf("parse score arguments: %w", err)
		}
		if help {
			options.helpText = scoreUsage
			return nil
		}
	default:
		return fmt.Errorf("unsupported command %q; expected noul, choice, or score", options.command)
	}

	if flags.NArg() != 1 {
		return fmt.Errorf("%s requires exactly one question argument", options.command)
	}
	options.question = flags.Arg(0)
	return nil
}

func validateExplicitOptions(options options) error {
	if options.providerSet {
		if options.provider == "" {
			return fmt.Errorf("--provider must not be empty")
		}
		if options.provider != "typesafe" && options.provider != "cloudflare" && options.provider != "vercel" {
			return fmt.Errorf("unsupported --provider %q; expected typesafe, cloudflare, or vercel", options.provider)
		}
	}
	return nil
}

func (o options) questions() (map[string]evaluation.Question, error) {
	switch o.command {
	case "noul":
		return evaluation.NewInlineNoul(o.question)
	case "choice":
		return evaluation.NewInlineChoice(o.values, o.question)
	case "score":
		return evaluation.NewInlineScore(o.values, o.question)
	default:
		return nil, fmt.Errorf("unsupported command %q", o.command)
	}
}

type stringList []string

func (values *stringList) Set(value string) error {
	*values = append(*values, value)
	return nil
}

func (values *stringList) String() string { return strings.Join(*values, ",") }

func (a App) state() (string, error) {
	data, err := io.ReadAll(a.Stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("state must not be empty")
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("state must be valid UTF-8")
	}
	return string(data), nil
}

func (a App) fail(code int, err error) int {
	message := strings.TrimSpace(err.Error())
	_, _ = fmt.Fprintf(a.Stderr, "jev: %s\n", message)
	return code
}

const usage = `Usage: jev [global options] COMMAND [command options] QUESTION

Commands:
  noul      ask a yes/no probability question
  choice    choose among repeated --option values
  score     score against repeated ordered --level values

Global options:
  --provider NAME  provider: typesafe, cloudflare, or vercel
  --output FORMAT  output format: json (default) or text
  --version        show version
  -h, --help       show help

Run "jev COMMAND --help" for command-specific help.
`

const noulUsage = `Usage: jev [global options] noul QUESTION
`

const choiceUsage = `Usage: jev [global options] choice --option LABEL --option LABEL [--option LABEL...] QUESTION

Options:
  --option LABEL  choice option; repeat between 2 and 255 times
`

const scoreUsage = `Usage: jev [global options] score --level LEVEL --level LEVEL [--level LEVEL...] QUESTION

Options:
  --level LEVEL  ordered score level; repeat between 2 and 10 times
`
