package httpsample

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

// opaURL stores the URL of the OPA server to query for decisions. If this is
// set to the empty string, then all decisions will return "true".
var opaURL string = ""

func SetOPAURL(url string) {
	opaURL = url
}

// decision asks the OPA server for a decision on the given path, user, and
// method. If the operation is permitted, then it returns true, and otherwise
// false. If no OPA URL is configured, then it always returns true.
func decision(path []string, user, method string) (bool, error) {
	log.Printf("Asking OPA for a decision on path='%v' user='%s', method='%s'\n", path, user, method)
	if opaURL == "" {
		log.Printf("No OPA URL configured, allowing operation")
		return true, nil
	}

	// Prepare the data to be sent to the OPA server
	var reqStruct struct {
		Input *OPAInput `json:"input"`
	}
	reqStruct.Input = &OPAInput{
		Path:   path,
		User:   user,
		Method: method,
	}
	reqData, err := json.Marshal(reqStruct)
	if err != nil {
		log.Println(err)
		return false, err
	}

	// Perform the PUT
	resp, err := http.Post(opaURL, "application/json; charset=utf-8", bytes.NewBuffer(reqData))
	if err != nil {
		log.Println(err)
		return false, err
	}
	defer resp.Body.Close()

	// Decode the response
	var decision OPADecision
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return false, err
	}
	err = json.Unmarshal(bodyBytes, &decision)
	if err != nil {
		log.Println(err)
		return false, err
	}

	log.Printf("OPA result: allowed=%v\n", decision.Result.Allowed)

	return decision.Result.Allowed, nil

}
