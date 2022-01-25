# Copyright 2022 Styra Inc. All rights reserved.
# Use of this source code is governed by an Apache2
# license that can be found in the LICENSE file.

from flask import Flask, json
from flask import request
from flask import Response
import argparse
import pathlib
import logging
import re
import os
import requests

##### Globals #################################################################
api = Flask(__name__)
data_file = None
opa_url = None

##### "database" access #######################################################

# These methods are not very fast - they hit disk every time any field is read
# or written. But it's easy to ensure correctness as there doesn't need to be
# any caching, and performance isn't essential as this is just an example.

def read_database():
    """
    Reads the entire database, returning it as a dictionary.
    """

    global data_file

    if not data_file.exists():
        return {
                "cars": {},
                "statuses": {},
        }

    with open(data_file, "r") as f:
        return json.load(f)

def write_database(data):
    """
    Overwrite the database on disk"
    """

    global data_file

    with open(data_file, "w") as f:
        json.dump(data, f)

        # This guarantees that the next time we read the database from disk
        # (which might be soon after this method is called), the data will
        # actually have been written.
        os.fsync(f)

def get_cars():
    """
    Return a dictionary of all cars in the database.
    """

    data = read_database()
    return data["cars"]

def get_statuses():
    """
    Return a dictionary of all statuses in the database.
    """

    data = read_database()
    return data["statuses"]

def validate_car_id(car_id):
    """
    Returns true if the car ID is valid and false otherwise.
    """

    ok = bool(re.match("^car(0|([1-9][0-9]*))$", car_id))
    api.logger.info("car_id '{}' is valid: {}".format(car_id, ok))
    return ok

def next_car_id():
    """
    Returns the next unused car ID in the database.
    """

    car_ids = set(get_cars().keys())
    new_id = 0

    # This is not very performant takes O(n).
    while True:
        if "car{}".format(new_id) not in car_ids:
            return "car{}".format(new_id)

        new_id += 1

def get_car(car_id):
    """
    Return the specified car if it exists, and otherwise None.
    """

    cars = get_cars()
    if car_id not in cars:
        return None

    return cars[car_id]

def set_car(car_id, new_car_data):
    """
    Modify a car with a new set of data.
    """

    if not validate_car_id(car_id):
        return KeyError("invalid car ID '{}'".format(car_id))

    data = read_database()

    # Poor man's schema validation - if any key is missing from the provided
    # object, it will throw a KeyError, and if any extra keys are provided,
    # they will be filtered out.
    data["cars"][car_id] = {
            "make":  str(new_car_data["make"]),
            "model": str(new_car_data["model"]),
            "year":  int(new_car_data["year"]),
            "color": str(new_car_data["color"]),
    }

    write_database(data)

def new_car(new_car_data):
    """
    Creates a new car with the next unused car ID. Returns the ID of the newly
    created car.
    """

    new_id = next_car_id()
    set_car(new_id, new_car_data)
    return new_id

def get_status(car_id):
    """
    Return the status of the specified if it exists, and otherwise None.
    """

    statuses = get_statuses()
    if car_id not in statuses:
        return None

    return statuses[car_id]

def set_status(car_id, new_status_data):
    """
    Modify a status with a new set of data.
    """

    data = read_database()

    if car_id not in data["cars"]:
        raise KeyError("cannot set status for non-existant car '{}'".format(car_id))

    # Poor man's schema validation - if any key is missing from the provided
    # object, it will throw a KeyError, and if any extra keys are provided,
    # they will be filtered out.
    data["statuses"][car_id] = {
            "ready": bool(new_status_data["ready"]),
            "sold":  bool(new_status_data["ready"]),
            "price": float(new_status_data["price"]),
    }

    write_database(data)

##### OPA integration #########################################################
def get_decision(path, user, method):
    """
    Call out to DAS to check if a particular request is permitted.
    """

    if opa_url is None:
        api.logger.info("no OPA URL provided, all requests are allowed")
        api.logger.info("path='{}' user='{}' method='{}' decision={}".format(path, user, method, True))
        return True

    response = requests.post(opa_url, json={"input": {"resource": path, "subject": user, "action": method}})
    if not response.ok:
        api.logger.error("path='{}' user='{}' method='{}' decision={} OPA reported status code {}, body: {}".format(path, user, method, False, response.status_code, response.text))
        return False

    result = response.json()
    if "result" not in result:
        api.logger.error("response from OPA '{}' had no result field".format(result))
        return False

    if "allowed" not in result["result"]:
        api.logger.error("response from OPA '{}' had no result.allowed field".format(result))
        return False

    decision = bool(response.json()["result"]["allowed"])

    api.logger.info("path='{}' user='{}' method='{}' decision={}".format(path, user, method, decision))
    return decision

