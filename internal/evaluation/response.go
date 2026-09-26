package evaluation

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

type Answer struct {
	Type          QuestionType       `json:"type"`
	Noul          *float64           `json:"noul,omitempty"`
	Choice        *string            `json:"choice,omitempty"`
	Score         *float64           `json:"score,omitempty"`
	Confidence    *float64           `json:"confidence,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Legend        map[string]string  `json:"legend,omitempty"`
}

type Usage struct {
	InputTokens  *int `json:"input_tokens"`
	OutputTokens *int `json:"output_tokens"`
}

type Response struct {
	Provider string            `json:"provider"`
	Model    string            `json:"model"`
	Answers  map[string]Answer `json:"answers"`
	Usage    *Usage            `json:"usage"`
}

func ValidateResponse(request Request, response Response) error {
	if response.Provider == "" {
		return fmt.Errorf("response provider is empty")
	}
	if response.Model == "" {
		return fmt.Errorf("response model is empty")
	}
	if response.Usage == nil {
		return fmt.Errorf("response usage is missing")
	}
	if response.Usage.InputTokens == nil || *response.Usage.InputTokens < 0 {
		return fmt.Errorf("response usage input_tokens is missing or negative")
	}
	if response.Usage.OutputTokens == nil || *response.Usage.OutputTokens < 0 {
		return fmt.Errorf("response usage output_tokens is missing or negative")
	}
	if len(request.Questions) != 1 {
		return fmt.Errorf("request must contain exactly one question")
	}
	question, ok := request.Questions["result"]
	if !ok {
		return fmt.Errorf("request question ID must be %q", "result")
	}
	if len(response.Answers) != 1 {
		return fmt.Errorf("response must contain exactly one answer")
	}
	answer, ok := response.Answers["result"]
	if !ok {
		return fmt.Errorf("response is missing answer %q", "result")
	}
	if answer.Type != question.Type {
		return fmt.Errorf("answer type %q does not match question type %q", answer.Type, question.Type)
	}
	if err := validateAnswer(question, answer); err != nil {
		return fmt.Errorf("answer: %w", err)
	}
	return nil
}

func validateAnswer(question Question, answer Answer) error {
	switch question.Type {
	case Noul:
		if answer.Noul == nil {
			return fmt.Errorf("noul is missing")
		}
		if err := probability("noul", *answer.Noul); err != nil {
			return err
		}
		if answer.Choice != nil || answer.Score != nil || answer.Confidence != nil || answer.Probabilities != nil || answer.Legend != nil {
			return fmt.Errorf("contains field not valid for noul")
		}
	case Choice:
		if answer.Choice == nil || *answer.Choice == "" {
			return fmt.Errorf("choice is missing")
		}
		if answer.Confidence == nil {
			return fmt.Errorf("confidence is missing")
		}
		if err := probability("confidence", *answer.Confidence); err != nil {
			return err
		}
		if answer.Noul != nil || answer.Score != nil || answer.Legend != nil {
			return fmt.Errorf("contains field not valid for choice")
		}
		expected, err := choiceKeys(question)
		if err != nil {
			return err
		}
		if _, ok := expected[*answer.Choice]; !ok {
			return fmt.Errorf("choice %q is not a requested option", *answer.Choice)
		}
		if err := validateDistribution(answer.Probabilities, expected); err != nil {
			return err
		}
	case Score:
		if answer.Score == nil {
			return fmt.Errorf("score is missing")
		}
		if answer.Confidence == nil {
			return fmt.Errorf("confidence is missing")
		}
		if err := probability("confidence", *answer.Confidence); err != nil {
			return err
		}
		if answer.Noul != nil || answer.Choice != nil {
			return fmt.Errorf("contains field not valid for score")
		}
		count, err := scoreLevelCount(question)
		if err != nil {
			return err
		}
		if !finite(*answer.Score) || *answer.Score < 0 || *answer.Score > float64(count-1) {
			return fmt.Errorf("score must be finite and between 0 and %d", count-1)
		}
		expected := make(map[string]struct{}, count)
		for i := 0; i < count; i++ {
			expected[strconv.Itoa(i)] = struct{}{}
		}
		if !sameKeys(answer.Legend, expected) {
			return fmt.Errorf("legend keys do not match score levels")
		}
		if err := validateDistribution(answer.Probabilities, expected); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported answer type %q", answer.Type)
	}
	return nil
}

func validateDistribution(values map[string]float64, expected map[string]struct{}) error {
	if !sameKeys(values, expected) {
		return fmt.Errorf("probability keys do not match criteria")
	}
	var sum float64
	for key, value := range values {
		if err := probability("probability "+key, value); err != nil {
			return err
		}
		sum += value
	}
	if math.Abs(sum-1) > 1e-6 {
		return fmt.Errorf("probability sum is %g, want 1", sum)
	}
	return nil
}

func probability(name string, value float64) error {
	if !finite(value) || value < 0 || value > 1 {
		return fmt.Errorf("%s must be finite and between 0 and 1", name)
	}
	return nil
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func choiceKeys(question Question) (map[string]struct{}, error) {
	var criteria map[string]json.RawMessage
	if err := json.Unmarshal(question.Criteria, &criteria); err != nil {
		return nil, fmt.Errorf("decode choice criteria: %w", err)
	}
	keys := make(map[string]struct{}, len(criteria))
	for key := range criteria {
		keys[key] = struct{}{}
	}
	return keys, nil
}

func scoreLevelCount(question Question) (int, error) {
	var criteria []json.RawMessage
	if err := json.Unmarshal(question.Criteria, &criteria); err != nil {
		return 0, fmt.Errorf("decode score criteria: %w", err)
	}
	return len(criteria), nil
}

func sameKeys[V any](values map[string]V, expected map[string]struct{}) bool {
	if len(values) != len(expected) {
		return false
	}
	for key := range values {
		if _, ok := expected[key]; !ok {
			return false
		}
	}
	return true
}
