package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/serzheka/serverrestart/config"
	"github.com/serzheka/serverrestart/connection"
	packdb "github.com/serzheka/serverrestart/db"
	"github.com/serzheka/serverrestart/util"
	"go.yaml.in/yaml/v4"
)

var connTypes = map[string]func(string, *config.Secret, string, *log.Logger) error{
	"ssh": connection.Ssh,
}

func runScript(restart *packdb.Restart, output []chan<- util.InOutMessage) {
	startMessage := fmt.Sprintf("Start processing %s for %s", restart.Command, restart.Server)
	log.Println(startMessage)
	for _, outchan := range output {
		outchan <- util.InOutMessage{
			Message: startMessage,
			Server:  restart.Server,
			ChatId:  restart.ChatId,
		}
	}

	logFile, err := os.OpenFile(config.ConfigPath+"/logs/"+time.Now().Format("06-01-02_15-04")+"_"+restart.Command+"_"+restart.Server+".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Println("error open log file", err)
		util.SendErrMessages(output, restart)
		return
	}
	defer logFile.Close()

	logger := log.New(io.MultiWriter(os.Stdout, logFile), "", log.LstdFlags)
	logger.Println("start init")

	filename := restart.Server + ".yaml"
	bytes, err := os.ReadFile(config.ConfigPath + "/" + filename)
	if err != nil {
		logger.Println("error reading", filename, err)
		util.SendErrMessages(output, restart)
		return
	}

	var serverConf config.ServerConfig
	err = yaml.Unmarshal(bytes, &serverConf)
	if err != nil {
		logger.Println("error unmarshalling yaml", err)
		util.SendErrMessages(output, restart)
		return
	}

	commandIndex := -1
	for i, command := range serverConf.Commands {
		if command.Name == restart.Command {
			commandIndex = i
			break
		}
	}
	if commandIndex == -1 {
		logger.Println("cannot find command", restart.Command)
		util.SendErrMessages(output, restart)
		return
	}

	log.Println("started processing scripts")
	for _, scriptName := range serverConf.Commands[commandIndex].Scripts {
		scriptIndex := -1
		for i, script := range serverConf.Scripts {
			if script.Name == scriptName {
				scriptIndex = i
				break
			}
		}
		if scriptIndex == -1 {
			logger.Println("cannot find script", scriptName)
			util.SendErrMessages(output, restart)
			return
		}
		script := serverConf.Scripts[scriptIndex]

		if script.Type == "tsm" {
			err = connection.ProcessTsm(script.Script, serverConf.Ip+":"+serverConf.JbossPort, serverConf.Timeout, &serverConf.OfsSecret, &serverConf.TafjSecret, logger)
		} else {
			if conntype, ok := connTypes[script.Type]; ok {
				err = conntype(serverConf.Ip+":"+serverConf.Port, &serverConf.ServerSecret, script.Script, logger)
			} else {
				err = errors.New("cannot find connection type " + script.Type)
			}
		}

		if err != nil {
			logger.Println("got error in command", restart.Command, err)
			util.SendErrMessages(output, restart)
			return
		}

		if script.Message != "" {
			for _, outchan := range output {
				outchan <- util.InOutMessage{
					Message: script.Message,
					Server:  restart.Server,
					ChatId:  restart.ChatId,
				}
			}
		}
	}
}
