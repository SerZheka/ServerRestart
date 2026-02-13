package main

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	packdb "github.com/serzheka/serverrestart/db"
	"github.com/serzheka/serverrestart/util"
)

var timeRegex = regexp.MustCompile(`(\d{1,2}:\d{1,2})|\+(\d+)(?:\/(5|10))?`)

func processInput(input <-chan util.InOutMessage, db *packdb.DB, output []chan<- util.InOutMessage) {
	for inputmsg := range input {
		log.Println("processing inputmsg", inputmsg.Message)
		values := strings.Split(inputmsg.Message, ";")
		if len(values) != 3 {
			log.Println("wrong size for values", values)
			for _, o := range output {
				o <- util.InOutMessage{
					Message:    "Wrong size for input message",
					LinkMethod: inputmsg.LinkMethod,
				}
			}
			continue
		}
		timeString, timeMinutes, err := parseTime(values[2])
		if err != nil {
			log.Println("error parsing time:", err)
			for _, o := range output {
				o <- util.InOutMessage{
					Message:    "Error parsing time",
					LinkMethod: inputmsg.LinkMethod,
				}
			}
			continue
		}

		err = db.Add(packdb.Restart{
			Server:  values[1],
			Command: strings.ToLower(values[0]),
			Time:    timeMinutes,
		})
		if err != nil {
			log.Println("error adding to db:", err)
			for _, o := range output {
				o <- util.InOutMessage{
					Message:    "Error adding to db",
					LinkMethod: inputmsg.LinkMethod,
				}
			}
			continue
		}

		msg := fmt.Sprintf("Planned %s for server %s at %v", values[0], values[1], timeString)
		log.Println(msg)
		for _, o := range output {
			o <- util.InOutMessage{
				Message: msg,
				Server:  values[1],
			}
		}
	}
}

func parseTime(possiblyTime string) (string, uint16, error) {
	groups := timeRegex.FindStringSubmatch(possiblyTime)
	if len(groups) == 0 {
		return "", 0, errors.New("string do not match regex: " + possiblyTime)
	}

	if groups[1] != "" {
		parsed, err := time.Parse("15:4", groups[1])
		if err != nil {
			return "", 0, err
		}
		return parsed.Format("15:04"), uint16(parsed.Hour()*60 + parsed.Minute()), nil
	}

	minToAdd, err := strconv.Atoi(groups[2])
	if err != nil {
		return "", 0, err
	}
	result := time.Now().Add(time.Duration(minToAdd) * time.Minute)

	if groups[3] != "" {
		minRound, err := strconv.Atoi(groups[3])
		if err != nil {
			return "", 0, err
		}
		if result.Minute()%minRound != 0 {
			result = result.Truncate(time.Duration(minRound) * time.Minute).Add(time.Duration(minRound) * time.Minute)
		}
	}

	return result.Format("15:04"), uint16(result.Hour()*60 + result.Minute()), nil
}
