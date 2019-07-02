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
	// fmt.Println("Trying connecting to", *device)
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

	time.Sleep(200 * time.Millisecond) // This sleep is documented and required.

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
	var totals []Value
	type Month struct {
		Channel string
		Value   float64
		Time    time.Time
	}
	var months []Month
	var totalPreviousMonth float64
	dates := make(map[string]time.Time)
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

		// My 1.4.0 seems to report a weird value.
		if strings.Contains(line, "0.000*") || strings.Contains(line, "0.0*") || strings.HasPrefix(line, "1.4.0(") {
			continue
		}
		// date entries: 0.1.2*30(19-04-01 00:00)
		if strings.HasPrefix(line, "0.1.2*") {
			p := strings.Split(line, "*")
			v := strings.Split(p[1], "(")
			id := v[0]
			datestring := strings.TrimRight(v[1], ")")
			layout := "06-01-02 15:04"
			t, err := time.Parse(layout, datestring)
			if err != nil {
				log.Println("Failed to parse month date", err)
				continue
			}
			// Subtract one month: because the total gets recorded at the beginning of a new month.
			dates[id] = t.AddDate(0, -1, 0)
			continue
		}

		// monthly kWh entries
		if strings.HasPrefix(line, "1.8.0*") || strings.HasPrefix(line, "1.8.1*") || strings.HasPrefix(line, "1.8.2*") {
			p := strings.Split(line, "*")
			channel := p[0]
			v := strings.Split(p[1], "(")
			id := v[0]
			valuestring := strings.TrimRight(v[1], ")")

			var value float64
			if value, err = strconv.ParseFloat(valuestring, 64); err != nil {
				log.Println(err)
				continue
			}
			t, ok := dates[id]
			if !ok {
				log.Println("No date found for", id)
				continue
			}
			// fmt.Println(t.Format("Jan 2006"), channel, value, "kWh")
			months = append(months, Month{
				Channel: channel,
				Value:   value,
				Time:    t,
			})
			if value > totalPreviousMonth {
				totalPreviousMonth = value
			}
		}

		// Current Consumption
		if strings.HasPrefix(line, "1.7.0(") {
			p := strings.Split(line, "(")
			channel := p[0]
			v := strings.Split(p[1], "*")

			var value float64
			if value, err = strconv.ParseFloat(v[0], 64); err != nil {
				log.Println(err)
				continue
			}
			unit := strings.TrimRight(v[1], ")")
			totals = append(totals, Value{
				Channel: channel,
				Unit:    unit,
				Value:   value,
			})
		}

		// Current Total Consumption
		// "1.8.0(" Is so that it does not match monthly records.
		if strings.HasPrefix(line, "1.8.0(") {
			p := strings.Split(line, "(")
			channel := p[0]
			v := strings.Split(p[1], "*")

			var value float64
			if value, err = strconv.ParseFloat(v[0], 64); err != nil {
				log.Println(err)
				continue
			}
			unit := strings.TrimRight(v[1], ")")
			totals = append(totals, Value{
				Channel: channel,
				Unit:    unit,
				Value:   value,
			})
			totals = append(totals, Value{
				Channel: "ongoing",
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

	for _, v := range totals {
		if v.Channel == "ongoing" { // Subtract previous month of the Energy total.
			v.Value = v.Value - totalPreviousMonth
		}
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

	// The monthly recorded data always contains the total, we store the diff.
	var previousMonth float64
	for i := len(months) - 1; i >= 0; i-- {
		v := months[i]

		// Every time the channel changes we reset previousMonth.
		if (i+1)%len(dates) == 0 {
			previousMonth = v.Value
			continue
		}

		diff := v.Value - previousMonth
		previousMonth = v.Value

		tags := map[string]string{
			"channel": v.Channel,
			"unit":    "kWh",
		}
		fields := map[string]interface{}{
			"value": diff,
		}
		pt, err := client.NewPoint("months", tags, fields, v.Time)

		// fmt.Println(v.Time.Format("Jan 2006"), v.Channel, v.Value, diff)
		if err != nil {
			fmt.Println("Error: ", err.Error())
			continue
		}
		bp.AddPoint(pt)
	}

	err = c.Write(bp)
	if err != nil {
		log.Fatal(err)
	}
}
