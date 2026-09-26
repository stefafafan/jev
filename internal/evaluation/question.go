package evaluation

import (
	"encoding/json"
	"fmt"
	"strings"
)

type QuestionType string

const (
	Noul   QuestionType = "noul"
	Choice QuestionType = "choice"
	Score  QuestionType = "score"
)

type Question struct {
	Type         QuestionType    `json:"type"`
	Instructions json.RawMessage `json:"instructions"`
	Criteria     json.RawMessage `json:"criteria,omitempty"`
}

type Request struct {
	State     string              `json:"state"`
	Questions map[string]Question `json:"questions"`
}

func NewInlineNoul(instructions string) (map[string]Question, error) {
	if strings.TrimSpace(instructions) == "" {
		return nil, fmt.Errorf("noul instructions must not be empty")
	}
	encoded, _ := json.Marshal(instructions)
	return map[string]Question{"result": {Type: Noul, Instructions: encoded}}, nil
}

func NewInlineChoice(labels []string, instructions string) (map[string]Question, error) {
	values, err := normalizeLabels(labels, 2, 255, "choice")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(instructions) == "" {
		return nil, fmt.Errorf("choice instructions must not be empty")
	}
	criteria := make(map[string]any, len(values))
	for _, label := range values {
		criteria[label] = nil
	}
	return inlineQuestion(Choice, instructions, criteria), nil
}

func NewInlineScore(levels []string, instructions string) (map[string]Question, error) {
	values, err := normalizeLabels(levels, 2, 10, "score")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(instructions) == "" {
		return nil, fmt.Errorf("score instructions must not be empty")
	}
	return inlineQuestion(Score, instructions, values), nil
}

func inlineQuestion(questionType QuestionType, instructions string, criteria any) map[string]Question {
	encodedInstructions, _ := json.Marshal(instructions)
	encodedCriteria, _ := json.Marshal(criteria)
	return map[string]Question{
		"result": {
			Type:         questionType,
			Instructions: encodedInstructions,
			Criteria:     encodedCriteria,
		},
	}
}

func normalizeLabels(input []string, minCount, maxCount int, kind string) ([]string, error) {
	if len(input) < minCount || len(input) > maxCount {
		return nil, fmt.Errorf("%s requires between %d and %d labels", kind, minCount, maxCount)
	}
	labels := make([]string, len(input))
	seen := make(map[string]struct{}, len(input))
	for i, raw := range input {
		labels[i] = strings.TrimSpace(raw)
		if labels[i] == "" {
			return nil, fmt.Errorf("%s labels must not be empty", kind)
		}
		if _, ok := seen[labels[i]]; ok {
			return nil, fmt.Errorf("%s label %q is duplicated", kind, labels[i])
		}
		seen[labels[i]] = struct{}{}
	}
	return labels, nil
}
