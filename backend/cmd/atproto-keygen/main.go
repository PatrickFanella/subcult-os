package main

import (
	"fmt"
	"log"

	atprotocol "git.subcult.tv/PatrickFanella/subcult-os/internal/atproto"
)

func main() {
	privateKey, err := atprotocol.GenerateOAuthClientPrivateKey()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(privateKey)
}
