# Copyright 2022 Styra Inc. All rights reserved.
# Use of this source code is governed by an Apache2
# license that can be found in the LICENSE file.

import pytest
import argparse
import requests
import json
import os

"""
Example Rego that is appropriate for this test:

enforce[decision] {
  #title: user is alice
  input.subject == "alice"
  decision := {
    "allowed": true,
    "entz": set(),
    "message": "user is alice"
  }
}

enforce[decision] {
  #title: user is bob
  input.subject == "bob"
  input.action == "GET"
  decision := {
    "allowed": true,
    "entz": set(),
    "message": "user is bob"
  }
}
"""

car0 = {"make": "Honda", "model": "CRV", "color": "black", "year": 2011}
car5 = {"make": "Ford", "model": "F-150", "color": "red", "year": 1999}
car0status = {"price": 15000, "ready": True, "sold": False}
car5status = {"price": 5000, "ready": False, "sold": True}

def request(path, method="GET", user=None, body=None):
    """
    Perform a request against the specified URL. The return is the HTTP status
    code and the JSON-decoded version of the response body (or None if it was
    empty).
    """

    url = "http://localhost:{}".format(int(os.environ["API_PORT"]))

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
    respbody = None

    try:
        if len(response.content) > 0:
            respbody = response.json()
    except json.decoder.JSONDecodeError as e:
        raise ValueError("response '{}' is not valid JSON: {}".format(str(response.content), e))

    return response.status_code, respbody

# Initially, we want to make sure that the database is empty, that alice and
# bob can read /cars, and than john cannot read /cars. We want to do this first
# because it will let us know if the environment has been misconfigured.

@pytest.mark.order(1)
def test_cars_initially_empty():
    code, response = request(["cars"], user="alice")
    assert code < 400
    assert len(response.keys()) == 0

    code, response = request(["cars"], user="bob")
    assert code < 400
    assert len(response.keys()) == 0

    # john should not be able to GET /cars
    code, response = request(["cars"], user="john")
    assert code >= 400

# Now we want alice to POST a car to /cars, which she should be allowed to do.
# bob and john should not be allowed to though.

@pytest.mark.order(2)
def test_post_cars():
    code, response = request(["cars"], user="alice", method="POST", body=car0)
    assert code < 400
    assert response == "car0"

    code, response = request(["cars"], user="bob", method="POST", body=car0)
    assert code >= 400

    code, response = request(["cars"], user="john", method="POST", body=car0)
    assert code >= 400

    # now read back /cars and make sure it contains only car0
    code, response = request(["cars"], user="alice", method="GET")
    assert code < 400
    assert response == {"car0": car0}


# Now we want alice to PUT a car into /cars with a specific ID, which isn't the
# next ID the server would already have picked for a POST. John and Bob should
# not be allowed to do this either.

@pytest.mark.order(3)
def test_put_cars():
    code, response = request(["cars", "car5"], user="alice", method="PUT", body=car5)
    assert code < 400

    code, response = request(["cars", "car6"], user="bob", method="PUT", body=car5)
    assert code >= 400

    code, response = request(["cars", "car7"], user="john", method="PUT", body=car5)
    assert code >= 400

    # now read back /cars and make sure it contains only car0 and car5
    code, response = request(["cars"], user="alice", method="GET")
    assert code < 400
    assert response == {"car0": car0, "car5": car5}

    # make sure we cannot create a car with an invalid ID
    code, response = request(["cars", "car05"], user="alice", method="PUT", body=car5)
    assert code >= 400
    code, response = request(["cars", "5"], user="alice", method="PUT", body=car5)
    assert code >= 400

# Make sure that the initial status for car0 and car5 is non-existent.

@pytest.mark.order(4)
def test_get_car_status():
    code, response = request(["cars", "car0", "status"], user="alice", method="GET")
    assert code == 404
    code, response = request(["cars", "car5", "status"], user="alice", method="GET")
    assert code == 404

# Now lets have alice PUT a car status, and also make sure that bob and john
# cannot do so.

@pytest.mark.order(5)
def test_put_car_status():
    code, response = request(["cars", "car0", "status"], user = "alice", method="PUT", body=car0status)
    assert code < 400

    code, response = request(["cars", "car0", "status"], user = "bob", method="PUT", body=car5status)
    assert code >= 400

    code, response = request(["cars", "car0", "status"], user = "john", method="PUT", body=car5status)
    assert code >= 400

    code, response = request(["cars", "car0", "status"], user="alice", method="GET")
    assert code  < 400
    assert response == car0status

    code, response = request(["cars", "car0", "status"], user="bob", method="GET")
    assert code  < 400
    assert response == car0status

    code, response = request(["cars", "car0", "status"], user="john", method="GET")
    assert code  >= 400

    # We should not be able to PUT a status to a car that does not exist
    code, response = request(["cars", "car17", "status"], user = "john", method="PUT", body=car5status)
    assert code >= 400

# Make sure that GET on a specific ar ID returns only that car ID.
@pytest.mark.order(6)
def test_get_car():
    code, response = request(["cars", "car0"], user = "alice", method="GET")
    assert code < 400
    assert response == car0

    code, response = request(["cars", "car5"], user = "alice", method="GET")
    assert code < 400
    assert response == car5

    # car1 does not exist, but this will catch it if the error message is not
    # JSON formatted
    code, response = request(["cars", "car1"], user = "alice", method="GET")
    assert code >= 400
