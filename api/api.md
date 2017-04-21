FORMAT: 1A

# Publish Carousel

A microservice that continuously republishes content and annotations available in the native store.

## Group Public API

### /cycles

#### Get Cycle Information [GET]

Displays state information for all configured cycles.

+ Response 200 (application/json)

    Shows the state of all configured cycles.

    + Body

            [
              {
                "id": "5118842b62670d2b",
                "name": "methode-whole-archive",
                "type": "ThrottledWholeCollection",
                "metadata": {
                  "currentPublishUuid": "c372ffba-7a7f-11e6-aca9-d6ece9a77557",
                  "errors": 0,
                  "progress": 1,
                  "state": [
                    "stopped",
                    "unhealthy"
                  ],
                  "completed": 2000,
                  "total": 100000,
                  "iteration": 3
                },
                "collection": "methode",
                "origin": "methode-web-pub",
                "coolDown": "5m"
              }
            ]

+ Response 500

    An error occurred while processing the cycles into json.

    + Body

#### Create a new Cycle [POST]

Creates and starts a new cycle with the provided configuration.

+ Request (application/json)

    + Body

            {
              "name": "methode-whole-archive-dredd",
              "type": "ThrottledWholeCollection",
              "origin": "methode-origin",
              "collection": "methode",
              "coolDown": "5m",
              "throttle": "15s"
            }

    + Schema

            {
              "type": "object",
              "properties": {
                "name": {
                  "type": "string"
                },
                "type": {
                  "type": "string",
                  "enum": [
                    "ThrottledWholeCollection",
                    "FixedWindow",
                    "ScalingWindow"
                  ]
                },
                "origin": {
                  "type": "string"
                },
                "collection": {
                  "type": "string"
                },
                "coolDown": {
                  "type": "string"
                },
                "throttle": {
                  "type": "string"
                },
                "timeWindow": {
                  "type": "string"
                },
                "minimumThrottle": {
                  "type": "string"
                },
                "maximumThrottle": {
                  "type": "string"
                }
              },
              "required": [
                "name",
                "type",
                "origin",
                "collection",
                "coolDown"
              ]
            }

+ Response 201

    The cycle has been created successfully

    + Body

+ Request (application/json)

    + Body

            {
              "name": "methode-whole-archive-dredd",
              "type": "ThrottledWholeCollection",
              "origin": "methode-origin",
              "collection": "methode",
              "coolDown": "5m",
              "throttle": "15s"
            }

    + Schema

            {
              "type": "object",
              "properties": {
                "name": {
                  "type": "string"
                },
                "type": {
                  "type": "string",
                  "enum": [
                    "ThrottledWholeCollection",
                    "FixedWindow",
                    "ScalingWindow"
                  ]
                },
                "origin": {
                  "type": "string"
                },
                "collection": {
                  "type": "string"
                },
                "coolDown": {
                  "type": "string"
                },
                "throttle": {
                  "type": "string"
                },
                "timeWindow": {
                  "type": "string"
                },
                "minimumThrottle": {
                  "type": "string"
                },
                "maximumThrottle": {
                  "type": "string"
                }
              },
              "required": [
                "name",
                "type",
                "origin",
                "collection",
                "coolDown"
              ]
            }

+ Response 400

    The provided cycle configuration is invalid.

    + Body

+ Request (application/json)

    + Body

            {
              "name": "methode-whole-archive-dredd",
              "type": "ThrottledWholeCollection",
              "origin": "methode-origin",
              "collection": "methode",
              "coolDown": "5m",
              "throttle": "15s"
            }

    + Schema

            {
              "type": "object",
              "properties": {
                "name": {
                  "type": "string"
                },
                "type": {
                  "type": "string",
                  "enum": [
                    "ThrottledWholeCollection",
                    "FixedWindow",
                    "ScalingWindow"
                  ]
                },
                "origin": {
                  "type": "string"
                },
                "collection": {
                  "type": "string"
                },
                "coolDown": {
                  "type": "string"
                },
                "throttle": {
                  "type": "string"
                },
                "timeWindow": {
                  "type": "string"
                },
                "minimumThrottle": {
                  "type": "string"
                },
                "maximumThrottle": {
                  "type": "string"
                }
              },
              "required": [
                "name",
                "type",
                "origin",
                "collection",
                "coolDown"
              ]
            }

+ Response 500

    An error occurred while creating the new cycle, or when adding it to the scheduler.

    + Body

### /cycles/{id}

#### Get Cycle Information for ID [GET]

Displays state information for the cycle with the given ID

+ Parameters

    + id: 5118842b62670d2b (required)

+ Response 200 (application/json)

    Shows the state of the cycle with the provided ID

    + Body

            {
              "id": "5118842b62670d2b",
              "name": "methode-whole-archive",
              "type": "ThrottledWholeCollection",
              "metadata": {
                "currentPublishUuid": "c372ffba-7a7f-11e6-aca9-d6ece9a77557",
                "errors": 0,
                "progress": 1,
                "state": [
                  "stopped",
                  "unhealthy"
                ],
                "completed": 2000,
                "total": 100000,
                "iteration": 3
              },
              "collection": "methode",
              "origin": "methode-web-pub",
              "coolDown": "5m"
            }

