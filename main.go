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

func main() {

	interval := flag.Int("interval", 5, "grouping interval in seconds")

	// hold counts of entry combinations seen
	var groupings = map[string]int{}

	// struct for calculated grouping aggregates
	type Aggregate struct {
		Device  string `json:"device"`
		Sps     int    `json:"sps"`
		Title   string `json:"title"`
		Country string `json:"country"`
	}

	// struct for JSON entry input
	type Entry struct {
		Device  string `json:"device"`
		Sev     string `json:"sev"`
		Title   string `json:"title"`
		Country string `json:"country"`
		EpochMs int    `json:"time"`
	}

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

	// the last time grouping seen
	timeGroupSeen := 0

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

		// deserialize, skip entry if invalid json
		var entry Entry
		if err := json.Unmarshal([]byte(data), &entry); err != nil {
			// fmt.Println("could not unmarshal", err)
			continue
		}

		if entry.Sev != "success" {
			continue
		}

		timeGroup := math.Floor(float64(entry.EpochMs / (*interval * 1000)))

		// if we have a new time grouping, print the old one
		if int(timeGroup) > timeGroupSeen && timeGroupSeen != 0 {

			// blank line for readability
			fmt.Println()

			// serialize to json
			for k, v := range groupings {
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
		timeGroupSeen = int(timeGroup)

		// bump the count if we've seen this combo, else set to 1
		key := strings.Join([]string{entry.Device, entry.Title, entry.Country}, ".")
		if val, ok := groupings[key]; ok {
			groupings[key] = val + 1
		} else {
			groupings[key] = 1
		}
	}
}
