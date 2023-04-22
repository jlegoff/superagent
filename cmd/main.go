package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"superagent/meta"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "c", "/etc/newrelic/meta.yaml", "path of the meta agent config file")
	flag.Parse()
	metaAgent, err := meta.NewMetaAgent(configPath)
	if err != nil {
		fmt.Printf("Error starting the meta agent %s", err)
		os.Exit(1)
	}
	err = metaAgent.Start()
	if err != nil {
		fmt.Printf("Error starting the meta agent %s", err)
		os.Exit(1)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	err = metaAgent.Stop()
	if err != nil {
		fmt.Printf("Error shutting down meta agent %s", err)
	}
}
