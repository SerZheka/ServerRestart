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
			(len(linkConf.Servers) == 0 || slices.Contains(linkConf.Servers, message.Server)) {
			log.Printf("Example: For %s received %s\n", message.Server, message.Message)
		}
	}
}
