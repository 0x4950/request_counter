package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

var FILE_NAME string = "request_timestamps.log"

type RequestCounter struct {
	sync.Mutex
	requests              []int64
	latestCurrentRequests chan []int64
	getNowTime            func() time.Time // makes testing easier
}

func main() {
	backupRequests := ReadTimestampsFromFile()
	rc := RequestCounter{
		requests:              backupRequests,
		latestCurrentRequests: make(chan []int64, 10_000),
		getNowTime:            time.Now,
	}

	go WaitForFileWrites(rc.latestCurrentRequests)

	http.HandleFunc("/", rc.HandleRequest)
	log.Fatal(http.ListenAndServe(":80", nil))
}

func (rc *RequestCounter) HandleRequest(w http.ResponseWriter, r *http.Request) {
	// take timestamps when the request was received to avoid lock hiding true time
	requestTimestamp := rc.getNowTime().Unix()
	lastMinuteTimestamp := requestTimestamp - 60

	rc.Lock()
	rc.requests = append(rc.requests, requestTimestamp)
	// Discard request timestamps older than 1 minute using binary search for efficiency
	index := sort.Search(len(rc.requests), func(i int) bool {
		return rc.requests[i] >= lastMinuteTimestamp
	})
	rc.requests = rc.requests[index:]
	rc.latestCurrentRequests <- rc.requests
	rc.Unlock()

	fmt.Fprint(w, strconv.Itoa(len(rc.requests)))
}

func ReadTimestampsFromFile() []int64 {
	file, err := os.OpenFile(FILE_NAME, os.O_CREATE|os.O_RDONLY, 0666)
	if err != nil {
		log.Fatalf("could not open file %s. error: %s", FILE_NAME, err)
	}
	defer file.Close()

	fileRequests := []int64{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		value, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			log.Printf("could not convert timestamp from line %s. error: %s", line, err)
			continue
		}
		fileRequests = append(fileRequests, value)
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("error occurred while reading file %s. error: %s", FILE_NAME, err)
	}

	return fileRequests
}

func WaitForFileWrites(latestCurrentRequests chan []int64) {
	for {
		currentRequests := <-latestCurrentRequests
		fileContent := []byte{}
		for _, reqTime := range currentRequests {
			fileContent = append(fileContent, []byte(fmt.Sprintf("%d\n", reqTime))...)
		}
		os.WriteFile(FILE_NAME, fileContent, 0666)
	}
}
