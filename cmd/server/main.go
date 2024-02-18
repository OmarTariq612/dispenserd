package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/OmarTariq612/dispenserd"
	"github.com/OmarTariq612/dispenserd/config"
)

func main() {
	configPath := flag.String("config", "", "path to json config file")
	sleepDuration := flag.Duration("sleep", 0, "sleep")
	custom := flag.Bool("custom", false, "custom runUp")
	unixPath := flag.String("unix", "", "unix socket path")
	flag.Parse()

	if *configPath == "" {
		log.Fatalln("config path must be specified")
	}
	if *unixPath == "" {
		log.Fatalln("unix socket path must be specified")
	}
	if *sleepDuration != 0 {
		time.Sleep(*sleepDuration)
	}

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("config load: %v", err)
	}
	if err = cfg.Complete(); err != nil {
		log.Fatalf("config complete: %v", err)
	}
	s, err := dispenserd.NewServer(context.Background(), cfg, *custom, dispenserd.NewLoggingObserver(), dispenserd.NewUnixObserverForwarer(*unixPath))
	if err != nil {
		log.Fatalf("new server: %v", err)
	}
	if err = s.ListenAndServe(); err != nil {
		log.Fatalf("server listen: %v", err)
	}
}
