package main

import (
	"log"

	nat "github.com/fd/go-nat"
)

func main() {
	nat, err := nat.DiscoverGateway()
	if err != nil {
		log.Fatalf("error: %s", err)
	}
	log.Printf("nat type: %s", nat.Type())
}
