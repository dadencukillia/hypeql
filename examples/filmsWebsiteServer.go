package main

import (
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	"github.com/dadencukillia/hypeql"
)

// Data struct declaration, you can take all data from database (but here I using random for database simulation)
type Response struct {
	Films []Film `json:"films" fun:"Rfilms"`
}

func (a Response) Rfilms(ctx *map[string]any, args map[string]any) []Film {
	// Param p = film part a user want get the information about
	if count, ok := args["p"]; ok {
		if q, ok := count.(int); ok {
			return []Film{
				{
					Id: q - 1,
				},
			}
		}
	}

	var randomin *rand.Rand
	// Loading random seed from context
	if e, ok := (*ctx)["randSeed"].(int64); ok {
		randomin = rand.New(rand.NewPCG(0, uint64(e)))
	}

	films := []Film{}
	for i := 0; i < randomin.IntN(10)+1; i++ {
		films = append(films, Film{
			Id: i,
		})
	}
	return films
}

type Film struct {
	Id          int       `json:"id" fun:"Rid"`
	Name        string    `json:"name" fun:"Rname"`
	Description string    `json:"description" fun:"Rdescription"`
	ReleaseYear int       `json:"releaseYear" fun:"RreleaseYear"`
	Comments    []Comment `json:"comments" fun:"Rcomments"`
}

func (a Film) Rname(ctx *map[string]any) string {
	var randomin *rand.Rand
	// Loading random seed from context
	if e, ok := (*ctx)["randSeed"].(int64); ok {
		randomin = rand.New(rand.NewPCG(uint64(a.Id+1), uint64(e+1)))
	}

	// List of possible part names
	partSuffixes := []string{
		"A spider under the bed",
		"Spider vs. fly",
		"The most epic movie",
		"New enemie",
		"Epic battle",
		"Tasty fly",
	}

	return "Spiderman " + fmt.Sprint(a.Id+1) + ". " + partSuffixes[randomin.IntN(len(partSuffixes))]
}

func (a Film) Rdescription(ctx *map[string]any) string {
	var randomin *rand.Rand
	// Loading random seed from context
	if e, ok := (*ctx)["randSeed"].(int64); ok {
		randomin = rand.New(rand.NewPCG(uint64(a.Id+1), uint64(e+2)))
	}

	// List of possible description texts
	descs := []string{
		"Epic movie about spiderman in " + fmt.Sprint(a.Id+1) + " part",
		"The city in the dangerous situation (again) and spiderman will save it (again)",
		"Cool movie about cool superhero",
	}

	return descs[randomin.IntN(len(descs))]
}

func (a Film) RreleaseYear(ctx *map[string]any) int {
	var randomin *rand.Rand
	// Loading random seed from context
	if e, ok := (*ctx)["randSeed"].(int64); ok {
		randomin = rand.New(rand.NewPCG(uint64(a.Id+1), uint64(e+3)))
	}

	// Release year is a number that can be from 1990 to 2024
	return 1990 + randomin.IntN(35)
}

func (a Film) Rcomments(ctx *map[string]any, args map[string]any) []Comment {
	// Comments does not changes
	return []Comment{
		{
			Username: "Spiderman fan",
			Text:     "Part " + fmt.Sprint(a.Id+1) + " is my favourite!",
		},
		{
			Username: "Spiderman",
			Text:     "Aw, I have so many fans!",
		},
	}
}

type Comment struct {
	Username string `json:"username" fun:"Rusername"`
	Text     string `json:"text" fun:"Rtext"`
}

func main() {
	// Main page
	http.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		// Not found page
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("<p>Looks like you have lost</p>"))
			return
		}

		w.Write([]byte(`<title>Movies List</title><body><noscript>Turn on the JavaScript engine!</noscript><script>
// Sending request to the API server
fetch("/api", {
	method: "post",
	body: "{films{id,name,description}}" // Response data struct
}).then(r => r.json()).then(r => {
	document.body.innerHTML = "<h2>Films:</h2>"
	for (const film of r.films) {
		document.body.innerHTML += ` + "`" + `<hr><div><h3>${film.name}</h3><p>${film.description}</p><a href="/p/${film.id+1}">More details</a></div>` + "`" + `
	}
});
</script></body>`))
	})

	// Movie detail page
	http.HandleFunc("GET /p/{id}", func(w http.ResponseWriter, r *http.Request) {
		part := r.PathValue("id")
		w.Write([]byte(`<title>Movie details</title><body><noscript>Turn on the JavaScript engine!</noscript><script>
let part = ` + fmt.Sprint(part) + `;
// Sending request to the API server
fetch("/api", {
	method: "post",
	body: "{films(p:"+part+"){name,description,releaseYear,comments{username,text}}}" // Response data struct
}).then(r => r.json()).then(r => {
	const p = r.films[0];
	document.body.innerHTML=` + "`" + `<h2>${p.name} (<i>${p.releaseYear}</i>)</h2><p>${p.description}</p><h3>Comments:</h3>` + "`" + `+p.comments.map(e => ` + "`" + `<hr><div><h3>${e.username}</h3><p>${e.text}</p></div>` + "`" + `).join("")
});
</script></body>`))
	})

	// API page
	http.HandleFunc("POST /api", func(w http.ResponseWriter, r *http.Request) {
		// Receiving random seed from cookies or creating new seed if not exists or invalid
		c, err := r.Cookie("randSeed")
		var seed int64
		if err != nil {
			seed = time.Now().Unix()
			// Writing new seed in cookies
			http.SetCookie(w, &http.Cookie{
				Name:  "randSeed",
				Value: fmt.Sprint(seed),
			})
		} else {
			// Converting seed from cookies to number
			newSeed, err := strconv.Atoi(c.Value)
			if err != nil {
				seed = time.Now().Unix()
				// Writing new seed in cookies
				http.SetCookie(w, &http.Cookie{
					Name:  "randSeed",
					Value: fmt.Sprint(seed),
				})
			} else {
				seed = int64(newSeed)
			}
		}

		// Reading request body
		cont, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			return
		}

		// Parsing request body
		i, err := hypeql.RequestBodyParse(string(cont))
		if err != nil {
			return
		}
		// Generating response body
		out, isError := hypeql.Process(i, Response{}, map[string]interface{}{
			"randSeed": seed,
		})

		if isError {
			return
		}

		// Sending response
		w.Write([]byte(out))
	})

	// Not found page for API
	http.HandleFunc("POST /", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("The API page: /api"))
	})

	// Start server
	log.Fatal(http.ListenAndServe(":8000", nil))
}
