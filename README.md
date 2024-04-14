# hypeql
GraphQL like query language and a runtime environment for executing queries with dynamic data loading (using Resolvers system) for Golang.

# Future plans (To Do)
## Will updated soon:
- Replace recursion algorithm

## Global plans:
- Stability and fast
- Data modification ability via query

# How to use?
## Installation
Add the dependency to your project using the command `go get github.com/dadencukillia/hypeql`. Make sure you have Golang version 1.22.2 or higher installed.
## Examples
There are a few simple examples that can help you understand the logic of use. Click on the links to see the examples: [examples](https://github.com/dadencukillia/hypeql/tree/main/examples)
## Quick Start:
<details><summary>1. Declare struct of response:</summary>

```
type Response struct {
    Version string // API version
    LastUpdate string
    IsBeta bool
    Features []Feature
}
```
```
type Feature struct {
    Title string
    Description string
}
```
</details>

<details><summary>2. Assign JSON tags to each of the fields</summary>

```
type Response struct {
    Version string `json:"version"`
    LastUpdate string `json:"lastUpdate"`
    IsBeta bool `json:"isBeta"`
    Features []Feature `json:"features"`
}
```
```
type Feature struct {
    Title string `json:"title"`
    Description string `json:"desc"`
}
```
</details>

<details><summary>3. Create Resolver functions for fields whose values will be loaded from other sources (database, for example) and assign them appropriate "fun" tags</summary>

**What is Resolver functions?**
Resolver functions are those functions that are called when a field assigned to it is needed. It can also change the value of fields, you can use this to load values from databases. Resolver functions is feature that provide dynamic data loading for hypeql.

**Assigning "fun" tags:**
```
type Response struct {
    Version string `json:"version" fun:"Rversion"`
    LastUpdate string `json:"lastUpdate" fun:"RlastUpdate"`
    IsBeta bool `json:"isBeta" fun:"RisBeta"`
    Features []Feature `json:"features" fun:"Rfeatures"`
}
```

**You have two ways to create Resolver functions that will take information from the database:**
> The names of the Resolver functions must match the values of the "fun" tags.<br>Also  important: Resolver functions is methods of the response structs and there is a rule:
> - ✔️ Correct: `func (a Response) Resolver(...) {...}`
> - ❌ Incorrect: `func (a *Response) Resolver(...) {...}` (Don't use `*` symbol)

*Way #1 (multiple database requests)*:
```
func (a Response) Rversion(ctx *map[string]any) string {
    // MagicFunctions does not exist, I invented it to show an example of possible operations
    return MagicFunctions.ReadValueFromDB("version")
}

func (a Response) RlastUpdate(ctx *map[string]any) string {
    // MagicFunctions does not exist, I invented it to show an example of possible operations
    return MagicFunctions.ReadValueFromDB("lastUpdate")
}

func (a Response) RisBeta(ctx *map[string]any) bool {
    // MagicFunctions does not exist, I invented it to show an example of possible operations
    return MagicFunctions.ReadValueFromDB("isBeta")
}

// "args" argument exclusively for Resolver functions whose field is a slice
func (a Response) Rfeatures(ctx *map[string]any, args map[string]any) []Feature {
    // MagicFunctions does not exist, I invented it to show an example of possible operations
    return MagicFunctions.ReadValueFromDB("features")
}
```
*Way #2 (one database request)*:
```
// neededFields is Slice, is can be ["version", "lastUpdate", "isBeta", "features"] in our example
func (a Response) Resolver(ctx *map[string]any, neededFields []string) {
    // MagicFunctions does not exist, I invented it to show an example of possible operations
    values := MagicFunctions.ReadValuesFromDB(neededFields)
    for index, field := range neededFields {
        // Works if MagicFunctions.ReadValuesFromDB returns values in the same order
        (*ctx)[field] = values[index]
    }
}

// Context variables (ctx) are passed through functions as an argument and can be changed in them

func (a Response) Rversion(ctx *map[string]any) any {
    return (*ctx)["version"]
}

func (a Response) RlastUpdate(ctx *map[string]any) any {
    return (*ctx)["lastUpdate"]
}

func (a Response) RisBeta(ctx *map[string]any) any {
    return (*ctx)["isBeta"]
}

// "args" argument exclusively for Resolver functions whose field is a slice
func (a Response) Rfeatures(ctx *map[string]any, args map[string]any) any {
    return (*ctx)["features"]
}
```
</details>

<details><summary>4. Learn query language</summary>
It's simple. We must to describe the needed fields in the query from client side and send the query to the server. Just compare the following sample query with our response structure:

```
{
    version
    isBeta
    features {
        title
    }
}
```
In example we take `version`, `isBeta` values and `title` of exists features. An example of a response that we can get to a query:
```
{
    "version": "1.0.0",
    "isBeta": false,
    "features": [
        {
            "title": "Fast"
        },
        {
            "title": "Comfortable"
        }
    ]
}
```
Do you remember the "args" argument in the Resolver function? Well, in a query, we can write values to this argument. You can do it like this:
```
{
    version
    isBeta
    features(max: 3) { # Query changed here, new arg "max"
        title
    }
}
```
An example of how we can get "max" arg in the Resolver function:
```
func (a Response) Rfeatures(ctx *map[string]any, args map[string]any) any {
    features := (*ctx)["features"]

    if maxAny, ok := args["max"]; ok {
        if max, ok := maxAny.(int); ok {
            features = features[:max]
        }
    }

    return features
}
```
</details>

<details><summary>5. Simple HTTP Server</summary>

Create a project and upload the package to your project ([here's how to do it](https://github.com/dadencukillia/hypeql/tree/master?tab=readme-ov-file#installation)). Don't forget to import the package:
```
import (
    "github.com/dadencukillia/hypeql"
)
```
There are two functions in the package: "Process" and "RequestBodyParse".
- "RequestBodyParse" function needed to convert a query to understandable hypeql data type.
- "Process" function needed to process query (put it as the first argument) and return the result (JSON string and if error happened boolean).

So let's create a server:
```
import (
    "net/http"
    "github.com/dadencukillia/hypeql"
)

// Structs and Resolver functions that we already created in previous steps must be here.

func main() {
    http.HandleFunc("POST /api", func(w http.ResponseWriter, r *http.Request) {
        // Reading request body
		bodyContent, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			return
		}

        // Parsing request body
		parsedBody, err := hypeql.RequestBodyParse(string(bodyContent))
		if err != nil {
			return
		}

		// Generating response body
        initialCtx := map[string]any{}
        responseStructInstance := Response{} // Can be filled if there are not Resolver functions

		out, isError := hypeql.Process(parsedBody, responseStructInstance, initialCtx)
        if isError {
            w.WriteHeader(http.StatusBadRequest)
        }

        return out
    })

    // Serve on 8000 port
    http.ListenAndServe(":8000", nil)
}
```
</details>