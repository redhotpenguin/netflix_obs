package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
)

// deserialized device/country/title with timestamp
type AggKey struct {
	Key       string
	TimeGroup int
}

// struct for JSON entry input
type Entry struct {
	Device  string `json:"device"`
	Sev     string `json:"sev"`
	Title   string `json:"title"`
	Country string `json:"country"`
	EpochMs int    `json:"time"`
}

// struct for calculated grouping aggregates
type Aggregate struct {
	Device  string `json:"device"`
	Sps     int    `json:"sps"`
	Title   string `json:"title"`
	Country string `json:"country"`
}

var interval *int
var cores *int

func main() {

	interval = flag.Int("interval", 5, "grouping interval in seconds")
	cores = flag.Int("cores", 2, "number of cores for deserialization")

	// channel to send deserialized entries to the aggregator
	jsonChan := make(chan string)
	aggChan := make(chan AggKey)

	// spin up goroutines for deserialization
	for i := 0; i < *cores; i++ {
		go deserialize(jsonChan, aggChan)
	}

	// spin up an aggregator
	go aggregate(aggChan)

	// try to connect every second
Connect:
	resp, err := http.Get("https://tweet-service.herokuapp.com/sps")
	//resp, err := http.Get("http://localhost:3000/")
	if err != nil {
		fmt.Printf("Connection error, sleep 1 and retry: '%s'\n", err)

		time.Sleep(1 * time.Second)

		// try to connect again
		goto Connect
	}

	// setup a reader
	reader := bufio.NewReader(resp.Body)
	for {
		// read in a line until newline
		line, err := reader.ReadString('\n')

		if err != nil {
			// handle connection and read failures
			fmt.Printf("Read error, try to connect again: %s\n", err)
			goto Connect
		}

		// skip newlines
		if line == "\n" {
			continue
		}

		// trim off the 'data :'
		data := string(line[6:])

		// send it to a json deserialize goroutine
		jsonChan <- data
	}
}

// function executed as a worker, only deserializes json, calculates time group
// generally JSON deserialization is the constraint on most programs out there
func deserialize(jsonChan chan string, aggChan chan AggKey) {

	for {
		// read data off the channel
		data := <-jsonChan

		// deserialize, skip entry if invalid json
		var entry Entry
		if err := json.Unmarshal([]byte(data), &entry); err != nil {
			// fmt.Println("could not unmarshal", err)
			continue
		}

		// skip unsuccessful entries
		if entry.Sev != "success" {
			continue
		}

		key := strings.Join([]string{entry.Device, entry.Title, entry.Country}, ".")

		// group entries by mathematically flooring a float which is the timestamp
		// divided by the (interval multiplied by 1000 for milliseconds)
		// this is a fairly simple grouping algorithm that doesn't take a lot of
		// code wrangling to implement
		timeGroup := math.Floor(float64(entry.EpochMs / (*interval * 1000)))

		// we have a key and a timeGroup, send it to the aggregator

		aggKey := AggKey{
			Key:       key,
			TimeGroup: int(timeGroup),
		}

		// send it to the aggregator worker
		aggChan <- aggKey
	}
}

// read entries off the aggregation channel, aggregate them and dump
// when timeGroup shifts
func aggregate(aggChan chan AggKey) {

	// the last time grouping seen
	timeGroupSeen := 0

	// hold counts of entry combinations seen
	var groupings = map[string]int{}

	for {

		aggKey := <-aggChan

		// if we have a new time grouping, print the old one
		if aggKey.TimeGroup > timeGroupSeen && timeGroupSeen != 0 {

			// blank line for readability
			fmt.Println()

			// serialize to json
			for k, v := range groupings {

				// deserialize our statsd format
				keys := strings.Split(k, ".")

				// create the aggregate for output
				aggregate := &Aggregate{
					Device:  keys[0],
					Sps:     v,
					Title:   keys[1],
					Country: keys[2],
				}

				jsonOut, err := json.Marshal(aggregate)
				if err != nil {
					// serialization error, skip (usually log)
					continue
				}
				// print the aggregate grouping to STDOUT
				fmt.Printf("%s\n", jsonOut)
			}

			// blank line for readability
			fmt.Println()

			// reinitialize
			groupings = map[string]int{}
		}

		// update the time group
		timeGroupSeen = int(aggKey.TimeGroup)

		// bump the count if we've seen this combo, else set to 1
		if val, ok := groupings[aggKey.Key]; ok {
			groupings[aggKey.Key] = val + 1
		} else {
			groupings[aggKey.Key] = 1
		}
	}
}
