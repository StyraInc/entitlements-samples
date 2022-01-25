// Copyright 2022 Styra Inc. All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package sdksample

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

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

// getCars handles GET /cars, returning a list of car objects.
func getCars(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s GET /cars\n", r.RemoteAddr)

	allowed, err := decision([]string{"cars"}, r.Header.Get("user"), "GET")
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	if !allowed {
		jsonError(w, "operation prohibited by OPA policy", 403)
		return
	}

	ids := GetCarIDs()
	cars := make(map[string]Car)
	for _, id := range ids {
		var ok bool
		cars[id], ok = GetCar(id)
		if !ok {
			jsonError(w, fmt.Sprintf("have id '%s', but not matching car", id), 500)
			return
		}
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cars)
}

// postCars handles POST /cars. It expects a Car object and returns the ID of
// the car created.
func postCars(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s POST /cars\n", r.RemoteAddr)

	allowed, err := decision([]string{"cars"}, r.Header.Get("user"), "POST")
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	if !allowed {
		jsonError(w, "operation prohibited by OPA policy", 403)
		return
	}

	car := &Car{}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}
	err = json.Unmarshal(body, car)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	id := NextCarID()
	if SetCar(id, *car) {
		// the car already existed
		w.WriteHeader(200)
	} else {
		// the car has been created for the first time
		w.WriteHeader(201)
	}

	go SaveToDisk()

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(id)
}

// getCarByID handles GET /cars/{carid}
func getCarByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	log.Printf("%s GET /cars/%s\n", r.RemoteAddr, id)

	allowed, err := decision([]string{"cars", id}, r.Header.Get("user"), "GET")
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	if !allowed {
		jsonError(w, "operation prohibited by OPA policy", 403)
		return
	}

	car, ok := GetCar(id)
	if !ok {
		jsonError(w, fmt.Sprintf("no such car with ID '%s'", id), 404)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(car)
}

// putCarByID handles PUT /cars/{carid}
func putCarByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	log.Printf("%s PUT /cars/%s\n", r.RemoteAddr, id)

	allowed, err := decision([]string{"cars", id}, r.Header.Get("user"), "PUT")
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	if !allowed {
		jsonError(w, "operation prohibited by OPA policy", 403)
		return
	}

	if !ValidateID(id) {
		jsonError(w, fmt.Sprintf("invalid ID '%s'", id), 400)
		return
	}

	car := &Car{}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}
	err = json.Unmarshal(body, car)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	go SaveToDisk()

	if SetCar(id, *car) {
		// the car already existed
		w.WriteHeader(200)
	} else {
		// the car has been created for the first time
		w.WriteHeader(201)
	}
}

// deleteCarByID handles DELETE /cars/{carid}
func deleteCarByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	log.Printf("%s DELETE /cars/%s\n", r.RemoteAddr, id)

	allowed, err := decision([]string{"cars", id}, r.Header.Get("user"), "DELETE")
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	if !allowed {
		jsonError(w, "operation prohibited by OPA policy", 403)
		return
	}

	DeleteCar(id)
	go SaveToDisk()
}

// putStatus handles PUT /cars/{carid}/status
func putStatus(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	log.Printf("%s PUT /cars/%s/status\n", r.RemoteAddr, id)

	allowed, err := decision([]string{"cars", id, "status"}, r.Header.Get("user"), "PUT")
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	if !allowed {
		jsonError(w, "operation prohibited by OPA policy", 403)
		return
	}

	status := &Status{}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}
	err = json.Unmarshal(body, status)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	exists, err := SetStatus(id, *status)
	if err != nil {
		jsonError(w, err.Error(), 400)
	}

	go SaveToDisk()

	if exists {
		// the status already existed
		w.WriteHeader(200)
	} else {
		// the status has been created for the first time
		w.WriteHeader(201)
	}
}

// getStatus GET /cars/{carid}/status
func getStatus(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	log.Printf("%s GET /cars/%s/status\n", r.RemoteAddr, id)

	allowed, err := decision([]string{"cars", id, "status"}, r.Header.Get("user"), "GET")
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	if !allowed {
		jsonError(w, "operation prohibited by OPA policy", 403)
		return
	}

	status, ok := GetStatus(id)
	if !ok {
		jsonError(w, fmt.Sprintf("no status for car with ID '%s'", id), 404)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func HandleRequests(port int) {
	router := mux.NewRouter()
	router.HandleFunc("/cars", getCars).Methods("GET")
	router.HandleFunc("/cars", postCars).Methods("POST")
	router.HandleFunc("/cars/{id}", getCarByID).Methods("GET")
	router.HandleFunc("/cars/{id}", putCarByID).Methods("PUT")
	router.HandleFunc("/cars/{id}", deleteCarByID).Methods("DELETE")
	router.HandleFunc("/cars/{id}/status", getStatus).Methods("GET")
	router.HandleFunc("/cars/{id}/status", putStatus).Methods("PUT")
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), router)
	if err != nil {
		panic(err)
	}
}
