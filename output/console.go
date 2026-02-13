package output

import (
	"log"

	"github.com/serzheka/serverrestart/config"
	"github.com/serzheka/serverrestart/util"
)

func Console(linkConf *config.LinkMethods, output <-chan util.InOutMessage) {
	for message := range output {
		log.Printf("Console: for %s received %s\n", message.Server, message.Message)
	}
}
