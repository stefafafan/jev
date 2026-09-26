package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strconv"

	"github.com/stefafafan/jev/internal/evaluation"
)

type outputFormat string

const (
	jsonOutput outputFormat = "json"
	textOutput outputFormat = "text"
)

func parseOutputFormat(value string) (outputFormat, error) {
	switch outputFormat(value) {
	case jsonOutput, textOutput:
		return outputFormat(value), nil
	default:
		return "", fmt.Errorf("unsupported output format %q; expected json or text", value)
	}
}

func writeOutput(w io.Writer, format outputFormat, response evaluation.Response) error {
	switch format {
	case jsonOutput:
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(response); err != nil {
			return fmt.Errorf("write JSON output: %w", err)
		}
		return nil
	case textOutput:
		return writeText(w, response)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func writeText(w io.Writer, response evaluation.Response) error {
	buffer := bufio.NewWriter(w)
	writer := textWriter{writer: buffer}
	writer.line("provider: %s", safeText(response.Provider))
	writer.line("model: %s", safeText(response.Model))
	writer.line("answers:")
	answer := response.Answers["result"]
	writer.line("  result (%s):", answer.Type)
	switch answer.Type {
	case evaluation.Noul:
		writer.line("    noul: %s", number(*answer.Noul))
	case evaluation.Choice:
		writer.line("    choice: %s", safeText(*answer.Choice))
		writer.line("    confidence: %s", number(*answer.Confidence))
		writeProbabilities(&writer, answer.Probabilities, mapKeys(answer.Probabilities))
	case evaluation.Score:
		writer.line("    score: %s", number(*answer.Score))
		writer.line("    confidence: %s", number(*answer.Confidence))
		writer.line("    legend:")
		for i := 0; i < len(answer.Legend); i++ {
			key := strconv.Itoa(i)
			writer.line("      %s: %s", key, safeText(answer.Legend[key]))
		}
		keys := make([]string, len(answer.Probabilities))
		for i := range keys {
			keys[i] = strconv.Itoa(i)
		}
		writeProbabilities(&writer, answer.Probabilities, keys)
	}
	writer.line("usage:")
	writer.line("  input_tokens: %d", *response.Usage.InputTokens)
	writer.line("  output_tokens: %d", *response.Usage.OutputTokens)
	if writer.err != nil {
		return fmt.Errorf("write text output: %w", writer.err)
	}
	if err := buffer.Flush(); err != nil {
		return fmt.Errorf("write text output: %w", err)
	}
	return nil
}

func writeProbabilities(writer *textWriter, probabilities map[string]float64, keys []string) {
	writer.line("    probabilities:")
	for _, key := range keys {
		writer.line("      %s: %s", safeText(key), number(probabilities[key]))
	}
}

func mapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func number(value float64) string { return strconv.FormatFloat(value, 'g', -1, 64) }

func safeText(value string) string {
	quoted := strconv.QuoteToGraphic(value)
	return quoted[1 : len(quoted)-1]
}

type textWriter struct {
	writer io.Writer
	err    error
}

func (w *textWriter) line(format string, args ...any) {
	if w.err != nil {
		return
	}
	_, w.err = fmt.Fprintf(w.writer, format+"\n", args...)
}
