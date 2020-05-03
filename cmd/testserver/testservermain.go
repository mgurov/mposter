package main

import (
	"flag"
	"fmt"

	"github.com/mgurov/mposter/internal/testserver"
)

func main() {

	portFlag := flag.String("bind", ":0", "binding to listen at. e.g. :8080 or localhost:80")
	flag.Parse()

	server := testserver.StartNewTestServerBinding(*portFlag)

	fmt.Printf("Listening on port %s. Press the Enter Key to terminate.\n", server.Addr())
	fmt.Scanln() // wait for Enter Key

}
