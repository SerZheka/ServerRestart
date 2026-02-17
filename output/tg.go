package output

import (
	"context"
	"log"
	"slices"

	"github.com/go-telegram/bot"
	"github.com/serzheka/serverrestart/config"
	"github.com/serzheka/serverrestart/util"
)

func Tg(linkConf *config.LinkMethods, output <-chan util.InOutMessage) {
	log.Println("starting tg output for servers", linkConf.Servers)
	tgbot, err := bot.New(linkConf.Key)
	if err != nil {
		log.Println("error creating tg bot", err)
		return
	}
	botName, err := tgbot.GetMyName(context.TODO(), nil)
	if err != nil {
		log.Println("error getting bot name", err)
		return
	}
	log.Println("authorized on account", botName.Name)

	for message := range output {
		if (message.LinkMethod == nil || message.LinkMethod == linkConf) &&
			(message.Server == "" || slices.Contains(linkConf.Servers, message.Server)) {
			log.Printf("Tg %s: For %s received %s\n", botName.Name, message.Server, message.Message)

			_, err = tgbot.SendMessage(context.TODO(), &bot.SendMessageParams{
				ChatID: message.ChatId,
				Text:   message.Message,
			})
			if err != nil {
				log.Println("error sending message to ", botName.Name, err)
			}
		}
	}
	log.Println("exit tg output", botName.Name)
}
