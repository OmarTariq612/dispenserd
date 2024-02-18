package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"

	"github.com/OmarTariq612/dispenserd"
)

func main() {
	unixPath := flag.String("unix", "", "unix socket path")
	flag.Parse()

	if *unixPath == "" {
		log.Fatalln("unix path must not be empty")
	}

	listener, err := dispenserd.UnixListen(*unixPath)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("POST /notify", func(w http.ResponseWriter, _ *http.Request) {
		log.Println("notified")
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("POST /resetTo", func(w http.ResponseWriter, r *http.Request) {
		var duration struct {
			Seconds uint64 `json:"duration"`
		}
		if err := json.NewDecoder(r.Body).Decode(&duration); err != nil {
			http.Error(w, "invalid duration (could not decode json)", http.StatusBadRequest)
			return
		}
		log.Printf("reseting to %d seconds", duration.Seconds)
		w.WriteHeader(http.StatusOK)
	})

	if err := http.Serve(listener, nil); err != nil {
		panic(err)
	}
}
