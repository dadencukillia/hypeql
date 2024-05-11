package hypeql

import (
	"fmt"
	"slices"
	"testing"
)

// TEST #1

type A struct {
	Foo   string `json:"foo"`
	Bar   bool   `json:"bar" fun:"Rbar"`
	Blist []B    `json:"blist" fun:"Rblist"`
	Clist []C    `json:"clist" fun:"Rclist"`
}

type B struct {
	Text string `json:"text"`
}

type C struct {
	Text string `json:"text" fun:"Rtext"`
	Foo  bool   `json:"foo" fun:"Rfoo"`
}

func (a A) Rbar(ctx *map[string]any) bool {
	return true
}

func (a A) Rblist(ctx *map[string]any, args map[string]any) []B {
	return []B{
		{}, {},
	}
}

func (a A) Rclist(ctx *map[string]any, args map[string]any) []C {
	return []C{
		{}, {},
	}
}

func (a C) Resolve(ctx *map[string]any, fields []string) {
	if slices.Contains(fields, "text") {
		(*ctx)["text"] = "Hello"
	}
	if slices.Contains(fields, "foo") {
		(*ctx)["foo"] = "bar"
	}
}

func (a C) Rtext(ctx *map[string]any) any {
	return (*ctx)["text"]
}

func (a C) Rfoo(ctx *map[string]any) any {
	return (*ctx)["foo"]
}

func TestGeneral(t *testing.T) {
	parser := NewQueryParser(QueryParserConfig{})
	generator := NewResponseGenerator(ResponseGeneratorConfig{})

	query := `
	{
		foo
		bar
		clist {
			text
		}
	}`

	// mustBe := `{}`

	parsed, err := parser.Parse(query)
	if err != nil {
		t.Fatal("Parsing error: " + err.Error())
	}

	resp, err := generator.Generate(parsed, A{}, map[string]any{})
	if err != nil {
		t.Fatal("Process error: " + err.Error())
	}

	mustBe := `{"bar":true,"clist":[{"text":"Hello"},{"text":"Hello"}],"foo":""}`

	if resp != mustBe {
		t.Fatal("Not equal")
	}
}

// TEST #2

type ThrowsError struct {
	Value bool `json:"value"`
}

func (a ThrowsError) Resolve(ctx *map[string]any, fields []string) error {
	return fmt.Errorf("Error throws here")
}

func TestResolveError(t *testing.T) {
	generator := NewResponseGenerator(ResponseGeneratorConfig{})

	query := []any{
		"value",
	}

	_, err := generator.Generate(query, ThrowsError{}, map[string]any{})
	if err == nil || err.Error() != "Error throws here" {
		t.Fatal("Not equal")
	}
}

// TEST #3

type First struct {
	Seconds []Second `json:"seconds"`
}

type Second struct {
	Thirds []Third `json:"thirds"`
}

type Third struct {
	Fourths []Fourth `json:"fourths"`
}

type Fourth struct {
	CantReach string `json:"cantreach"` // Can't reach because of deep recursion limit
}

func TestDeepRecursionLimitGenerator(t *testing.T) {
	parser := NewQueryParser(QueryParserConfig{})
	generator := NewResponseGenerator(ResponseGeneratorConfig{
		MaxDeepRecursion: 3,
	})

	parsedQuery, err := parser.Parse("{seconds{thirds{fourths{cantreach}}}}")
	if err != nil {
		t.Fatal("Parsing error: " + err.Error())
	}

	_, err = generator.Generate(parsedQuery, First{
		Seconds: []Second{
			{
				Thirds: []Third{
					{
						Fourths: []Fourth{
							{
								CantReach: "",
							},
						},
					},
				},
			},
		},
	}, map[string]any{})
	if err == nil {
		t.Fatal("Not equal")
	}
}
