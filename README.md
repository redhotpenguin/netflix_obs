# Observability Exercise solution

## Overview

Hello! I spent between 4-5 hours to develop this solution to the observability exercise. I chose Go as the implementation language since I'm a bit faster to develop there than Java (and time was of the essence), yet many of the data structures and methods will be similar to a Java based approach. I have provided three implementations, a simple version for a single processor implementation, an implementation using Go channels and goroutines (worker threads) for the multi-processor implementation, and a distributed implementation using UDP for gossip between hosts. There are some limitations to this approach, but it was quick to implement a feature complete solution. In a production situation, I would refactor functionality between the three approaches, but for this exercise I created a new implementation containing the previous structure.

## Running the solution

To run the program, execute the following command, which includes a specification for the grouping interval in seconds. The reader will need to have a `go` binary available, this was tested with 1.13. There are no esoteric go version needs. All library calls are from the core library, no external dependencies are needed. The default result grouping interval is 5 seconds.

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

This solution uses the Go programming language to open up an HTTP connection to the exercise endpoint, reads lines one at a time, trims the `data :` header, deserialized from JSON, and groups them by device, title, and country to create a count of combinatorial entries. When grouping the entries, the device, title, and country are joined together in a statsd type 'foo.bar' format. I could have used a nested hash to do this, but fewer code gymnastics are required with the statsd type approach.

To handle situations where a read or connection error occurs, the program uses a `goto` statement to restart the HTTP connection and begin the program again. This is a rather crude approach (current grouping state is lost), but programmatically simple and straightforward to implement. From experience, this is the most likely situation to encounter from a reliability standpoint with a streaming HTTP feed.

The multicore implementation use go channels to push the serialized JSON to worker processes which deserialize (the most expensive operation and therefore the one we should parallelize), compute a time group, and send over a channel to a goroutine worker which aggregates the results and prints to STDOUT.

The multihost implementation follows the same approach as the multicore, but sends the JSON over UDP to worker hosts, which then send back via UDP to the aggregator. Again, the deserialization of JSON is the most expensive op, so we want to spread that out.


## Assumptions

* Time interval grouping need not be on integer boundaries (groupings are starttime+interval)
* Data presented in the exercise specifications are representative of data that needs to be processed (though "busted data:..." was an interesting variation, I added a regex sanitizer to deal with that).

## Further Work

* How would you scale this if all events could not be processed on a single processor, or a single machine?
    * Added versions for multi-processor (uses go channels) and multi-host (uses UDP for JSON deserializer worker communication). Thse solutions basically spread out the expensive JSON deserialization operation to more processors/hosts. I did not profile this program, but I've worked with dozens of other similar programs - JSON serialization/deserialization is nearly always the constraint.
* How can your solution handle variations in data volume throughout the day?
    * I didn't add anything specific for oscillating traffic patterns. I'd likely try to optimize the performance for a couple factors over the highest traffic pattern seen; that might require rewriting in a lower level language. I used a perl randomized feed (included in the solution folder) to send example traffic to `127.0.0.1:3000` and tested the app against that, which produced higher counts than the live feed (though it looks like the cardinality of the live feed was a bit higher). We could add some sort of resource scaling with the deployment that matches the traffic patterns, but you never know when you're going to get hit with a traffic surge (well, some of the time you do). The multi-host implementation could be adapted to have a resource management worker which could provision additional hosts if the need arises; think autoscaling, but controlled by the application. If I had to manage this service without that and limited dependencies, I might be tempted to deploy a high performance C version which should be able to handle a very high rate of requests and avoid horizontal scaling challenges.
* How would you productize this application?
    * In the simple case, I'd create a Dockerfile to package up the app, and run the program under netcat to create a simple web server output that a front end client could consume for display. Essentially have the service cache the lastest result grouping and provide that as JSON to a consuming UI element. If I had more time, I might do something more elegant if the needs dictated it. More information would be needed regarding the personas of the product users to solve this problem.
* How would you test your solution to verify that it is functioning correctly?
    * I would put together a test data set, run it through the system, and compare it to the expected result. Might not be so easy to create a unit test that does this with the implementations for multi-proc and multi-host. An integration test is probably the best approach here overall. The grouping logic would probably be served well by a unit test since that represents the core business logic; everything else is transport focused.
