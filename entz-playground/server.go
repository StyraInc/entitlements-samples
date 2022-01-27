// Copyright 2022 Styra Inc. All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package playground

import (
	_ "embed"
	"encoding/json"
	"io/ioutil"
	"log"

	"fmt"
	"net/http"
	"text/template"

	"github.com/gorilla/mux"
)

//go:embed index.html
var indexHTMLFile string

type FormInput struct {
	Subject  string
	Action   string
	Resource string
	Body     string
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

func LaunchServer(port int) {
	tmpl := template.New("index.html")
	template.Must(tmpl.Parse(indexHTMLFile))

	router := mux.NewRouter()

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			tmpl.Execute(w, nil)
			return
		}

		data := FormInput{
			Subject:  r.FormValue("subject"),
			Action:   r.FormValue("action"),
			Resource: r.FormValue("resource"),
			Body:     r.FormValue("body"),
		}

		fmt.Printf("form response: %v\n", data)

		tmpl.Execute(w, struct{ Success bool }{true})
	})

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

	fmt.Printf("index\n%s\n", indexHTMLFile)

	err := http.ListenAndServe(fmt.Sprintf(":%d", port), router)
	if err != nil {
		panic(err)
	}
}