+ Response 404

    We couldn't find a cycle with the provided ID.

    + Body

+ Response 500

    An error occurred while processing the cycle into json.

    + Body

#### Delete the Cycle [DELETE]

Stops and removes the cycle from the Carousel. Deleted cycles cannot be resumed, and must be recreated.

+ Parameters

    + id: 5118842b62670d2b (required)

+ Response 204

    The cycle has been deleted successfully.

    + Body

+ Response 404

    We couldn't find a cycle with the provided ID.

    + Body

### /cycles/{id}/stop

#### Stop Cycle [POST]

Stops the running of the cycle with the provided ID, and frees up connections to Mongo.

+ Parameters

    + id: 7085a0ac743eddd8 (required)

+ Request (application/json)

    + Body

+ Response 200

    A stop has been triggered for the cycle.

    + Body

+ Request (application/json)

    + Body

+ Response 404

    We couldn't find a cycle with the provided ID.

    + Body

### /cycles/{id}/resume

#### Resume Cycle [POST]

Resumes a stopped cycle with ID.

+ Parameters

    + id: 7085a0ac743eddd8 (required)

+ Request (application/json)

    + Body

+ Response 200

    A resume has been triggered for the cycle.

    + Body

+ Request (application/json)

    + Body

+ Response 404

    We couldn't find a cycle with the provided ID.

    + Body

### /cycles/{id}/reset

#### Reset Cycle [POST]

Stops the provided cycle, and resets it back to the beginning of its iteration. N.B. the cycle will need to be resumed after being reset.

+ Parameters

    + id: 7085a0ac743eddd8 (required)

+ Request (application/json)

    + Body

+ Response 200

    A resume has been triggered for the cycle.

    + Body

+ Request (application/json)

    + Body

+ Response 404

    We couldn't find a cycle with the provided ID.

    + Body

### /scheduler/shutdown

#### Scheduler Shutdown [POST]

Stops all cycles. Useful in a production incident.

+ Request (application/json)

    + Body

+ Response 200

    Shutdown was successful.

    + Body

+ Request (application/json)

    + Body

+ Response 500

    An error occurred while shutting down the scheduler, please see the logs for details.

    + Body

### /scheduler/start

#### Start Scheduler [POST]

Resumes all cycles. Useful when the production incident has been resolved.

+ Request (application/json)

    + Body

+ Response 200

    Shutdown was successful.

    + Body

+ Request (application/json)

    + Body

+ Response 500

    An error occurred while shutting down the scheduler, please see the logs for details.

    + Body

## Group Health

### /__ping

#### Ping [GET]

Returns "pong" if the server is running.

+ Response 200 (text/plain; charset=utf-8)

    We return pong in plaintext only.

    + Body

            pong

### /__health

#### Healthchecks [GET]

Runs application healthchecks and returns FT Healthcheck style json.

+ Request

    + Headers

            Accept: application/json

    + Body

+ Response 200 (application/json)

    Should always return 200 along with the output of the healthchecks - regardless of whether the healthchecks failed or not. Please inspect the overall `ok` property to see whether or not the application is healthy.

    + Body

            {
              "checks": [
                {
                  "businessImpact": "No Business Impact.",
                  "checkOutput": "OK",
                  "lastUpdated": "2017-01-16T10:26:47.222805121Z",
                  "name": "UnhealthyCycles",
                  "ok": true,
                  "panicGuide": "https://dewey.ft.com/upp-publish-carousel.html",
                  "severity": 1,
                  "technicalSummary": "At least one of the Carousel cycles is unhealthy. This should be investigated."
                }
              ],
              "description": "Notifies clients of updates to UPP Lists.",
              "name": "publish-carousel",
              "ok": true,
              "schemaVersion": 1
            }

### /__gtg

#### Good To Go [GET]

Lightly healthchecks the application, and returns a 200 if it's Good-To-Go.

+ Response 200

    The application is healthy enough to perform all its functions correctly - i.e. good to go.

    + Body

+ Response 503

    One or more of the applications healthchecks have failed, so please do not use the app. See the /__health endpoint for more detailed information.

    + Body

## Group Info

### /__build-info

#### Build Information [GET]

Returns application build info, such as the git repository and revision, the golang version it was built with, and the app release version.

+ Response 200 (application/json; charset=UTF-8)

    Outputs build information as described in the summary.

    + Body

            {
              "version": "0.0.7",
              "repository": "https://github.com/Financial-Times/publish-carousel.git",
              "revision": "7cdbdb18b4a518eef3ebb1b545fc124612f9d7cd",
              "builder": "go version go1.6.3 linux/amd64",
              "dateTime": "20161123122615"
            }

