package main

import (
	"fmt"
	"log"

	"github.com/honeycombio/otel-config-go/otelconfig"
)

func main() {
	otelShutdown, err := otelconfig.ConfigureOpenTelemetry()
	if err != nil {
		log.Fatalf("error setting up OTel SDK - %e", err)
	}
	defer otelShutdown()

	s := newServer()
	fmt.Println("Starting server")
	if err := s.ListenAndServe(); err != nil {
		panic(err)
	}
}
