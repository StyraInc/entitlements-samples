#!/usr/bin/env python3

# This script tests that an implementation of the CarInfoStore API actually
# works. It requires that the policy be configured such that use 'alice' can
# access all API endpoints, user 'bob' can access read-only endpoints only, and
# any other user (we use 'john' for this purpose) cannot access any API
# endpoints.


# example Rego for this test:

"""
enforce[decision] {
  #title: user is alice
  input.user == "alice"
  decision := {
    "allowed": true,
    "entz": set(),
    "message": "user is alice"
  }
}

enforce[decision] {
  #title: user is bob
  input.user == "bob"
  input.method == "GET"
  decision := {
    "allowed": true,
    "entz": set(),
    "message": "user is bob"
  }
}
"""


import argparse
import requests
import json

def request(url, path, method="GET", user=None, body=None):
    """
    Perform a request against the specified URL. The return is the HTTP status
    code and the JSON-decoded version of the response body (or None if it was
    empty).
    """


    reqfunc = {
        "get": requests.get,
        "put": requests.put,
        "post": requests.post,
        "delete": requests.delete,
    }[str(method).lower()]

    headers = None
    if user is not None:
        headers = {"user": user}

    response = reqfunc("{}/{}".format(url, "/".join(path)), json=body, headers=headers)

    return response.status_code, response.json()

def main():
    parser = argparse.ArgumentParser(description="Tests the specified server against the expected behavior for CarInfoServer. The server should be started with an empty data file. User 'alice' should be allowed all endpoints and user 'bob' should be allowed read-only endpoints only.")

    parser.add_argument("url", help="URL of the CarInfoStore server to test with.")

    args = parser.parse_args()

    # results[test description] -> {"errors": ["text"]}
    results = {}

###############################################################################

    test = "/cars should be initially empty"
    results[test] = {"errors": []}
    try:
        code, response = request(args.url, ["cars"], user="alice")
        if code > 300:
            results[test]["errors"].append("status code {} should have been < 300".format(code))
        if len(response.keys()) > 0:
            results[test]["errors"].append("response '{}' should have been empty".format(response))
    except Exception as e:
        results[test]["errors"].append("exception during test: '{}'".format(e))

###############################################################################

    test = "bob should be allowed to read /cars"
    results[test] = {"errors": []}
    try:
        code, response = request(args.url, ["cars"], user="bob")
        if code > 300:
            results[test]["errors"].append("status code {} should have been < 300".format(code))
        if len(response.keys()) > 0:
            results[test]["errors"].append("response '{}' should have been empty".format(response))
    except Exception as e:
        results[test]["errors"].append("exception during test: '{}'".format(e))

###############################################################################

    # only Alice and Bob should be allowed to hit any endpoint

    test = "john should not be allowed to read /cars"
    results[test] = {"errors": []}
    try:
        code, response = request(args.url, ["cars"], user="john")
        if code < 300:
            results[test]["errors"].append("john should not be allowed to read /cars")
    except Exception as e:
        results[test]["errors"].append("exception during test: '{}'".format(e))

###############################################################################

    test = "alice should be able to POST to /cars"
    results[test] = {"errors": []}
    car0 = {"make": "Honda", "model": "CRV", "color": "black", "year": 2011}
    try:
        code, response = request(args.url, ["cars"], user="alice", method="POST", body=car0)
        if code > 300:
            results[test]["errors"].append("status code {} should have been < 300".format(code))
        if response != "car0":
            results[test]["errors"].append("response '{}' should have been 'car0'".format(response))
    except Exception as e:
        results[test]["errors"].append("exception during test: '{}'".format(e))

###############################################################################

    test = "bob should NOT be able to POST to /cars"
    results[test] = {"errors": []}
    try:
        code, response = request(args.url, ["cars"], user="bob", method="POST", body=car0)
        if code < 300:
            results[test]["errors"].append("bob should not be allowed to POST to /cars".format(code))
    except Exception as e:
        results[test]["errors"].append("exception during test: '{}'".format(e))

###############################################################################

    test = "john should NOT be able to POST to /cars"
    results[test] = {"errors": []}
    try:
        code, response = request(args.url, ["cars"], user="john", method="POST", body=car0)
        if code < 300:
            results[test]["errors"].append("john should not be allowed to POST to /cars".format(code))
    except Exception as e:
        results[test]["errors"].append("exception during test: '{}'".format(e))


###############################################################################

    test = "/cars should contain only the car Alice created"
    results[test] = {"errors": []}
    try:
        code, response = request(args.url, ["cars"], user="alice", method="GET")
        if code > 300:
            results[test]["errors"].append("status code {} should have been < 300".format(code))
        if "car0" not in response:
            results[test]["errors"].append("response '{}' should contained 'car0'".format(response))
        if response["car0"] != car0:
            results[test]["errors"].append("response '{}' should equal '{}''".format(response, car0))
    except Exception as e:
        results[test]["errors"].append("exception during test: '{}'".format(e))

###############################################################################

    failed = 0
    for test in results:
        if len(results[test]["errors"]) > 0:
            failed += 1
            print("{}: FAILED".format(test))
            for e in results[test]["errors"]:
                print("\t{}\n".format(e))
        else:
            print("{}: PASSED".format(test))

    print("")
    print("FAILED TESTS: {}".format(failed))
    exit(failed)


if __name__ == "__main__":
    main()
