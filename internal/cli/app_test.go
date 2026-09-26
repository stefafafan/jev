package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stefafafan/jev/internal/evaluation"
	"github.com/stefafafan/jev/internal/provider"
)

func TestHelpAndVersionNeedNoProvider(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{"help", []string{"--help"}, "Usage: jev"},
		{"short help", []string{"-h"}, "Usage: jev"},
		{"choice help", []string{"choice", "--help"}, "Usage: jev [global options] choice"},
		{"version", []string{"--version"}, "jev test-version\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, stdout, stderr, calls := testApp()
			code := app.Run(context.Background(), tt.args)
			if code != 0 || !strings.Contains(stdout.String(), tt.want) || stderr.Len() != 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
			if *calls != 0 {
				t.Fatalf("provider calls = %d", *calls)
			}
		})
	}
}

func TestNoulFromStdinProducesJSON(t *testing.T) {
	app, stdout, stderr, calls := testApp()
	app.Stdin = strings.NewReader("some state")
	code := app.Run(context.Background(), []string{"noul", "Is this safe?"})
	if code != 0 || stderr.Len() != 0 || *calls != 1 {
		t.Fatalf("code=%d stderr=%q calls=%d", code, stderr, *calls)
	}
	var response evaluation.Response
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Provider != "typesafe" || *response.Answers["result"].Noul != 0.75 {
		t.Fatalf("response = %#v", response)
	}
}

func TestChoiceAndScoreInlineArguments(t *testing.T) {
	for _, tt := range []struct {
		name     string
		args     []string
		typeWant evaluation.QuestionType
	}{
		{"choice", []string{"choice", "--option", "safe,automatic", "--option", "review", "Classify this"}, evaluation.Choice},
		{"score", []string{"score", "--level", "low", "--level", "medium", "--level", "high", "Rate this"}, evaluation.Score},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, _, stderr, _ := testApp()
			app.Stdin = strings.NewReader("state")
			var got evaluation.Request
			app.NewProvider = func(provider.Config) (provider.Provider, error) {
				return providerFunc(func(_ context.Context, request evaluation.Request) (evaluation.Response, error) {
					got = request
					return responseFor(request), nil
				}), nil
			}
			if code := app.Run(context.Background(), tt.args); code != 0 {
				t.Fatalf("code=%d stderr=%q", code, stderr)
			}
			if got.Questions["result"].Type != tt.typeWant {
				t.Fatalf("type = %q", got.Questions["result"].Type)
			}
		})
	}
}

func TestVercelProviderIsAccepted(t *testing.T) {
	app, _, stderr, _ := testApp()
	app.Stdin = strings.NewReader("state")
	app.Getenv = func(key string) string {
		if key == "AI_GATEWAY_API_KEY" {
			return "test-token"
		}
		return ""
	}
	var got provider.Config
	app.NewProvider = func(config provider.Config) (provider.Provider, error) {
		got = config
		return providerFunc(func(_ context.Context, request evaluation.Request) (evaluation.Response, error) {
			return responseFor(request), nil
		}), nil
	}
	if code := app.Run(context.Background(), []string{"--provider", "vercel", "noul", "Safe?"}); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	if got.Name != "vercel" || got.VercelAPIKey != "test-token" {
		t.Fatalf("config = %#v", got)
	}
}

func TestJSONLookingStateComesFromStdin(t *testing.T) {
	app, _, stderr, _ := testApp()
	app.Stdin = strings.NewReader(`{"looks":"json"}`)
	var got evaluation.Request
	app.NewProvider = func(provider.Config) (provider.Provider, error) {
		return providerFunc(func(_ context.Context, request evaluation.Request) (evaluation.Response, error) {
			got = request
			return responseFor(request), nil
		}), nil
	}
	code := app.Run(context.Background(), []string{"noul", "Check?"})
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	if got.State != `{"looks":"json"}` {
		t.Fatalf("state = %q", got.State)
	}
}

func TestLocalErrorsExitTwoWithoutProviderCall(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		stdin string
	}{
		{"no command", nil, "state"},
		{"unsupported command", []string{"batch"}, "state"},
		{"noul missing question", []string{"noul"}, "state"},
		{"noul extra argument", []string{"noul", "x", "extra"}, "state"},
		{"choice missing options", []string{"choice", "question"}, "state"},
		{"choice too few options", []string{"choice", "--option", "safe", "question"}, "state"},
		{"choice missing question", []string{"choice", "--option", "safe", "--option", "unsafe"}, "state"},
		{"score too few levels", []string{"score", "--level", "low", "question"}, "state"},
		{"invalid output", []string{"--output", "yaml", "noul", "x"}, "state"},
		{"empty explicit provider", []string{"--provider=", "noul", "x"}, "state"},
		{"invalid explicit provider", []string{"--provider", "bogus", "noul", "x"}, "state"},
		{"empty state", []string{"noul", "x"}, ""},
		{"invalid UTF-8 state", []string{"noul", "x"}, string([]byte{0xff})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, stdout, stderr, calls := testApp()
			app.Stdin = strings.NewReader(tt.stdin)
			code := app.Run(context.Background(), tt.args)
			if code != 2 || stdout.Len() != 0 || stderr.Len() == 0 || *calls != 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q calls=%d", code, stdout, stderr, *calls)
			}
		})
	}
}

