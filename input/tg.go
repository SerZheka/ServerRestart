package input

import (
	"context"
	"log"
	"regexp"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/serzheka/serverrestart/config"
	"github.com/serzheka/serverrestart/util"
)

func Tg(ctx context.Context, linkConf *config.LinkMethods, output chan<- util.InOutMessage) {
	logger := log.New(log.Writer(), "TG INPUT:", log.Flags()|log.Lmsgprefix)
	logger.Println("starting tg bot for servers", linkConf.Servers)

	serverReg := regexp.MustCompile(`[ ]\d{1,3}[ .]\d{1,3}(?: |$)`)
	opts := []bot.Option{
		bot.WithMessageTextHandler("/help", bot.MatchTypePrefix, helpHandler),
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if update.Message != nil && update.Message.Text != "" {
				logger.Println("Received:", update.Message.Text)

				text := strings.ReplaceAll(update.Message.Text, "_", " ")[1:]
				if atIndex := strings.Index(text, "@"); atIndex != -1 {
					text = text[:atIndex]
				}
				serverIndex := serverReg.FindStringIndex(text)
				if serverIndex != nil {
					textToSend := text[:serverIndex[0]] + ";" + strings.Replace(text[serverIndex[0]+1:serverIndex[1]], " ", ".", 1)
					if serverIndex[1] != len(text) {
						textToSend = textToSend[:len(textToSend)-1] + ";" + text[serverIndex[1]:]
					} else {
						textToSend += ";+10/10"
					}
					logger.Println("Sending to output", textToSend)
					output <- util.InOutMessage{Message: textToSend, LinkMethod: linkConf, ChatId: update.Message.Chat.ID}
				}
			}
		}),
	}

	tgbot, err := bot.New(linkConf.Key, opts...)
	if err != nil {
		logger.Println("error creating tg bot", err)
		return
	}
	botName, err := tgbot.GetMyName(ctx, nil)
	if err != nil {
		logger.Println("error getting bot name", err)
		return
	}
	logger.Println("authorized on account", botName.Name)

	tgbot.RegisterHandlerRegexp(bot.HandlerTypeMessageText, regexp.MustCompile(`^\/start(?:$|@)`), helpHandler)
	botCommands := []models.BotCommand{
		{Command: "start", Description: "start bot"},
		{Command: "help", Description: "help"},
	}
	for _, serverId := range linkConf.Servers {
		tgServerId := strings.Replace(serverId, ".", "_", 1)
		botCommands = append(botCommands,
			models.BotCommand{Command: "restart_" + tgServerId, Description: "restart server in 10 min"},
			models.BotCommand{Command: "restart_jboss_" + tgServerId, Description: "restart jboss in 10 min"},
			models.BotCommand{Command: "start_tsm_" + tgServerId + "_now", Description: "start tsm now"},
			models.BotCommand{Command: "cancel_" + tgServerId, Description: "cancel job"},
		)
	}
	tgbot.SetMyCommands(ctx, &bot.SetMyCommandsParams{Commands: botCommands})

	logger.Println("start tg bot", botName.Name)
	tgbot.Start(ctx)

	logger.Println("exit tg bot", botName.Name)
}

func helpHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: "This bot is used to control servers\n" +
			"You can use the predefiend commands or send a message in following format:\n" +
			"/<command> <server_name> <time>",
	})
}
