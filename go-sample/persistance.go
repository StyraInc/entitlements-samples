// Copyright 2022 Styra Inc. All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package sample

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Car represents information about a car on the lot.
type Car struct {
	// Make is the car's make, for example "Honda"
	Make string `json:"make"`

	// Model is the car's model, for example "Accord"
	Model string `json:"model"`

	// Year is the car's year of manufacture, for example 2017
	Year int `json:"year"`

	// Color is the color of the car's paint
	Color string `json:"color"`
}

// Status represents information about the status of the car.
type Status struct {
	// Sold is true if the car has already been sold.
	Sold bool `json:"sold"`

	// Ready is true if the car is ready to be sold.
	Ready bool `json:"ready"`

	// Price is the asking price for the car.
	Price float32 `json:"price"`
}

// PersistanceData represents the JSON data stored to disk by the persistence
// layer.
type PersistanceData struct {
	Cars     map[string]Car    `json:"cars"`
	Statuses map[string]Status `json:"statuses"`
}

var persistanceCars map[string]Car = map[string]Car{}
var persistanceStatuses map[string]Status = map[string]Status{}

var validIDRegex = regexp.MustCompile("^car(0|([1-9][0-9]*))$")

// Note that because we are using maps, and maps don't support concurrent
// accesses, we need to use a mutex for any operation that manipulates these
// maps, since we expect these methods to be used in a threaded context.
var persistanceMutex = new(sync.Mutex)

var persistanceFile = "./data.json"

func SetStorageDir(path string) error {
	dinfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !dinfo.IsDir() {
		return fmt.Errorf("'%s' is not a directory", path)
	}

	persistanceMutex.Lock()
	defer persistanceMutex.Unlock()

	persistanceFile = filepath.Join(path, "data.json")
	return nil
}

// SaveToDisk saves the persistance data to the disk.
func SaveToDisk() {
	persistanceMutex.Lock()
	defer persistanceMutex.Unlock()

	pd := &PersistanceData{Cars: persistanceCars, Statuses: persistanceStatuses}

	raw, err := json.Marshal(pd)
	if err != nil {
		panic(err)
	}

	// TODO: allow configuring persistence location.
	err = ioutil.WriteFile(persistanceFile+".new", raw, 0644)
	if err != nil {
		panic(err)
	}

	// File moves are (on most systems) atomic, so this mitigates the
	// chances of ending up with a half-written data file.
	os.Rename(persistanceFile+".new", persistanceFile)
}

// LoadFromDisk loads the persistence data from the disk.
func LoadFromDisk() {
	persistanceMutex.Lock()
	defer persistanceMutex.Unlock()

	pd := &PersistanceData{}

	if _, err := os.Stat(persistanceFile); errors.Is(err, os.ErrNotExist) {
		// the file does not exist
		return
	}

	raw, err := ioutil.ReadFile(persistanceFile)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(raw, pd)
	if err != nil {
		panic(err)
	}

	persistanceCars = pd.Cars
	persistanceStatuses = pd.Statuses
}

// GetCarIDs returns a list of all extant car IDs.
func GetCarIDs() []string {
	persistanceMutex.Lock()
	defer persistanceMutex.Unlock()
	ids := []string{}
	for key := range persistanceCars {
		ids = append(ids, key)
	}
	return ids
}

// GetCar returns the car with the specified ID, and a boolean indicating if
// the requested ID existed or not.
func GetCar(id string) (Car, bool) {
	persistanceMutex.Lock()
	defer persistanceMutex.Unlock()

	car, ok := persistanceCars[id]
	if !ok {
		return Car{}, false
	}

	return car, true
}

// ValidateID returns true if the given ID is valid. A car ID must be of the
// form "carXXX" where "XXX" is an integer with no leading zeros
func ValidateID(id string) bool {
	return validIDRegex.MatchString(id)
}

// DeleteCar deletes the car, as well as any associated status. If the car
// with the given ID does not exist, this has no effect.
func DeleteCar(id string) {
	persistanceMutex.Lock()
	defer persistanceMutex.Unlock()

	if _, ok := persistanceCars[id]; ok {
		delete(persistanceCars, id)
	}

	if _, ok := persistanceStatuses[id]; ok {
		delete(persistanceStatuses, id)
	}
}

// SetCar stores the specified car at the given ID, returning true if a car
// with that ID already existed. The status of the car is not updated - the
// caller may wish to delete or modify the status of the car if the ID existed
// already. The caller must validate the ID before calling this function.
func SetCar(id string, car Car) bool {
	persistanceMutex.Lock()
	defer persistanceMutex.Unlock()

	if !ValidateID(id) {
		// This should never happen, since the caller is supposed to
		// validate the ID.
		panic(fmt.Sprintf("invalid ID passed to SetCar: '%s'", id))
	}

	_, exists := persistanceCars[id]
	persistanceCars[id] = car
	return exists
}

// SetStatus overwrites the status for the specified car ID. It returns true
// if the status already existed before (e.g. this was an overwrite). It return
// an error if the specified ID does not exist in the cars list.
func SetStatus(id string, status Status) (bool, error) {
	persistanceMutex.Lock()
	defer persistanceMutex.Unlock()

	if _, ok := persistanceCars[id]; !ok {
		return false, fmt.Errorf("cannot set status of non-existent car '%s'", id)
	}

	_, exists := persistanceStatuses[id]
	persistanceStatuses[id] = status
	return exists, nil
}

// GetStatus returns the status of the specified car if one exists. The bool
// will be true if the status existed.
//
// The existence of a car does not imply the existence of a status.
func GetStatus(id string) (Status, bool) {
	persistanceMutex.Lock()
	defer persistanceMutex.Unlock()

	status, ok := persistanceStatuses[id]
	if !ok {
		return Status{}, false
	}

	return status, true
}

// NextCarID returns the next valid unused car ID.
func NextCarID() string {
	ids := GetCarIDs()
	if len(ids) == 0 {
		return "car0"
	}

	sort.Strings(ids)

	persistanceMutex.Lock()
	defer persistanceMutex.Unlock()

	lastID := ids[len(ids)-1]
	lastIDNumS := strings.Replace(lastID, "car", "", -1)
	lastIDNum, err := strconv.Atoi(lastIDNumS)
	if err != nil {
		// This should never happen, because we control the set of
		// allowed car IDs.
		panic(err)
	}

	// Just in case the given ID already somehow exists, keep incrementing
	// the number until we find an unused one.
	for {
		lastIDNum++
		newID := fmt.Sprintf("car%d", lastIDNum)
		if _, ok := persistanceCars[newID]; !ok {
			return newID
		}
	}
}
