package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"testing"
	"time"
)

type FakeTimer struct {
	currentTime time.Time
}

func (ft FakeTimer) Now() time.Time {
	return time.Now()
}

func (ft *FakeTimer) Set(newTime time.Time) {
	ft.currentTime = newTime
}

func TestHandler(t *testing.T) {
	fakeTimer := FakeTimer{currentTime: time.Now()}

	rc := RequestCounter{
		requests:              []int64{},
		latestCurrentRequests: make(chan []int64, 10_000),
		getNowTime:            fakeTimer.Now}

	responseRecorder := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(rc.HandleRequest)
		handler.ServeHTTP(rr, req)
		return rr
	}

	// this will overwrite FILE_NAME in package
	FILE_NAME = "request_timestamps.temp1.log"
	defer os.Remove("request_timestamps.temp1.log")

	responseCount := responseRecorder().Body.String()
	if responseCount != "1" {
		t.Errorf("expected 1 requests, got %v", responseCount)
	}
	responseCount = responseRecorder().Body.String()
	if responseCount != "2" {
		t.Errorf("expected 2 requests, got %v", responseCount)
	}
	responseCount = responseRecorder().Body.String()
	if responseCount != "3" {
		t.Errorf("expected 3 requests, got %v", responseCount)
	}
}

func TestHandlerAfter60Seconds(t *testing.T) {
	fakeTimer := FakeTimer{currentTime: time.Now()}

	rc := RequestCounter{
		requests:              []int64{},
		latestCurrentRequests: make(chan []int64, 10_000),
		getNowTime:            fakeTimer.Now}

	responseRecorder := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(rc.HandleRequest)
		handler.ServeHTTP(rr, req)
		return rr
	}

	// this will overwrite FILE_NAME in package
	FILE_NAME = "request_timestamps.temp1.log"
	defer os.Remove("request_timestamps.temp1.log")

	responseCount := responseRecorder().Body.String()
	if responseCount != "1" {
		t.Errorf("expected 1 requests, got %v", responseCount)
	}
	responseCount = responseRecorder().Body.String()
	if responseCount != "2" {
		t.Errorf("expected 2 requests, got %v", responseCount)
	}

	rc.getNowTime = func() time.Time { return time.Now().Add(65 * time.Second) }
	responseCount = responseRecorder().Body.String()
	if responseCount != "1" {
		t.Errorf("expected 1 requests, got %v", responseCount)
	}
}

func TestHandlerWithPreExistingEntries(t *testing.T) {
	filename := "request_timestamps.temp2.log"
	// this will overwrite FILE_NAME in package
	FILE_NAME = filename
	defer os.Remove(filename)

	fakeTimer := FakeTimer{currentTime: time.Now()}

	rc := RequestCounter{requests: []int64{
		time.Now().Add(-90 * time.Second).Unix(),
		time.Now().Add(-62 * time.Second).Unix(),
		time.Now().Add(-30 * time.Second).Unix(),
		time.Now().Add(-10 * time.Second).Unix()},
		latestCurrentRequests: make(chan []int64, 10_000),
		getNowTime:            fakeTimer.Now,
	}

	responseRecorder := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(rc.HandleRequest)
		handler.ServeHTTP(rr, req)
		return rr
	}

	responseCount := responseRecorder().Body.String()
	if responseCount != "3" {
		t.Errorf("expected 3 requests, got %v", responseCount)
	}
}

func TestReadTimestampsFromFile(t *testing.T) {
	filename := "testReadTimestampsFromFile.log"
	f, err := os.Create(filename)
	if err != nil {
		t.Fatalf("could not create file. err: %s", err)
		return
	}
	defer os.Remove(filename)

	someday, err := time.Parse("2006-01-02", "2024-08-31")
	if err != nil {
		t.Fatalf("could not parse example date")
	}
	timestampsWritten := []int64{
		someday.Add(30 * time.Second).Unix(),
		someday.Add(60 * time.Second).Unix(),
		someday.Add(90 * time.Second).Unix(),
		someday.Add(120 * time.Second).Unix(),
	}

	_, err = f.WriteString(fmt.Sprintf(
		"%d\n%d\n%d\n%d",
		timestampsWritten[0],
		timestampsWritten[1],
		timestampsWritten[2],
		timestampsWritten[3]))
	if err != nil {
		t.Fatalf("could not write data to file")
	}

	timestampesRead := ReadTimestampsFromFile()
	for _, ts := range timestampesRead {
		if !slices.Contains(timestampsWritten, ts) {
			t.Errorf("expected timestamp %d not found in file", ts)
		}
	}
}
