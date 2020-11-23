package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// deserialized 'device.country.title' with timestamp
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
var workerHost string
var myHost string
var aggregatorHost string

func main() {

	interval = flag.Int("interval", 5, "grouping interval in seconds")
	cores = flag.Int("cores", 2, "number of cores for deserialization")
	flag.StringVar(&aggregatorHost, "aggregator_host", "127.0.0.1:6666", "hostport of the aggregator")
	flag.StringVar(&workerHost, "worker_host", "127.0.0.1:5555", "hostport of the worker host")
	flag.StringVar(&myHost, "my_host", "127.0.0.1:4444", "hostport of this host")
	flag.Parse()

	// channel to send deserialized entries to the aggregator

	fmt.Println("worker host ", workerHost)
	fmt.Println("my host ", myHost)

	//
	// Spin up some worker goroutines depending on the role of this machine

	// Deserializer worker role
	if workerHost == myHost {
		fmt.Println("worker host, spinning up deserializers")
		// this host is a worker, spin up the deserializers
		deserialize()
	}

	// Aggregator role
	if workerHost != myHost {
		fmt.Println("aggregator host")
		// this host is not a worker, spin up the aggregator
		go aggregate()

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

		// prepare a UDP client
		fmt.Println("opening up socket to worker host ", workerHost)
		conn, err := net.Dial("udp", workerHost)
		if err != nil {
			panic(err)
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

			// send it to a json deserialize worker
			fmt.Fprintf(conn, data)
		}
	}
}

// function executed as a worker, only deserializes json, calculates time group
// generally JSON deserialization is the constraint on most programs out there
func deserialize() {

	// open up a UDP socket to read incoming json data
	fmt.Println("opening up UDP socket to receive data for ", myHost)
	udpAddr, err := net.ResolveUDPAddr("udp4", myHost)
	if err != nil {
		panic(err)
	}

	listener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	// prepare a UDP client to send results to aggregator
	fmt.Println("preparing aggregator client ", aggregatorHost)
	conn, err := net.Dial("udp", aggregatorHost)
	if err != nil {
		panic(err)
	}

	// regexp to sanitize title data
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		panic(err)
	}

	for {
		// read data off the socket
		buff := make([]byte, 1024)
		n, _, err := listener.ReadFromUDP(buff)
		if err != nil {
			// log the error??
		}
		data := string(buff[:n])

		// deserialize, skip entry if invalid json
		var entry Entry
		if err := json.Unmarshal([]byte(data), &entry); err != nil {
			fmt.Println("could not unmarshal", err)
			continue
		}

		// skip unsuccessful entries
		if entry.Sev != "success" {
			continue
		}

		sanitizedTitle := reg.ReplaceAllString(entry.Title, "")
		key := strings.Join([]string{entry.Device, sanitizedTitle, entry.Country}, ".")

		// group entries by mathematically flooring a float which is the timestamp
		// divided by the (interval multiplied by 1000 for milliseconds)
		// this is a fairly simple grouping algorithm that doesn't take a lot of
		// code wrangling to implement
		timeGroup := math.Floor(float64(entry.EpochMs / (*interval * 1000)))

		// we have a key and a timeGroup, send it to the aggregator colon delimited
		encoded := strings.Join([]string{key, strconv.Itoa(int(timeGroup))}, ":")
		fmt.Fprintf(conn, encoded)
	}
}

// read entries off the UDP socket 'key:timeGroup'
// aggregate them and dump when timeGroup shifts
func aggregate() {

	// the last time grouping seen
	timeGroupSeen := 0

	// hold counts of entry combinations seen
	var groupings = map[string]int{}

	// open up a UDP socket to read worker data
	fmt.Println("aggregator opening up UDP reader ", aggregatorHost)
	udpAddr, err := net.ResolveUDPAddr("udp4", aggregatorHost)
	if err != nil {
		panic(err)
	}

	listener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {

		buff := make([]byte, 1024)
		n, _, err := listener.ReadFromUDP(buff)
		if err != nil {
			// log the error??
		}
		line := string(buff[:n])

		// split the message parts into 'device.title.country' and timegroup
		values := strings.Split(line, ":")
		key := values[0]
		timeGroup, _ := strconv.Atoi(values[1])

		// if we have a new time grouping, print the old one
		if timeGroup > timeGroupSeen && timeGroupSeen != 0 {

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
		timeGroupSeen = int(timeGroup)

		// bump the count if we've seen this combo, else set to 1
		if val, ok := groupings[key]; ok {
			groupings[key] = val + 1
		} else {
			groupings[key] = 1
		}
	}
}
