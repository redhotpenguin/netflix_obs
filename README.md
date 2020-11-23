# Observability Exercise solution

## Running the solution

To run the program, execute the following command, which includes a specification for the grouping interval in seconds. The reader will need to have a `go` binary available, this was tested with 1.13. There are no esoteric go version needs. All library calls are from the core library, no external dependencies are needed. The default interval is 5 seconds.

### To run the single core simple version:
`go run main.go --interval=5`

### To run the multicore implementation:

`go run multiproc_main.go --interval=5 --cores=2`

### To run the multihost implementation (on one host):

start up the worker
`go run multihost_main.go --my_host="127.0.0.1:5555"`

 start up the aggregator
`go run multihost_main.go`


## Solution architecture

This solution uses the Go programming language to open up an HTTP connection to the exercise endpoint, reads lines one at a time, trims the `data :` header, deserialized from JSON, and groups them by device, title, and country to create a count of combinatorial entries.

To handle situations where a read or connection error occurs, the program uses a `goto` statement to restart the HTTP connection and begin the program again. This is a rather crude approach (current grouping state is lost), but programmatically simple and straightforward to implement.

## Assumptions

* Time interval grouping need not be on integer boundaries (groupings are starttime+interval)
* Data presented in the exercise specifications are representative of data that needs to be processed (though "busted data:..." was an interesting variation, I added a regex sanitizer in the multi-host version to deal with that).

## Further Work

* How would you scale this if all events could not be processed on a single processor, or a single machine?
    * Added versions for multi-processor (uses go channels) and multi-host (uses UDP for JSON deserializer worker communication)
* How can your solution handle variations in data volume throughout the day?
    * I didn't add anything specific for oscillating traffic patterns.
* How would you productize this application?
    * In the simple case, I'd create a Dockerfile to package up the app, and run the program under netcat to create a simple web server output that a front end client could consume for display.
* How would you test your solution to verify that it is functioning correctly?
    * I would put together a test data set, run it through the system, and compare it to the expected result. Might not be so easy to create a unit test that does this with the implementations for multi-proc and multi-host
