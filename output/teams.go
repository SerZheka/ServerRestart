package output

import (
	"bytes"
	"log"
	"net/http"
	"slices"

	"github.com/serzheka/serverrestart/config"
	"github.com/serzheka/serverrestart/util"
)

func Teams(linkConf *config.LinkMethods, output <-chan util.InOutMessage) {
	messageStart := `{ "type": "message", "attachments": [ { "contentType": "application/vnd.microsoft.card.adaptive", "contentUrl": null, "content": { "$schema": "http://adaptivecards.io/schemas/adaptive-card.json", "type": "AdaptiveCard", "version": "1.2", "body": [ { "type": "TextBlock", "text": "`

	for message := range output {
		if (message.LinkMethod == nil || message.LinkMethod == linkConf) &&
			(message.Server == "" || slices.Contains(linkConf.Servers, message.Server)) {
			log.Printf("Teams: For %s received %s\n", message.Server, message.Message)

			sendMessage := messageStart + message.Message + `" } ] } } ] }`
			_, err := http.Post(linkConf.Key, "application/json", bytes.NewBufferString(sendMessage))
			if err != nil {
				log.Printf("Teams: Error sending message to %s: %v\n", message.Server, err)
			}
		}
	}
}
