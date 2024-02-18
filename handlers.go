package dispenserd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/OmarTariq612/dispenserd/config"
	"golang.org/x/exp/slices"
	"tailscale.com/types/key"
)

func (s *server) registerHandlers() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /{$}", http.HandlerFunc(s.Dashboard))
	mux.Handle("GET /ping", http.HandlerFunc(s.Pong))
	mux.Handle("GET /duration", http.HandlerFunc(s.Duration))
	mux.Handle("POST /setDuration/self", http.HandlerFunc(s.SetDurationSelf))
	mux.Handle("POST /setDuration/other", http.HandlerFunc(s.SetDurationOther))

	return mux
}

func (s *server) Dashboard(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ReadDevices(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "{\"error\": \"%v\"}", err)
		return
	}
	if err := json.NewEncoder(w).Encode(s.devices); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "{\"error\": \"%v\"}", err)
	}
}

func (s *server) Pong(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Pong!")
}

func (s *server) Duration(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cfg == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	fmt.Fprintf(w, "{\"duration\": \"%v\"}", s.cfg.Duration)
}

func (s *server) SetDurationSelf(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cfg == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var duration struct {
		Duration string `json:"duration"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&duration); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "{\"error\": \"%v\"}", err.Error())
		return
	}

	d, err := time.ParseDuration(duration.Duration)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "{\"error\": \"%v is not a valid duration\"}", duration.Duration)
		return
	}

	s.cfg.Duration = d
	s.cfg.StrDuration = duration.Duration
	s.notifyDuration <- d

	w.WriteHeader(http.StatusOK)
}

func (s *server) SetDurationOther(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var durationOther struct {
		Duration  string         `json:"duration"`
		PublicKey key.NodePublic `json:"public_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&durationOther); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "{\"error\": \"%v\"}", err.Error())
		return
	}

	d, err := time.ParseDuration(durationOther.Duration)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "{\"error\": \"%v is not a valid duration\"}", durationOther.Duration)
		return
	}

	index := slices.IndexFunc(s.devices, func(d Device) bool {
		return d.PeerStatus.PublicKey == durationOther.PublicKey
	})

	if index == -1 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "{\"error\": \"%s is not a public key for a device known at this moment\"}", durationOther.PublicKey)
		return
	}

	buf := bytes.NewBuffer([]byte(fmt.Sprintf("{\"duration\": \"%s\"}", durationOther.Duration)))
	resp, err := http.Post(fmt.Sprintf("http://%s:%d/setDuration/self", s.devices[index].PeerStatus.TailscaleIPs[0], config.Port), "application/json", buf)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	s.devices[index].Duration = d

	w.WriteHeader(http.StatusOK)
}
