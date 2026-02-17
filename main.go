package main

import (
	"context"
	"log"
	"os"
	"slices"
	"sync"

	"github.com/go-co-op/gocron/v2"
	"github.com/serzheka/serverrestart/config"
	pkgDb "github.com/serzheka/serverrestart/db"
	"github.com/serzheka/serverrestart/input"
	"github.com/serzheka/serverrestart/output"
	"github.com/serzheka/serverrestart/util"
	"github.com/xlab/closer"
	"go.yaml.in/yaml/v4"
)

var (
	inputFunctions = map[string]func(context.Context, *config.LinkMethods, chan<- util.InOutMessage){
		"example": input.Example,
		"tg":      input.Tg,
	}
	outputFunctions = map[string]func(*config.LinkMethods, <-chan util.InOutMessage){
		"console": output.Console,
		"example": output.Example,
		"tg":      output.Tg,
		"teams":   output.Teams,
	}
)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	log.Println("Starting components deployment")

	db, err := pkgDb.NewDB(config.ConfigPath + "/serverrestart.db")
	if err != nil {
		log.Panicln(err)
	}

	os.MkdirAll(config.ConfigPath+"/logs", os.ModePerm)

	inputLinks, outputLinks := loadProjectLinks()
	var wg, wgin, wgout sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Panicln("Cannot start scheduler: " + err.Error())
	}
	inputChain := make(chan util.InOutMessage)
	outputChans := make([]chan util.InOutMessage, len(outputLinks))
	for i := range outputChans {
		outputChans[i] = make(chan util.InOutMessage)
	}

	closer.Bind(func() {
		log.Println("Received programm close")
		cancel()
		err = scheduler.Shutdown()
		if err != nil {
			log.Println("error shutdown scheduler", err)
		}
		wgin.Wait()

		close(inputChain)
		wg.Wait()

		for _, c := range outputChans {
			close(c)
		}
		wgout.Wait()

		if db != nil {
			err = db.Close()
			if err != nil {
				log.Println("error closing db", err)
			}
		}
		log.Println("Programm closed successfully")
	})
	log.Println("Variables init completed")

	noinput := true
	for _, inputLink := range inputLinks {
		if inputFunction, ok := inputFunctions[inputLink.Name]; ok {
			wgin.Go(func() {
				inputFunction(ctx, inputLink, inputChain)
			})
			noinput = false
		} else {
			log.Println("Cannot find input routine " + inputLink.Name)
		}
	}
	if noinput {
		log.Panicln("No input function started, exitting")
	}
	log.Println("Input functions started")

	sendOutputChans := make([]chan<- util.InOutMessage, len(outputChans))
	for i, ch := range outputChans {
		sendOutputChans[i] = ch
	}
	wg.Go(func() {
		processInput(inputChain, db, sendOutputChans)
	})
	log.Println("Process input started")

	scheduler.NewJob(
		gocron.CronJob(`* * * * *`, false),
		gocron.NewTask(func() {
			processRestart(db, sendOutputChans)
		}),
	)
	log.Println("Scheduler job created")

	for i, outputLink := range outputLinks {
		if outputFunction, ok := outputFunctions[outputLink.Name]; ok {
			wgout.Go(func() {
				outputFunction(outputLink, outputChans[i])
			})
		} else {
			log.Println("Cannot find output routine " + outputLink.Name)
		}
	}
	log.Println("Output functions started")

	scheduler.Start()
	log.Println("All components started")

	closer.Hold()
}

func loadProjectLinks() ([]*config.LinkMethods, []*config.LinkMethods) {
	projectConfigBytes, err := os.ReadFile(config.ConfigPath + "/projects.yaml")
	if err != nil {
		log.Panicln(err)
	}

	var projectConfigs map[string]config.ProjectConfig
	err = yaml.Unmarshal(projectConfigBytes, &projectConfigs)
	if err != nil {
		log.Panicln(err)
	}

	projectNames := make([]string, 0, len(projectConfigs))
	var (
		inputLinks []*config.LinkMethods
		outLinks   []*config.LinkMethods
	)
	for name, projectConfig := range projectConfigs {
		if slices.Contains(projectNames, name) {
			log.Panicln("Duplicate project " + name)
		}

		projectNames = append(projectNames, name)
		var serverCommands []config.ServerCommand
		for _, server := range projectConfig.Servers {
			serverFile := server + ".yaml"
			bytes, err := os.ReadFile(config.ConfigPath + "/" + serverFile)
			if err != nil {
				log.Panicf("error reading %s (project %s): %v", serverFile, name, err)
			}
			var serverConf config.ServerConfig
			err = yaml.Unmarshal(bytes, &serverConf)
			if err != nil {
				log.Panicf("error unmarshalling %s (project %s): %v", serverFile, name, err)
			}
			commands := make([]string, 0, len(serverConf.Commands))
			for _, command := range serverConf.Commands {
				commands = append(commands, command.Name)
			}
			serverCommands = append(serverCommands, config.ServerCommand{
				Server:   server,
				Commands: commands,
			})
		}
		for _, link := range projectConfig.InOutLinks {
			link.ServerCommands = serverCommands
			inputLinks = append(inputLinks, &link)
			outLinks = append(outLinks, &link)
		}
		for _, link := range projectConfig.OutLinks {
			link.ServerCommands = serverCommands
			outLinks = append(outLinks, &link)
		}
	}

	log.Println("projects.yaml successfully revised")
	return inputLinks, outLinks
}
