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
	log.Println("got restarts", restarts)
	var wg sync.WaitGroup
	now := uint16(timeNow.Hour()*60 + timeNow.Minute())
	for _, restart := range restarts {
		if restart.Time == now {
			db.Lock(restart.Server)
			wg.Go(func() {
				runScript(&restart, outchans)
				db.DeleteWithLocked(restart.Server)
			})
		} else if !util.CheckTime(restart.Time, now) {
			message := fmt.Sprintf("Skipped %s for %s at %v", restart.Command, restart.Server, restart.Time)
			log.Println(message)
			for _, outchan := range outchans {
				outchan <- util.InOutMessage{
					Message: message,
					Server:  restart.Server,
					ChatId:  restart.ChatId,
				}
			}

			db.DeleteWithLocked(restart.Server)
		}
	}

	wg.Wait()
	log.Println("finished processing restarts from", timeNow.Format("15:04"))
}
