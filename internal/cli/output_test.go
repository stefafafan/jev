package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stefafafan/jev/internal/evaluation"
)

func TestWriteJSON(t *testing.T) {
	response := evaluation.Response{
		Provider: "typesafe",
		Model:    "jev-1.13.0",
		Answers: map[string]evaluation.Answer{
			"result": {Type: evaluation.Noul, Noul: ptr(0.0)},
		},
		Usage: &evaluation.Usage{InputTokens: ptr(0), OutputTokens: ptr(2)},
	}
	var output bytes.Buffer
	if err := writeOutput(&output, jsonOutput, response); err != nil {
		t.Fatal(err)
	}
	want := `{"provider":"typesafe","model":"jev-1.13.0","answers":{"result":{"type":"noul","noul":0}},"usage":{"input_tokens":0,"output_tokens":2}}` + "\n"
	if output.String() != want {
		t.Fatalf("output:\n%s\nwant:\n%s", output.String(), want)
	}
}

func TestParseFormat(t *testing.T) {
	for _, input := range []string{"json", "text"} {
		format, err := parseOutputFormat(input)
		if err != nil || string(format) != input {
			t.Fatalf("parseOutputFormat(%q) = %q, %v", input, format, err)
		}
	}
	if _, err := parseOutputFormat("yaml"); err == nil {
		t.Fatal("expected error for unknown format")
	}
}

func TestWriteTextNoul(t *testing.T) {
	response := evaluation.Response{
		Provider: "typesafe",
		Model:    "jev-1.13.0",
		Answers:  map[string]evaluation.Answer{"result": {Type: evaluation.Noul, Noul: ptr(0.95)}},
		Usage:    &evaluation.Usage{InputTokens: ptr(392), OutputTokens: ptr(21)},
	}
	var output bytes.Buffer
	if err := writeOutput(&output, textOutput, response); err != nil {
		t.Fatal(err)
	}
	want := "provider: typesafe\n" +
		"model: jev-1.13.0\n" +
		"answers:\n" +
		"  result (noul):\n" +
		"    noul: 0.95\n" +
		"usage:\n" +
		"  input_tokens: 392\n" +
		"  output_tokens: 21\n"
	if output.String() != want {
		t.Fatalf("output:\n%s\nwant:\n%s", output.String(), want)
	}
}

func TestWriteTextChoiceAndScore(t *testing.T) {
	tests := []struct {
		name   string
		answer evaluation.Answer
		body   string
	}{
		{"choice", evaluation.Answer{
			Type: evaluation.Choice, Choice: ptr("safe"), Confidence: ptr(0.75),
			Probabilities: map[string]float64{"safe": 0.75, "review": 0.25},
		}, `  result (choice):
    choice: safe
    confidence: 0.75
    probabilities:
      review: 0.25
      safe: 0.75
`},
		{"score", evaluation.Answer{
			Type: evaluation.Score, Score: ptr(1.25), Confidence: ptr(0.8),
			Legend:        map[string]string{"2": "high", "0": "low", "1": "medium"},
			Probabilities: map[string]float64{"2": 0.3, "0": 0.05, "1": 0.65},
		}, `  result (score):
    score: 1.25
    confidence: 0.8
    legend:
      0: low
      1: medium
      2: high
    probabilities:
      0: 0.05
      1: 0.65
      2: 0.3
`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := evaluation.Response{
				Provider: "cloudflare", Model: "jev-1.13.0",
				Answers: map[string]evaluation.Answer{"result": tt.answer},
				Usage:   &evaluation.Usage{InputTokens: ptr(10), OutputTokens: ptr(4)},
			}
			var output bytes.Buffer
			if err := writeOutput(&output, textOutput, response); err != nil {
				t.Fatal(err)
			}
			want := "provider: cloudflare\nmodel: jev-1.13.0\nanswers:\n" + tt.body +
				"usage:\n  input_tokens: 10\n  output_tokens: 4\n"
			if output.String() != want {
				t.Fatalf("output:\n%s\nwant:\n%s", output.String(), want)
			}
		})
	}
}

func TestWriteRejectsUnknownFormatBeforeWriting(t *testing.T) {
	var output bytes.Buffer
	err := writeOutput(&output, outputFormat("yaml"), evaluation.Response{})
	if err == nil || !strings.Contains(err.Error(), "output format") {
		t.Fatalf("error = %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("wrote %q", output.String())
	}
}

func TestWriteTextEscapesDynamicControlCharacters(t *testing.T) {
	response := evaluation.Response{
		Provider: "type\nsafe",
		Model:    "jev\x1b[31m",
		Answers: map[string]evaluation.Answer{
			"result": {
				Type: evaluation.Choice, Choice: ptr("safe\rchoice"), Confidence: ptr(1.0),
				Probabilities: map[string]float64{"safe\rchoice": 1, "other\toption": 0},
			},
		},
		Usage: &evaluation.Usage{InputTokens: ptr(1), OutputTokens: ptr(1)},
	}
	var output bytes.Buffer
	if err := writeOutput(&output, textOutput, response); err != nil {
		t.Fatal(err)
	}
	want := `provider: type\nsafe
model: jev\x1b[31m
answers:
  result (choice):
    choice: safe\rchoice
    confidence: 1
    probabilities:
      other\toption: 0
      safe\rchoice: 1
usage:
  input_tokens: 1
  output_tokens: 1
`
	if output.String() != want {
		t.Fatalf("output:\n%s\nwant:\n%s", output.String(), want)
	}
}

func ptr[T any](value T) *T { return &value }
