package evaluation

import (
	"encoding/json"
	"testing"
)

func TestNewInlineNoul(t *testing.T) {
	questions, err := NewInlineNoul(" Is this safe? ")
	if err != nil {
		t.Fatal(err)
	}
	question := questions["result"]
	if question.Type != Noul {
		t.Fatalf("type = %q, want %q", question.Type, Noul)
	}
	if got := string(question.Instructions); got != `" Is this safe? "` {
		t.Fatalf("instructions = %s", got)
	}
	if question.Criteria != nil {
		t.Fatalf("criteria = %s, want nil", question.Criteria)
	}
}

func TestNewInlineChoiceUsesLabelsVerbatim(t *testing.T) {
	questions, err := NewInlineChoice([]string{"safe,automatic", "needs review", "unsafe"}, "Classify this")
	if err != nil {
		t.Fatal(err)
	}
	question := questions["result"]
	if question.Type != Choice {
		t.Fatalf("type = %q, want %q", question.Type, Choice)
	}
	var criteria map[string]any
	if err := json.Unmarshal(question.Criteria, &criteria); err != nil {
		t.Fatal(err)
	}
	for _, label := range []string{"safe,automatic", "needs review", "unsafe"} {
		description, ok := criteria[label]
		if !ok || description != nil {
			t.Fatalf("criteria[%q] = %#v, present=%v", label, description, ok)
		}
	}
}

func TestNewInlineScorePreservesOrder(t *testing.T) {
	questions, err := NewInlineScore([]string{"low", "medium", "high"}, "Rate this")
	if err != nil {
		t.Fatal(err)
	}
	var levels []string
	if err := json.Unmarshal(questions["result"].Criteria, &levels); err != nil {
		t.Fatal(err)
	}
	want := []string{"low", "medium", "high"}
	for i := range want {
		if levels[i] != want[i] {
			t.Fatalf("levels = %#v, want %#v", levels, want)
		}
	}
}

func TestNewInlineRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		call func() error
	}{
		{"empty noul", func() error { _, err := NewInlineNoul(" "); return err }},
		{"empty choice label", func() error { _, err := NewInlineChoice([]string{"safe", "", "unsafe"}, "classify"); return err }},
		{"duplicate choice label", func() error { _, err := NewInlineChoice([]string{"safe", "safe"}, "classify"); return err }},
		{"too few choice labels", func() error { _, err := NewInlineChoice([]string{"safe"}, "classify"); return err }},
		{"too few score levels", func() error { _, err := NewInlineScore([]string{"low"}, "rate"); return err }},
		{"too many score levels", func() error { _, err := NewInlineScore(make([]string, 11), "rate"); return err }},
		{"empty score instruction", func() error { _, err := NewInlineScore([]string{"low", "high"}, ""); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
