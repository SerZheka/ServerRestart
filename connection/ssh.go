package connection

import (
	"errors"
	"log"

	"github.com/serzheka/serverrestart/config"
	"golang.org/x/crypto/ssh"
)

func Ssh(address string, secret *config.Secret, command string, logger *log.Logger) error {
	logger.Println("start processing ssh command", command)
	clientConf := &ssh.ClientConfig{
		User:            secret.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.Password(secret.Password),
		},
	}
	client, err := ssh.Dial("tcp", address, clientConf)
	if err != nil {
		return errors.New("cannot connect to server: " + err.Error())
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return errors.New("cannot open new session for server: " + err.Error())
	}

	session.Stdout = logger.Writer()
	err = session.Run(command)
	session.Close()

	if err != nil {
		return errors.New("script " + command + "got error: " + err.Error())
	}

	logger.Println("finished processing ssh command")
	return nil
}
