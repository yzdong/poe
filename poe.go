package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"strings"
	"sync"
)

var (
	Trace   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

var changeId string

func Init(
	traceHandle io.Writer,
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer) {

	Trace = log.New(traceHandle, "TRACE: ", log.Ldate|log.Ltime|log.Lshortfile)
	Info = log.New(infoHandle, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Warning = log.New(infoHandle, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(errorHandle, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

type Property struct {
	Name string `json:"name"`
}

type Item struct {
	Name       string `json:"name"`
	Price      string `json:"note"`
	Properties []Property
}

type Stash struct {
	Account string `json:"accountName"`
	Items   []Item `json:"items"`
}

type Req struct {
	NextChangeId string  `json:"next_change_id"`
	Stashes      []Stash `json:"stashes"`
}

func (r *Req) findItem(name string) {
	for _, stash := range r.Stashes {
		for _, item := range stash.Items {
			if strings.Contains(item.Name, name) {
				Info.Println("Found", item, "for", item.Price, "in", stash.Account)
			}
		}
	}
}

func processResponse(body io.ReadCloser) (r *Req) {
	Info.Println("Start processing response with request id", changeId)
	jsonStream := json.NewDecoder(body)
	err := jsonStream.Decode(&r)
	if err != nil {
		Error.Fatal(err)
	}
	Info.Println("Finishing processing response with new change id", r.NextChangeId, "and obtained", len(r.Stashes), "new stashes")
	/*for _, stash := range r.Stashes {
		Info.Println("Stash", stash)
		for _, item := range stash.Items {
			Info.Println("Item", item, "has properties", item.Properties)
		}
	}*/
	changeId = r.NextChangeId
	return
}

func main() {
	var wg sync.WaitGroup
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	log.Println(mem.Alloc)
	logFile, _ := os.OpenFile("log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	multiInfo := io.MultiWriter(logFile, os.Stdout)
	multiError := io.MultiWriter(logFile, os.Stderr)
	Init(logFile, multiInfo, multiInfo, multiError)
	for {
		resp, err := http.Get(fmt.Sprintf("http://www.pathofexile.com/api/public-stash-tabs?id=%s", changeId))
		if err != nil {
			Error.Println(err)
		}
		r := processResponse(resp.Body)
		r.findItem("Chaos Orb")
	}
	runtime.ReadMemStats(&mem)
	log.Println(mem.Alloc)
	wg.Add(1)
	wg.Wait()
}
