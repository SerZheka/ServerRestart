package util

import (
	"fmt"

	"github.com/serzheka/serverrestart/config"
	packdb "github.com/serzheka/serverrestart/db"
)

type InOutMessage struct {
	Message    string
	Server     string
	ChatId     int64
	LinkMethod *config.LinkMethods
}

func SendErrMessages(output []chan<- InOutMessage, restartInfo *packdb.Restart) {
	message := fmt.Sprintf("Error processing %s for server %s. Please see server logs", restartInfo.Command, restartInfo.Server)
	for _, outchan := range output {
		outchan <- InOutMessage{
			Message: message,
			Server:  restartInfo.Server,
			ChatId:  restartInfo.ChatId,
		}
	}
}
