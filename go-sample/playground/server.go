// Copyright 2022 Styra Inc. All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package playground

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io/ioutil"
	"log"
	"time"

	"fmt"
	"net/http"
	"text/template"

	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/gorilla/mux"
)

//go:embed index.html
var indexHTMLFile string

type FormInput struct {
	Subject  *string
	Action   *string
	Resource *string
	Body     *string
}

func jsonError(w http.ResponseWriter, message string, code int) {
	var msg struct {
		Msg string `json:"msg"`
	}
	msg.Msg = message

	b, err := json.Marshal(msg)
	if err != nil {
		// should never happen
		panic(fmt.Sprintf("error while marshaling '%s': %v\n", msg, err))
	}

	http.Error(w, string(b), code)
}

// GetAPIHandler creates a router for the playground.
func GetAPIHandler() http.Handler {
	tmpl := template.New("index.html")
	template.Must(tmpl.Parse(indexHTMLFile))

	router := mux.NewRouter()

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, nil)
	})

	// serve static assets
	staticFileDirectory := http.Dir("./assets/")
	staticFileHandler := http.StripPrefix("/assets/", http.FileServer(staticFileDirectory))
	router.PathPrefix("/assets/").Handler(staticFileHandler).Methods("GET")

	router.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		// expects a FormInput object
		log.Printf("%s POST /submit\n", r.RemoteAddr)

		input := &FormInput{}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			jsonError(w, err.Error(), 400)
			return
		}
		err = json.Unmarshal(body, input)
		if err != nil {
			jsonError(w, err.Error(), 400)
			return
		}

		response, allowed, err := response(input)

		errText := ""
		if err != nil {
			errText = err.Error()
		}

		resp := struct {
			Error    string      `json:"error"`
			Allowed  bool        `json:"allowed"`
			Response interface{} `json:"response"`
		}{
			Error:    errText,
			Allowed:  allowed,
			Response: response,
		}

		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&resp)

	}).Methods("POST")

	router.HandleFunc("/highlight/{language}", func(w http.ResponseWriter, r *http.Request) {
		// POST data to this endpoint to get back a syntax-highlighted
		// HTML fragment.

		language := mux.Vars(r)["language"]

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			jsonError(w, err.Error(), 400)
			return
		}

		lexer := lexers.Get(language)
		style := styles.Get("github")
		formatter := html.New(
			html.Standalone(false),
			html.WithClasses(false),
		)
		iterator, err := lexer.Tokenise(nil, string(body))
		if err != nil {
			jsonError(w, err.Error(), 400)
			return
		}

		buf := new(bytes.Buffer)
		err = formatter.Format(buf, style, iterator)
		if err != nil {
			jsonError(w, err.Error(), 400)
			return
		}

		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/html")
		w.Write(buf.Bytes())

	}).Methods("PUT")

	router.HandleFunc("/bundle-count", func(w http.ResponseWriter, r *http.Request) {
		// Returns a single integer, being the number of times the
		// bundle has updated.

		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bundleUpdateCounter)
	}).Methods("GET")

	router.HandleFunc("/bundle-time", func(w http.ResponseWriter, r *http.Request) {
		// Returns the time at which the bundle was last updated as a
		// string.

		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bundleUpdateTimestamp.Format(time.RFC822))
	}).Methods("GET")

	fmt.Printf("index\n%s\n", indexHTMLFile)

	return router
}
