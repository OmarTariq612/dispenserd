package dispenserd

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"time"
)

type Observer interface {
	Notified(time.Duration)
	ResetedTo(time.Duration)
}

type loggingObserver struct{}

func NewLoggingObserver() loggingObserver {
	return loggingObserver{}
}

func (n loggingObserver) Notified(duration time.Duration) {
	log.Println("dispensing")
}

func (n loggingObserver) ResetedTo(duration time.Duration) {
	log.Printf("reseting duration to %s", duration.String())
}

type unixObserverForwarer struct {
	*http.Transport
	*http.Client
}

func NewUnixObserverForwarer(path string) *unixObserverForwarer {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("unix", path)
		},
	}
	client := &http.Client{
		Transport: transport,
	}
	return &unixObserverForwarer{
		Transport: transport,
		Client:    client,
	}
}

func (forwarder *unixObserverForwarer) Notified(time.Duration) {
	req, err := http.NewRequest(http.MethodPost, "http://socket.unix/notify", nil)
	if err != nil {
		log.Printf("notified request error: %v", err)
		return
	}
	resp, err := forwarder.Do(req)
	if err != nil {
		log.Printf("notified response error: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Printf("notified resposne status code not 200: %d", resp.StatusCode)
	}
}

func (forwarder *unixObserverForwarer) ResetedTo(duration time.Duration) {
	var body struct {
		Duration uint64 `json:"duration"`
	}
	body.Duration = uint64(duration.Seconds())

	b, err := json.Marshal(body)
	if err != nil {
		log.Printf("mashaling: %v", err)
		return
	}
	req, err := http.NewRequest(http.MethodPost, "http://socket.unix/resetTo", bytes.NewBuffer(b))
	if err != nil {
		log.Printf("notified request error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := forwarder.Do(req)
	if err != nil {
		log.Printf("notified response error: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Printf("notified resposne status code not 200: %d", resp.StatusCode)
	}
}

var (
	_ Observer = loggingObserver{}
	_ Observer = (*unixObserverForwarer)(nil)
)
