# Observability Exercise solution

## Running the solution

To run the program, execute the following command, which includes a specification for the grouping interval in seconds. The reader will need to have a `go` binary available, this was tested with 1.13. There are no esoteric go version needs. All library calls are from the core library, no external dependencies are needed.

`go run main.go --interval`


## Solution architecture

This solution uses the Go programming language to open up an HTTP connection to the exercise endpoint, reads lines one at a time, trims the `data :` header, deserialized from JSON, and groups them by device, title, and country to create a count of combinatorial entries.

To handle situations where a read or connection error occurs, the program uses a `goto` statement to restart the HTTP connection and begin the program again. This is a rather crude approach (current grouping state is lost), but programmatically simple and straightforward to implement.

## Assumptions

* Time interval grouping need not be on integer boundaries (groupings are starttime+interval)
* Data presented in the exercise specifications are representative of data that needs to be processed (though "busted data:..." was an interesting variation)

## Further Work

* How would you scale this if all events could not be processed on a single processor, or a single machine?
* How can your solution handle variations in data volume throughout the day?
* How would you productize this application?
* How would you test your solution to verify that it is functioning correctly?
