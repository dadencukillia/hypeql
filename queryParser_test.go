package hypeql

import (
	"reflect"
	"testing"
)

func TestQueryAndInterface(t *testing.T) {
	query := `
	{
		version
		isBeta
		features(max: "1") {
			title
		}
	}`

	mustBeParsed := []any{
		"version",
		"isBeta",
		[]any{
			"features",
			[]any{
				"title",
			},
			map[string]any{
				"max": "1",
			},
		},
	}

	parsed, err := RequestBodyParse(query)
	if err != nil {
		t.Fatal("Parsing error: ", err.Error())
	}

	if !reflect.DeepEqual(parsed, mustBeParsed) {
		t.Fatal("Not equal")
	}
}

func TestShortenVariant(t *testing.T) {
	longQuery := `
	{
		version # Many Comments
		isBeta # Comment too
		feature(max: "1") {
			title
		}
	}`
	shortQuery := `{version,isBeta,feature(max:"1"){title}}`

	longParsed, err := RequestBodyParse(longQuery)
	if err != nil {
		t.Fatal("(long) Parsing error: ", err.Error())
	}

	shortParsed, err := RequestBodyParse(shortQuery)
	if err != nil {
		t.Fatal("(short) Parsing error: ", err.Error())
	}

	if !reflect.DeepEqual(longParsed, shortParsed) {
		t.Fatal("Not equal")
	}
}

func TestArgsQuery(t *testing.T) {
	query := `
	{
		test(A: 1, B: "1", C: "Hello\nWorld\"", D: "Hello,"" World") {}
	}`

	mustBeParsed := []any{
		[]any{
			"test",
			[]any{},
			map[string]any{
				"A": 1,
				"B": "1",
				"C": "Hello\nWorld\"",
				"D": "Hello, World",
			},
		},
	}

	parsed, err := RequestBodyParse(query)
	if err != nil {
		t.Fatal("Parsing error: " + err.Error())
	}

	if !reflect.DeepEqual(mustBeParsed, parsed) {
		t.Fatal("Not equal")
	}
}

func TestInvalidQuery(t *testing.T) {
	query := `
	{
		Hello(Args1
		) { {
	}`

	_, err := RequestBodyParse(query)
	if err == nil {
		t.Fatal("No errors")
	}
}
