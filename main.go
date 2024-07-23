// Note: run `go doc -all` in this package to see all of the types and functions available.
// ./pdk.gen.go contains the domain types from the host where your plugin will run.
package main

import (
	"fmt"
	"regexp"
	"strconv"
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
	// black magic, don't touch
	var re = regexp.MustCompile(`(?:([0-9]{4}-[0-9]{2}-[0-9]{2})[ 	])?([0-9]{1,2}:[0-9]{2})[ 	]?(AM|PM)?[ 	]?([A-Z]+[\+\-]?[0-9]*)[ 	]?[^ 	]+[ 	]?([A-Z]+[\+\-]?[0-9]*)`)

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

	var parsedTime time.Time
	if sourceMeridian == "" {
		parsedTime, err = time.ParseInLocation("2006-01-02 15:04", sourceDateTime, sourceLocation)
	} else {
		parsedTime, err = time.ParseInLocation("2006-01-02 3:04 PM", sourceDateTime, sourceLocation)
	}

	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("I couldn't understand the source time: %s", sourceDateTime)
	}

	destTime := parsedTime.In(destLocation)

	return parsedTime, destTime, nil
}

// parseOffset converts a string like "UTC+4" or "UTC-5" to a *time.Location
func parseOffset(offsetStr string) (*time.Location, error) {
	// Remove "UTC" prefix if present
	if strings.HasPrefix(offsetStr, "UTC") {
		offsetStr = strings.TrimPrefix(offsetStr, "UTC")
	}

	// Parse the offset
	offsetStr = strings.TrimSpace(offsetStr)
	sign := 1
	if strings.HasPrefix(offsetStr, "-") {
		sign = -1
		offsetStr = strings.TrimPrefix(offsetStr, "-")
	} else if strings.HasPrefix(offsetStr, "+") {
		offsetStr = strings.TrimPrefix(offsetStr, "+")
	}

	// Parse the offset hours and minutes
	offsetParts := strings.Split(offsetStr, ":")
	hours, err := strconv.Atoi(offsetParts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid hours in offset: %v", err)
	}
	minutes := 0
	if len(offsetParts) > 1 {
		minutes, err = strconv.Atoi(offsetParts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid minutes in offset: %v", err)
		}
	}

	// Calculate total offset in seconds
	offsetSeconds := (hours*3600 + minutes*60) * sign

	// Create and return the fixed zone
	return time.FixedZone(fmt.Sprintf("UTC%+d", offsetSeconds/3600), offsetSeconds), nil
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

	if strings.HasPrefix(name, "UTC") {
		return parseOffset(name)
	}

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