def response_with_decision(path, user, method, response):
    """
    Returns the provided Flask response (if OPA allows the (path, user,
    method)), and otherwise a 403 response with a generic error message.
    """

    if not get_decision(path, user, method):
        return Response(json.dumps({"msg": "action prohibited by OPA policy"}), status=403, mimetype="application/json")

    return response

##### API implementation ######################################################

@api.route('/cars', methods=['GET'])
def api_get_cars():
    user = request.headers.get("user")
    return response_with_decision(
            ["cars"],
            user,
            "GET",
            Response(json.dumps(get_cars()), status=200, mimetype="application/json")
    )

@api.route("/cars", methods=["POST"])
def api_post_cars():
    user = request.headers.get("user")
    data = request.get_json(force=True)

    if not get_decision(["cars"], user, "POST"):
        return Response(json.dumps({"msg": "action prohibited by OPA policy"}), status=403, mimetype="application/json")

    new_id = new_car(data)
    return Response(json.dumps(new_id), status=200, mimetype="application/json")

@api.route("/cars/<string:car_id>", methods=["GET"])
def api_get_car_by_id(car_id):
    user = request.headers.get("user")

    cars = get_cars()
    resp = Response(json.dumps({"msg": "no such car with ID '{}'".format(car_id)}), status=404, mimetype="text/plain")
    if car_id in cars:
        resp = Response(json.dumps(get_car(car_id)), status=200, mimetype="application/json")

    return response_with_decision(
            ["cars", car_id],
            user,
            "GET",
            resp
    )

@api.route("/cars/<string:car_id>", methods=["PUT"])
def api_put_car_by_id(car_id):
    user = request.headers.get("user")

    if not validate_car_id(car_id):
        return Response(json.dumps({"msg": "invalid car ID '{}'".format(car_id)}), status=400, mimetype="application/json")

    if not get_decision(["cars", car_id], user, "PUT"):
        return Response(json.dumps({"msg": "action prohibited by OPA policy"}), status=403, mimetype="application/json")

    cars = get_cars()
    status = 201
    if car_id in cars:
        status=200

    set_car(car_id, request.get_json(force=True))

    return Response(None, status=status)

@api.route("/cars/<string:car_id>", methods=["DELETE"])
def api_delete_car_by_id(car_id):
    user = request.headers.get("user")

    if not get_decision(["cars", car_id], user, "DELETE"):
        return Response(json.dumps({"msg": "action prohibited by OPA policy"}), status=403, mimetype="application/json")

    data = read_database()
    if car_id in data["cars"]:
        data["cars"].pop(car_id)
        write_database(data)

    if car_id in data["statuses"]:
        data["statuses"].pop(car_id)
        write_database(data)

    return Response(None, status=200)

@api.route("/cars/<string:car_id>/status", methods=["GET"])
def api_get_car_status(car_id):
    user = request.headers.get("user")

    cars = get_cars()
    statuses = get_statuses()
    resp = None
    if car_id in cars:
        if car_id in statuses:
            resp = Response(json.dumps(statuses[car_id]), status=200, mimetype="application/json")
        else:
            resp = Response(json.dumps({"msg": "no status for car with ID '{}'".format(car_id)}), status=404, mimetype="application/json")
    else:
        resp = Response(json.dumps({"msg": "no such car with ID '{}'".format(car_id)}), status=404, mimetype="applicaiton/json")

    return response_with_decision(["cars", car_id, "status"], user, "GET", resp)

@api.route("/cars/<string:car_id>/status", methods=["PUT"])
def api_put_car_status(car_id):
    user = request.headers.get("user")

    if not get_decision(["cars", car_id, "status"], user, "PUT"):
        return Response(json.dumps({"msg": "action prohibited by OPA policy"}), status=403, mimetype="application/json")

    cars = get_cars()
    statuses = get_statuses()
    data = read_database()
    resp = None
    status_data = request.get_json(force=True)
    if car_id in cars:
        status = 201
        if car_id in statuses:
            status = 200

        data["statuses"][car_id] = status_data
        write_database(data)

        resp = Response(None, status=200)
    else:
        resp = Response(json.dumps({"msg": "no such car with ID '{}'".format(car_id)}), status=404, mimetype="application/json")

    return response_with_decision(["cars", car_id, "status"], user, "GET", resp)


def main():
    global data_file
    global opa_url
    global api

    parser = argparse.ArgumentParser(description="Example app for DAS entitlements implementing the CarInfoServer API.")

    parser.add_argument("--data", "-d", default=pathlib.Path("./data.json"), type=pathlib.Path, help="Path where the JSON data file should be stored.")

    parser.add_argument("--opa_url", "-u", default=None, help="URL of OPA endpoint to query.")

    parser.add_argument("--port", "-p", default=8123, type=int, help="Port on which the API should be served.")

    args = parser.parse_args()


    data_file = args.data
    opa_url = args.opa_url

    api.logger.setLevel(logging.INFO)
    api.run(port = args.port)


if __name__ == '__main__':
    main()
