package main

import (
	"log"

	"github.com/crmne/mqtt-alive-daemon/pkg/mqttalive"
)

func main() {
	if err := mqttalive.Run(); err != nil {
		log.Fatal(err)
	}
}
