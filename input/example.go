package input

import (
	"context"
	"log"
	"time"

	"github.com/serzheka/serverrestart/config"
	"github.com/serzheka/serverrestart/util"
)

func Example(ctx context.Context, linkConf *config.LinkMethods, output chan<- util.InOutMessage) {
	log.Println("sending restart request for", linkConf.ServerCommands[0].Server)
	restartTime := time.Now().Add(time.Minute).Format("15:04")
	output <- util.InOutMessage{
		Message:    "restart;" + linkConf.ServerCommands[0].Server + ";" + restartTime,
		LinkMethod: linkConf,
	}
}
