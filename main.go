// Note: run `go doc -all` in this package to see all of the types and functions available.
// ./pdk.gen.go contains the domain types from the host where your plugin will run.
package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	_ "time/tzdata"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func handleMessage(input Message) (Message, error) {
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
	// black magic
	var re = regexp.MustCompile(`(?:(\d{4}-\d{2}-\d{2})\s)?(\d{1,2}:\d{2})\s?(AM|PM)?\s?([A-Z]+[\+\-]?\d*)\s?in\s?([A-Z]+[\+\-]?\d*)`)
	// Find the match
	match := re.FindStringSubmatch(input)
	if match == nil {
		return time.Time{}, time.Time{}, fmt.Errorf("I couldn't understand the input")
	}

	sourceDate := match[1]
	if sourceDate == "" {
		sourceDate = time.Now().Format("2006-01-02")
	}

	sourceTime := match[2]
	sourceMeridian := match[3]
	sourceTZ := match[4]
	sourceLocation, err := loadLocation(sourceTZ)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("I couldn't understand the source timezone: %s", sourceTZ)
	}

	destTZ := match[5]
	destLocation, err := loadLocation(destTZ)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("I couldn't understand the destination timezone: %s", destTZ)
	}

	// Parse the source time
	sourceDateTime := strings.TrimSpace(sourceDate + " " + sourceTime + " " + strings.ToUpper(sourceMeridian))

	parsedTime, err := time.ParseInLocation("2006-01-02 3:04 PM", sourceDateTime, sourceLocation)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("I couldn't understand the source time: %s", sourceDateTime)
	}

	// Convert the time to the destination timezone
	destTime := parsedTime.In(destLocation)

	return parsedTime, destTime, nil
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
