package main

import (
	"context"
	"log"
	"os"
	"slices"
	"sync"
	"time"

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
	}
	outputFunctions = map[string]func(*config.LinkMethods, <-chan util.InOutMessage){
		"console": output.Console,
		"example": output.Example,
	}
)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	log.Println("Starting components deployment")

	db, err := pkgDb.NewDB("serverrestart.db")
	if err != nil {
		log.Panicln(err)
	}

	entries, err := os.ReadDir(config.ConfigPath)
	if err != nil {
		log.Panicln(err)
	}
	if slices.ContainsFunc(entries, func(entry os.DirEntry) bool { return entry.Name() == "clearRestarts" }) {
		db.Clear()
		os.Remove(config.ConfigPath + "/clearRestarts")
	}
	os.MkdirAll(config.ConfigPath+"/logs", os.ModePerm)

	inputLinks, outputLinks := loadProjectLinks(&entries)
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
		gocron.DurationJob(time.Minute),
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

func loadProjectLinks(entries *[]os.DirEntry) ([]*config.LinkMethods, []*config.LinkMethods) {
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
		for _, server := range projectConfig.Servers {
			if serverFile := server + ".yaml"; !slices.ContainsFunc(*entries, func(entry os.DirEntry) bool { return entry.Name() == serverFile }) {
				log.Panicln("Config directory does not contain config for " + server + " which is needed for project " + name)
			}
		}
		for _, link := range projectConfig.InOutLinks {
			newconf := config.LinkMethods{
				Name:    link.Name,
				Key:     link.Key,
				Servers: projectConfig.Servers,
			}
			inputLinks = append(inputLinks, &newconf)
			outLinks = append(outLinks, &newconf)
		}
		for _, link := range projectConfig.OutLinks {
			outLinks = append(outLinks, &config.LinkMethods{
				Name:    link.Name,
				Key:     link.Key,
				Servers: projectConfig.Servers,
			})
		}
	}

	log.Println("projects.yaml successfully revised")
	return inputLinks, outLinks
}
