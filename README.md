# Observability Exercise solution

## Running the solution

To run the program, execute the following command, which includes a specification for the grouping interval in seconds. The reader will need to have a `go` binary available, this was tested with 1.13. There are no esoteric go version needs. All library calls are from the core library, no external dependencies are needed. The default interval is 5 seconds.

### To run the single core simple version:
`go run main.go --interval=5`

### To run the multicore implementation:

`go run multiproc_main.go --interval=5 --cores=2`

### To run the multihost implementation (on one host):

> start up the worker

`go run multihost_main.go --my_host="127.0.0.1:5555"`

> start up the aggregator

`go run multihost_main.go`


## Solution architecture

This solution uses the Go programming language to open up an HTTP connection to the exercise endpoint, reads lines one at a time, trims the `data :` header, deserialized from JSON, and groups them by device, title, and country to create a count of combinatorial entries.

To handle situations where a read or connection error occurs, the program uses a `goto` statement to restart the HTTP connection and begin the program again. This is a rather crude approach (current grouping state is lost), but programmatically simple and straightforward to implement.

The multicore implementation use go channels to push the serialized JSON to worker processes which deserialize (the most expensive operation and therefore the one we should parallelize), compute a time group, and send over a channel to a goroutine worker which aggregates the results and prints to STDOUT.

The multihost implementation follows the same approach as the multicore, but sends the JSON over UDP to worker hosts, which then send back via UDP to the aggregator. Again, the deserialization of JSON is the most expensive op, so we want to spread that out.


## Assumptions

* Time interval grouping need not be on integer boundaries (groupings are starttime+interval)
* Data presented in the exercise specifications are representative of data that needs to be processed (though "busted data:..." was an interesting variation, I added a regex sanitizer in the multi-host version to deal with that).

## Further Work

* How would you scale this if all events could not be processed on a single processor, or a single machine?
    * Added versions for multi-processor (uses go channels) and multi-host (uses UDP for JSON deserializer worker communication). Thse solutions basically spread out the expensive JSON deserialization operation to more processors/hosts. I did not profile this program, but I've worked with dozens of other similar programs - JSON serialization/deserialization is nearly always the constraint.
* How can your solution handle variations in data volume throughout the day?
    * I didn't add anything specific for oscillating traffic patterns. I'd likely try to optimize the performance for a couple factors over the highest traffic pattern seen; that might require rewriting in a lower level language. I used a perl randomized feed (included in the solution folder) to send example traffic to `127.0.0.1:3000` and tested the app against that, which produced higher counts than the live feed (though it looks like the cardinality of the live feed was a bit higher). We could add some sort of resource scaling with the deployment that matches the traffic patterns, but you never know when you're going to get hit with a traffic surge (well, some of the time you do).
* How would you productize this application?
    * In the simple case, I'd create a Dockerfile to package up the app, and run the program under netcat to create a simple web server output that a front end client could consume for display. If I had more time, I might do something more elegant.
* How would you test your solution to verify that it is functioning correctly?
    * I would put together a test data set, run it through the system, and compare it to the expected result. Might not be so easy to create a unit test that does this with the implementations for multi-proc and multi-host. An integration test is probably the best approach here.
