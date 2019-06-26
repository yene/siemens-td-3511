package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	client "github.com/influxdata/influxdb1-client/v2"
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
				log.Println(reply, string(reply))
				log.Println(err)
			}
			break
		}
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

	type Value struct {
		Channel string
		Unit    string
		Value   float64
	}
	var values []Value
	reader = bufio.NewReader(s)
	for {
		reply, err := reader.ReadBytes('\n')
		if err != nil { // At the end, err will equal io.EOF
			if err != io.EOF {
				log.Println(err)
			}
			break
		}
		line := string(reply)
		line = strings.Replace(line, "\n", "", -1)
		line = strings.TrimSpace(line)
		if strings.Contains(line, "0.000*") || strings.Contains(line, "0.0*") {
			continue
		}
		if strings.HasSuffix(line, "*kW)") || strings.HasSuffix(line, "*kWh)") {
			p := strings.Split(line, "(")
			channel := p[0]
			v := strings.Split(p[1], "*")

			var value float64
			if value, err = strconv.ParseFloat(v[0], 32); err != nil {
				log.Println(err)
				continue
			}
			unit := strings.TrimRight(v[1], ")")
			fmt.Println(channel, value, unit)
			values = append(values, Value{
				Channel: channel,
				Unit:    unit,
				Value:   value,
			})
		}
	}
	s.Close()

	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: "http://localhost:8086",
	})
	if err != nil {
		fmt.Println("Error creating InfluxDB Client: ", err.Error())
	}
	defer c.Close()

	q := client.NewQuery("CREATE DATABASE data", "", "")
	if response, err := c.Query(q); err == nil && response.Error() == nil {
	}

	bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  "data",
		Precision: "s",
	})

	for _, v := range values {
		tags := map[string]string{
			"channel": v.Channel,
			"unit":    v.Unit,
		}
		fields := map[string]interface{}{
			"value": v.Value,
		}
		pt, err := client.NewPoint("energy", tags, fields, time.Now())
		if err != nil {
			fmt.Println("Error: ", err.Error())
		}
		bp.AddPoint(pt)
	}

	err = c.Write(bp)
	if err != nil {
		log.Fatal(err)
	}
}
