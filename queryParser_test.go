package hypeql

import (
	"reflect"
	"testing"
)

func TestQueryAndInterface(t *testing.T) {
	parser := NewQueryParser(QueryParserConfig{})

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

	parsed, err := parser.Parse(query)
	if err != nil {
		t.Fatal("Parsing error: ", err.Error())
	}

	if !reflect.DeepEqual(parsed, mustBeParsed) {
		t.Fatal("Not equal")
	}
}

func TestShortenVariant(t *testing.T) {
	parser := NewQueryParser(QueryParserConfig{})

	longQuery := `
	{
		version # Many Comments
		isBeta # Comment too
		feature(max: "1") {
			title
		}
	}`
	shortQuery := `{version,isBeta,feature(max:"1"){title}}`

	longParsed, err := parser.Parse(longQuery)
	if err != nil {
		t.Fatal("(long) Parsing error: ", err.Error())
	}

	shortParsed, err := parser.Parse(shortQuery)
	if err != nil {
		t.Fatal("(short) Parsing error: ", err.Error())
	}

	if !reflect.DeepEqual(longParsed, shortParsed) {
		t.Fatal("Not equal")
	}
}

func TestArgsQuery(t *testing.T) {
	parser := NewQueryParser(QueryParserConfig{})

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

	parsed, err := parser.Parse(query)
	if err != nil {
		t.Fatal("Parsing error: " + err.Error())
	}

	if !reflect.DeepEqual(mustBeParsed, parsed) {
		t.Fatal("Not equal")
	}
}

func TestInvalidQuery(t *testing.T) {
	parser := NewQueryParser(QueryParserConfig{})

	query := `
	{
		Hello(Args1
		) { {
	}`

	_, err := parser.Parse(query)
	if err == nil {
		t.Fatal("No errors")
	}
}

func TestDeepRecursionLimitParser(t *testing.T) {
	parser := NewQueryParser(QueryParserConfig{
		MaxDeepRecursion: 3,
	})

	_, err := parser.Parse("{seconds{thirds{fourths{cantreach}}}}")
	if err == nil {
		t.Fatal("Not equal")
	}
}