func TestWhitespaceStateIsPreserved(t *testing.T) {
	app, _, stderr, _ := testApp()
	app.Stdin = strings.NewReader(" \n")
	var state string
	app.NewProvider = func(provider.Config) (provider.Provider, error) {
		return providerFunc(func(_ context.Context, request evaluation.Request) (evaluation.Response, error) {
			state = request.State
			return responseFor(request), nil
		}), nil
	}
	if code := app.Run(context.Background(), []string{"noul", "x"}); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	if state != " \n" {
		t.Fatalf("state = %q", state)
	}
}

func TestRuntimeErrorsExitOneWithoutResult(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*App)
	}{
		{"provider selection", func(app *App) { app.Getenv = func(string) string { return "" } }},
		{"invalid provider environment", func(app *App) {
			app.Getenv = func(key string) string {
				if key == "JEV_PROVIDER" {
					return "bogus"
				}
				return ""
			}
		}},
		{"factory", func(app *App) {
			app.NewProvider = func(provider.Config) (provider.Provider, error) { return nil, errors.New("factory failed") }
		}},
		{"evaluate", func(app *App) {
			app.NewProvider = func(provider.Config) (provider.Provider, error) {
				return providerFunc(func(context.Context, evaluation.Request) (evaluation.Response, error) {
					return evaluation.Response{}, errors.New("evaluate failed")
				}), nil
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, stdout, stderr, _ := testApp()
			app.Stdin = strings.NewReader("state")
			tt.setup(&app)
			code := app.Run(context.Background(), []string{"noul", "x"})
			if code != 1 || stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
		})
	}
}

func TestOutputFailureExitsOne(t *testing.T) {
	app, _, stderr, _ := testApp()
	app.Stdin = strings.NewReader("state")
	app.Stdout = failingWriter{}
	if code := app.Run(context.Background(), []string{"noul", "x"}); code != 1 || stderr.Len() == 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
}

func TestHelpAndVersionOutputFailuresExitOne(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"--version"}} {
		app, _, stderr, _ := testApp()
		app.Stdout = failingWriter{}
		if code := app.Run(context.Background(), args); code != 1 || stderr.Len() == 0 {
			t.Fatalf("args=%v code=%d stderr=%q", args, code, stderr)
		}
	}
}

func testApp() (App, *bytes.Buffer, *bytes.Buffer, *int) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	calls := 0
	app := App{
		Stdin:   strings.NewReader("state"),
		Stdout:  stdout,
		Stderr:  stderr,
		Version: "test-version",
		Getenv: func(key string) string {
			if key == "TYPESAFE_API_KEY" {
				return "test-token"
			}
			return ""
		},
		NewProvider: func(provider.Config) (provider.Provider, error) {
			return providerFunc(func(_ context.Context, request evaluation.Request) (evaluation.Response, error) {
				calls++
				return responseFor(request), nil
			}), nil
		},
	}
	return app, stdout, stderr, &calls
}

func responseFor(request evaluation.Request) evaluation.Response {
	answers := make(map[string]evaluation.Answer, len(request.Questions))
	for id, question := range request.Questions {
		switch question.Type {
		case evaluation.Noul:
			answers[id] = evaluation.Answer{Type: evaluation.Noul, Noul: value(0.75)}
		case evaluation.Choice:
			var criteria map[string]json.RawMessage
			_ = json.Unmarshal(question.Criteria, &criteria)
			probabilities := make(map[string]float64, len(criteria))
			var selected string
			for option := range criteria {
				if selected == "" {
					selected = option
				}
			}
			probabilities[selected] = 1
			answers[id] = evaluation.Answer{Type: evaluation.Choice, Choice: &selected, Confidence: value(1.0), Probabilities: probabilities}
		case evaluation.Score:
			var criteria []string
			_ = json.Unmarshal(question.Criteria, &criteria)
			legend := make(map[string]string, len(criteria))
			probabilities := make(map[string]float64, len(criteria))
			for i, label := range criteria {
				key := string(rune('0' + i))
				legend[key] = label
				probabilities[key] = 0
			}
			probabilities["0"] = 1
			answers[id] = evaluation.Answer{Type: evaluation.Score, Score: value(0.0), Confidence: value(1.0), Legend: legend, Probabilities: probabilities}
		}
	}
	return evaluation.Response{
		Provider: "typesafe", Model: "jev-test", Answers: answers,
		Usage: &evaluation.Usage{InputTokens: value(1), OutputTokens: value(1)},
	}
}

type providerFunc func(context.Context, evaluation.Request) (evaluation.Response, error)

func (f providerFunc) Evaluate(ctx context.Context, request evaluation.Request) (evaluation.Response, error) {
	return f(ctx, request)
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func value[T any](v T) *T { return &v }
