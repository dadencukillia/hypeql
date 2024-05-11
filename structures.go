package hypeql

// Struct that has "Parse" function that parses query
type queryParser struct {
	Config QueryParserConfig
}

type QueryParserConfig struct {
	MaxDeepRecursion uint64 // Stay 0 if unlimited
}

func NewQueryParser(config QueryParserConfig) queryParser {
	return queryParser{
		Config: config,
	}
}

// Struct that has "Generate" function that generates response
type responseGenerator struct {
	Config ResponseGeneratorConfig
}

type ResponseGeneratorConfig struct {
	MaxDeepRecursion uint64 // Stay 0 if unlimited
}

func NewResponseGenerator(config ResponseGeneratorConfig) responseGenerator {
	return responseGenerator{
		Config: config,
	}
}
