# Endpoint Health Monitor

A simple Go utility to monitor the health and availability of HTTP endpoints defined in a YAML configuration file. It periodically sends requests, tracks response success/failure, and prints a summary table to the terminal.

## Features

Supports GET and POST (or any method) with custom headers and JSON or plain-text bodies.

Ignores port numbers when grouping domains for statistics.

Configurable request timeout (default: 500ms).

Periodic polling using a ticker (default interval: 15s) with immediate first run.

Aggregates success/failure counts per domain and displays availability percentage in a formatted table.

## Requirements

Go 1.18+ (modules enabled)


## Clone The Repo
`git clone https://github.com/stoutes/fetch-take-home.git endpoint-health-monitor`

## Build The Binary
`go build -o endpoint-health-monitor .`

## Configuration

Define your endpoints in a YAML file. Each entry supports the following fields:

- name: <arbitrary name for logging>
  method: <HTTP method, e.g. GET, POST>
  url: <full URL, including port if needed>
  headers:
    <Header-Name>: <value>
  body: <JSON literal or plain text>

Example endpoints.yaml

- name: sample-get
  method: GET
  url: https://httpbin.org/get

- name: sample-post
  method: POST
  url: https://httpbin.org/post
  headers:
    Content-Type: application/json
  body: '{"foo":"bar"}'

- name: sample-delay
  method: GET
  url: https://httpstat.us/200?sleep=1000

## Usage

Run the tool by supplying the path to your YAML config:

`go run main.go sample.yaml`

You should see log output as endpoints are checked, followed by a table:

Checking https://httpbin.org/get endpoint...

✅ available: https://httpbin.org/get

Checking https://httpbin.org/post endpoint...

✅ available: https://httpbin.org/post

Checking https://httpstat.us/200?sleep=1000 endpoint...

⏱️ timeout (500ms) for https://httpstat.us/200?sleep=1000

| DOMAIN       | SUCCESS | TOTAL | AVAILABILITY |
|--------------|---------|-------|--------------|
| httpbin.org  | 2       | 2     | 100.00%      |
| httpstat.us  | 0       | 1     | 0.00%        |

You can also check domains manually for a sanity check via:

`curl.exe --% -X POST -H "Content-Type: application/json" -d "{\"foo\":\"bar\"}" https://dev-sre-take-home-exercise-rubric.us-east-1.recruiting-public.fetchrewards.com/body`

## Customization

Timeout: Change httpClient.Timeout or the WithTimeout duration in checkHealth.

Interval: Adjust the ticker duration in monitorEndpoints.

Logging: Replace log.Printf calls with your preferred logger.

## Changes From Previous Iteration

Here's an overview of the changes made to the original source code:
### logResults()
- Changed to use tablewriter for printing out cumlative summary report.
- Changed calculation to only print out percentages in standard percentage formatting.
### monitorEndpoints()
- Refactored the sleep calculation to use Ticker instead of a raw sleep. Makes use of a ticker channel. The rest is pretty much the same (it just loops over the ticker channel).
### checkHealth()
- Refactored some of the operations into new functions. The checkHealth function itself was getting a little large after adding in all of the necessary requiremets. The functions split off were prepareBody(), buildRequest(), and recordOutcome().
- Added in a context timeout to deal with requests taking longer than 500ms.
### prepareBody()
- Changed the way it handles marshalling JSON. Before it would marshal valid JSON and cause issues with formatting and would result in 422s.
### buildRequest()
- No real changes here.
### recordOutcome()
- Added a switch statement to handle the cases of timeouts, failures, or available.
### extractDomain()
- Added a feature to ignore ports in the domain url.
### main()
- Just created a background context and passed it into monitorEndpoints()
### other / misc
- Created a shared httpClient for ease of implementation.



# License

This project is released under the MIT License. Feel free to reuse and modify.
