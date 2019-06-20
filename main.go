package main

import (
	"flag"
	"fmt"
)

func main() {
	device := flag.String("device", "/dev/ttyUSB0", "IR read/write head")
	flag.Parse()
	fmt.Println("Trying connecting to", *device)

}
