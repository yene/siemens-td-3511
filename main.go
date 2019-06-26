package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/tarm/serial"
)

func main() {
	device := flag.String("device", "/dev/ttyUSB0", "IR read/write head")
	flag.Parse()
	fmt.Println("Trying connecting to", *device)
	config := &serial.Config{
		Name:        *device,
		Baud:        300,
		ReadTimeout: 1,
		Size:        7,
		Parity:      serial.ParityEven,
		StopBits:    serial.Stop1,
	}
	s, err := serial.OpenPort(config)
	if err != nil {
		log.Println("Could not open port.")
		log.Fatal(err)
	}

	// sending inital sequence
	_, err = s.Write([]byte("/?!\r\n"))
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)

	reader := bufio.NewReader(s)
	for {
		reply, err := reader.ReadBytes('\n')
		if err != nil { // At the end, err will equal io.EOF
			if err != io.EOF {
				log.Println(err)
			}
			break
		}
		fmt.Println(string(reply))
		fmt.Printf("% x", reply)
	}

	// requesting baud rate
	// \x06050\r\n = 9600
	// \x06060\r\n = 19200
	_, err = s.Write([]byte("\x06060\r\n"))
	if err != nil {
		log.Fatal(err)
	}
	s.Close()

	time.Sleep(500 * time.Millisecond)

	config = &serial.Config{
		Name:        *device,
		Baud:        19200, // should match the requested baud rate
		ReadTimeout: 1,
		Size:        7,
		Parity:      serial.ParityEven,
		StopBits:    serial.Stop1,
	}
	s, err = serial.OpenPort(config)
	if err != nil {
		log.Println("Could not open port.")
		log.Fatal(err)
	}

	reader = bufio.NewReader(s)
	for {
		reply, err := reader.ReadBytes('\n')
		if err != nil { // At the end, err will equal io.EOF
			if err != io.EOF {
				log.Println(err)
			}
			break
		}
		// fmt.Println(reply)
		fmt.Println(string(reply))
	}

	s.Close()
}
