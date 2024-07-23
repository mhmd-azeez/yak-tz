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

func handleMessage(input Message) (Message, error) {
	if !strings.HasPrefix(input.Body, "/tz") {
		input.Body = "/tz " + input.Body
	}

	if input.Type != "text" {
		return Reply("I can only handle text messages"), nil
	}

	// Split the input into parts
	parts := strings.Fields(input.Body)
	if len(parts) < 6 {
		return Reply("invalid format: expected /tz [time] [AM/PM] [source timezone] in [destination timezone]"), nil
	}

	// Parse the input time
	inputTime, err := time.Parse("3:04 PM", fmt.Sprintf("%s %s", parts[1], parts[2]))
	if err != nil {
		return Reply(fmt.Sprintf("invalid time format: %v", err)), nil
	}

	// Get source and destination time zones
	sourceTimezone := strings.Join(parts[3:len(parts)-2], " ")
	destTimezone := parts[len(parts)-1]

	// Load source time zone
	sourceLocation, err := loadLocation(sourceTimezone)
	if err != nil {
		return Reply(fmt.Sprintf("invalid source timezone: %v", err)), nil
	}

	// Load destination time zone
	destLocation, err := loadLocation(destTimezone)
	if err != nil {
		return Reply(fmt.Sprintf("invalid destination timezone: %v", err)), nil
	}

	// Set the time in the source time zone
	sourceTime := time.Date(time.Now().Year(), inputTime.Month(), inputTime.Day(), inputTime.Hour(), inputTime.Minute(), 0, 0, sourceLocation)

	pdk.Log(pdk.LogInfo, fmt.Sprintf("sourceTime: %v", sourceTime))

	// Convert to destination time zone
	destTime := sourceTime.In(destLocation)

	// Format the response
	response := formatResponse(sourceTime, destTime, sourceTimezone, destTimezone)

	return Reply(response), nil
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

func formatResponse(sourceTime, destTime time.Time, sourceTimezone, destTimezone string) string {
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
