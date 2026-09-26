package evaluation

import (
	"math"
	"strings"
	"testing"
)

func TestValidateResponseAcceptsAllPrimitives(t *testing.T) {
	for _, tt := range []struct {
		name string
		pair func(*testing.T) (Request, Response)
	}{
		{"noul", validNoulPair},
		{"choice", validChoicePair},
		{"score", validScorePair},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request, response := tt.pair(t)
			if err := ValidateResponse(request, response); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestValidateResponseRejectsInvalidResponses(t *testing.T) {
	tests := []struct {
		name   string
		make   func() (Request, Response)
		needle string
	}{
		{"empty model", func() (Request, Response) { r, p := validNoulPair(t); p.Model = ""; return r, p }, "model"},
		{"missing usage", func() (Request, Response) { r, p := validNoulPair(t); p.Usage = nil; return r, p }, "usage"},
		{"missing input usage", func() (Request, Response) { r, p := validNoulPair(t); p.Usage.InputTokens = nil; return r, p }, "input_tokens"},
		{"negative usage", func() (Request, Response) { r, p := validNoulPair(t); p.Usage.OutputTokens = pointer(-1); return r, p }, "output_tokens"},
		{"missing answer", func() (Request, Response) { r, p := validNoulPair(t); delete(p.Answers, "result"); return r, p }, "answer"},
		{"extra answer", func() (Request, Response) {
			r, p := validNoulPair(t)
			p.Answers["extra"] = p.Answers["result"]
			return r, p
		}, "answer"},
		{"type mismatch", func() (Request, Response) {
			r, p := validNoulPair(t)
			a := p.Answers["result"]
			a.Type = Choice
			p.Answers["result"] = a
			return r, p
		}, "type"},
		{"missing noul", func() (Request, Response) {
			r, p := validNoulPair(t)
			a := p.Answers["result"]
			a.Noul = nil
			p.Answers["result"] = a
			return r, p
		}, "noul"},
		{"noul out of range", func() (Request, Response) {
			r, p := validNoulPair(t)
			a := p.Answers["result"]
			a.Noul = pointer(1.1)
			p.Answers["result"] = a
			return r, p
		}, "noul"},
		{"noul NaN", func() (Request, Response) {
			r, p := validNoulPair(t)
			a := p.Answers["result"]
			a.Noul = pointer(math.NaN())
			p.Answers["result"] = a
			return r, p
		}, "noul"},
		{"foreign score on noul", func() (Request, Response) {
			r, p := validNoulPair(t)
			a := p.Answers["result"]
			a.Score = pointer(0.0)
			p.Answers["result"] = a
			return r, p
		}, "field"},
		{"choice wrong keys", func() (Request, Response) {
			r, p := validChoicePair(t)
			a := p.Answers["result"]
			a.Probabilities = map[string]float64{"a": 0.5, "c": 0.5}
			p.Answers["result"] = a
			return r, p
		}, "keys"},
		{"choice unknown result", func() (Request, Response) {
			r, p := validChoicePair(t)
			a := p.Answers["result"]
			a.Choice = pointer("c")
			p.Answers["result"] = a
			return r, p
		}, "choice"},
		{"choice bad sum", func() (Request, Response) {
			r, p := validChoicePair(t)
			a := p.Answers["result"]
			a.Probabilities = map[string]float64{"a": 0.8, "b": 0.3}
			p.Answers["result"] = a
			return r, p
		}, "sum"},
		{"choice infinite confidence", func() (Request, Response) {
			r, p := validChoicePair(t)
			a := p.Answers["result"]
			a.Confidence = pointer(math.Inf(1))
			p.Answers["result"] = a
			return r, p
		}, "confidence"},
		{"score wrong legend", func() (Request, Response) {
			r, p := validScorePair(t)
			a := p.Answers["result"]
			a.Legend = map[string]string{"0": "low", "2": "high"}
			p.Answers["result"] = a
			return r, p
		}, "legend"},
		{"score wrong probability keys", func() (Request, Response) {
			r, p := validScorePair(t)
			a := p.Answers["result"]
			a.Probabilities = map[string]float64{"0": 1, "2": 0}
			p.Answers["result"] = a
			return r, p
		}, "keys"},
		{"score out of range", func() (Request, Response) {
			r, p := validScorePair(t)
			a := p.Answers["result"]
			a.Score = pointer(2.0)
			p.Answers["result"] = a
			return r, p
		}, "score"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, response := tt.make()
			err := ValidateResponse(request, response)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.needle) {
				t.Fatalf("error = %q, want substring %q", err, tt.needle)
			}
		})
	}
}

func validNoulPair(t *testing.T) (Request, Response) {
	t.Helper()
	questions, err := NewInlineNoul("Yes?")
	if err != nil {
		t.Fatal(err)
	}
	request := Request{State: "state", Questions: questions}
	response := baseResponse(Answer{Type: Noul, Noul: pointer(0.5)})
	return request, response
}

func validChoicePair(t *testing.T) (Request, Response) {
	t.Helper()
	questions, err := NewInlineChoice([]string{"a", "b"}, "Choose?")
	if err != nil {
		t.Fatal(err)
	}
	request := Request{State: "state", Questions: questions}
	response := baseResponse(Answer{
		Type: Choice, Choice: pointer("a"), Confidence: pointer(0.5),
		Probabilities: map[string]float64{"a": 0.75, "b": 0.25},
	})
	return request, response
}

func validScorePair(t *testing.T) (Request, Response) {
	t.Helper()
	questions, err := NewInlineScore([]string{"low", "high"}, "Rate?")
	if err != nil {
		t.Fatal(err)
	}
	request := Request{State: "state", Questions: questions}
	response := baseResponse(Answer{
		Type: Score, Score: pointer(0.5), Confidence: pointer(0.5),
		Legend:        map[string]string{"0": "low", "1": "high"},
		Probabilities: map[string]float64{"0": 0.5, "1": 0.5},
	})
	return request, response
}

func baseResponse(answer Answer) Response {
	return Response{
		Provider: "typesafe", Model: "jev-1.13.0",
		Answers: map[string]Answer{"result": answer},
		Usage:   &Usage{InputTokens: pointer(1), OutputTokens: pointer(1)},
	}
}

func pointer[T any](value T) *T { return &value }
