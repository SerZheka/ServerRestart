package output

import (
	"log"
	"slices"

	"github.com/serzheka/serverrestart/config"
	"github.com/serzheka/serverrestart/util"
)

func Example(linkConf *config.LinkMethods, output <-chan util.InOutMessage) {
	for message := range output {
		if (message.LinkMethod == nil || message.LinkMethod == linkConf) &&
			(message.Server == "" || slices.ContainsFunc(linkConf.ServerCommands,
				func(serv config.ServerCommand) bool { return serv.Server == message.Server })) {
			log.Printf("Example: For %s received %s\n", message.Server, message.Message)
		}
	}
}
