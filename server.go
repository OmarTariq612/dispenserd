package dispenserd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/OmarTariq612/dispenserd/config"
	"github.com/OmarTariq612/dispenserd/tail"
	"golang.org/x/exp/slices"
	"tailscale.com/client/tailscale"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/types/views"
)

type server struct {
	Device
	devices        []Device // other devices
	mu             sync.Mutex
	ctx            context.Context
	cfg            *config.Config
	localClient    *tailscale.LocalClient
	notifyDuration chan time.Duration

	observers []Observer

	custom bool
}

func (s *server) startDispenser() {
	duration := <-s.notifyDuration
	ticker := time.NewTicker(duration)
	for {
		select {
		case <-ticker.C:
			for _, observer := range s.observers {
				observer.Notified(duration)
			}
		case duration = <-s.notifyDuration:
			for _, observer := range s.observers {
				observer.ResetedTo(duration)
			}
			ticker.Reset(duration)
		case <-s.ctx.Done():
			break
		}
	}
}

func NewServer(ctx context.Context, cfg *config.Config, custom bool, observers ...Observer) (*server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config must be specified")
	}
	if err := cfg.Complete(); err != nil {
		return nil, err
	}
	return &server{ctx: ctx,
		cfg: cfg,
		Device: Device{
			Duration: cfg.Duration,
		},
		devices:        make([]Device, 0),
		notifyDuration: make(chan time.Duration, 1),
		custom:         custom,
		observers:      observers,
	}, nil
}

func (s *server) startConnection() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cfg == nil {
		return fmt.Errorf("config is nil")
	}
	s.localClient = &tailscale.LocalClient{}

	runUpFunc := tail.RunUp
	if s.custom {
		runUpFunc = tail.RunUpCustom
	}
	if err := runUpFunc(s.ctx, s.localClient, s.cfg.AuthKey); err != nil {
		return err
	}
	s.notifyDuration <- s.cfg.Duration
	go s.startDispenser()
	return nil
}

func (s *server) ReadDevices() error {
	status := tail.RunStatus(s.ctx, s.localClient)
	s.PeerStatus = status.Self
	tag := status.Self.Tags.At(0)

	var duration struct {
		Duration string `json:"duration"`
	}

	for peerPublicKey, peerStatus := range status.Peer {
		if peerStatus.Tags != nil && views.SliceContains(*peerStatus.Tags, tag) && peerStatus.Online {
			if index := slices.IndexFunc(s.devices, func(d Device) bool {
				return d.PeerStatus.PublicKey == peerPublicKey
			}); index != -1 {
				s.devices[index].PeerStatus = peerStatus
				continue
			}

			resp, err := http.Get(fmt.Sprintf("http://%s:%d/duration", peerStatus.TailscaleIPs[0], config.Port))
			if err != nil {
				log.Println(err)
				continue
			}
			defer resp.Body.Close()
			decoder := json.NewDecoder(resp.Body)
			decoder.DisallowUnknownFields()
			if err = decoder.Decode(&duration); err != nil {
				log.Println(err)
				continue
			}
			dur, err := time.ParseDuration(duration.Duration)
			if err != nil {
				log.Println(err)
				continue
			}
			s.devices = append(s.devices, Device{PeerStatus: peerStatus, Duration: dur})
		}
	}

	return nil
}

func (s *server) ListenAndServe() error {
	if err := s.startConnection(); err != nil {
		log.Println(err)
	}
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Port))
	if err != nil {
		return err
	}
	fmt.Println("listening")

	mux := s.registerHandlers()

	return http.Serve(ln, mux)
}

type Device struct {
	Duration   time.Duration        `json:"duration"`
	PeerStatus *ipnstate.PeerStatus `json:"peer_status"`
}
