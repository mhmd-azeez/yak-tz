// Note: run `go doc -all` in this package to see all of the types and functions available.
// ./pdk.gen.go contains the domain types from the host where your plugin will run.
package main

import (
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/extism/go-pdk"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func doStuff() {
	// Define the time format
	const layout = "15:04:05"

	// Define the time string
	timeString := "14:00:00"

	// Load the location
	location, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		fmt.Println("Error loading location:", err)
		return
	}

	// Parse the time string in the specified location
	parsedTime, err := time.ParseInLocation(layout, timeString, location)
	if err != nil {
		fmt.Println("Error parsing time:", err)
		return
	}

	// Print the parsed time
	pdk.Log(pdk.LogInfo, fmt.Sprintf("Parsed time: %v", parsedTime))
}

func handleMessage(input Message) (Message, error) {
	doStuff()
	if input.Type != "text" {
		return Reply("I can only handle text messages"), nil
	}

	sourceTime, destTime, err := parseInput(input.Body)
	if err != nil {
		return Reply(err.Error()), nil
	}

	// Format the response
	response := formatResponse(sourceTime, destTime)

	return Reply(response), nil
}

func parseInput(input string) (time.Time, time.Time, error) {

	input = strings.TrimSpace(input)

	parts := strings.Fields(input)
	if len(parts) < 4 {
		examples := strings.Join([]string{
			"14:20 GMT in CET",
			"04:20 PM GMT in UTC-8",
			"04:20 PM UTC+5 in GMT",
		}, "\n")

		return time.Time{}, time.Time{}, fmt.Errorf("invalid format: expected /tz [time] [AM/PM] [source timezone] in [destination timezone]. examples:\n%s", examples)
	}

	// Parse the input time
	idx := 0
	timePart := parts[idx]
	idx++

	if len(parts) >= 5 {
		timePart += " " + parts[idx]

		idx++
	}

	fromTZ, err := loadLocation(parts[idx])
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid source timezone: %v", err)
	}
	pdk.Log(pdk.LogInfo, fmt.Sprintf("fromTZ: %v", fromTZ))
	idx++

	var sourceTime time.Time
	if len(parts) >= 5 {
		sourceTime, err = time.ParseInLocation("3:04 PM", timePart, fromTZ)
	} else {
		sourceTime, err = time.ParseInLocation("15:04", timePart, fromTZ)
	}

	s, _ := sourceTime.Zone()
	pdk.Log(pdk.LogInfo, fmt.Sprintf("sourceTime: %v %v", sourceTime, s))

	idx++ // skip "in"

	toTZ, err := loadLocation(parts[idx])
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid destination timezone: %v", err)
	}

	destTime := sourceTime.In(toTZ)

	return sourceTime, destTime, nil
}

func Reply(body string) Message {
	nick := "bot"
	return Message{
		Body: body,
		Type: string(Html),
		Nick: &nick,
	}
}

func loadLocation(name string) (*time.Location, error) {
	commonAbbreviations := map[string]string{
		"EST":  "America/New_York",
		"EDT":  "America/New_York",
		"CT":   "America/Chicago",
		"CST":  "America/Chicago",
		"CDT":  "America/Chicago",
		"MST":  "America/Denver",
		"MDT":  "America/Denver",
		"PST":  "America/Los_Angeles",
		"PDT":  "America/Los_Angeles",
		"GMT":  "Europe/London",
		"BST":  "Europe/London",
		"CET":  "Europe/Paris",
		"CEST": "Europe/Paris",
		"IST":  "Asia/Kolkata",
		"JST":  "Asia/Tokyo",
		"AEST": "Australia/Sydney",
		"AEDT": "Australia/Sydney",
	}

	name = strings.TrimSpace(name)

	if ianaName, ok := commonAbbreviations[strings.ToUpper(name)]; ok {
		name = ianaName
	}

	return time.LoadLocation(name)
}

func formatResponse(sourceTime, destTime time.Time) string {
	// Initialize a message printer for formatting
	p := message.NewPrinter(language.English)

	// Get the UTC offsets
	s, sourceOffset := sourceTime.Zone()
	d, destOffset := destTime.Zone()

	// Format the response
	return p.Sprintf("%s %s (UTC%s) is <b>%s</b> in %s (UTC%s)",
		sourceTime.Format("3:04 PM"),
		s,
		formatOffset(sourceOffset),
		destTime.Format("3:04 PM"),
		d,
		formatOffset(destOffset))
}

func formatOffset(offsetSeconds int) string {
	offset := time.Duration(offsetSeconds) * time.Second
	hours := int(offset.Hours())
	minutes := int(offset.Minutes()) % 60

	sign := "+"
	if hours < 0 {
		sign = "-"
		hours = -hours
		minutes = -minutes
	}

	return fmt.Sprintf("%s%02d:%02d", sign, hours, minutes)
}
