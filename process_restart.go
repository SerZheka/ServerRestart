package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	packdb "github.com/serzheka/serverrestart/db"
	"github.com/serzheka/serverrestart/util"
)

func processRestart(db *packdb.DB, outchans []chan<- util.InOutMessage) {
	timeNow := time.Now()
	log.Println("started processing restarts at", timeNow.Format("15:04"))

	restarts := db.Select()

	var wg sync.WaitGroup
	var processed []string
	now := uint16(timeNow.Hour()*60 + timeNow.Minute())
	for _, restart := range restarts {
		if restart.Time == now {
			wg.Go(func() {
				runScript(&restart, outchans)
			})

			processed = append(processed, restart.Server)
		} else if restart.Time < now {
			message := fmt.Sprintf("Skipped %s for %s at %v", restart.Command, restart.Server, restart.Time)
			log.Println(message, "cause before now")
			for _, outchan := range outchans {
				outchan <- util.InOutMessage{
					Message: message,
					Server:  restart.Server,
				}
			}

			processed = append(processed, restart.Server)
		} else if restart.Time-now > 60 {
			message := fmt.Sprintf("Skipped %s for %s at %v", restart.Command, restart.Server, restart.Time)
			log.Println(message, "cause more than hour before command")
			for _, outchan := range outchans {
				outchan <- util.InOutMessage{
					Message: message,
					Server:  restart.Server,
				}
			}

			processed = append(processed, restart.Server)
		}
	}

	wg.Wait()
	log.Println("scripts finished, deleting processed")
	for _, processedServer := range processed {
		err := db.Delete(processedServer)
		if err != nil {
			log.Println("error deleting from db", err)
		}
	}
}
