package connection

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/serzheka/serverrestart/config"
)

type ofsResponse struct {
	OfsRequest  string `json:"ofsRequest"`
	OfsResponse string `json:"ofsResponse"`
}

type SessionResponse struct {
	ServerID           string `json:"serverId"`
	ObjectName         string `json:"objectName"`
	SessionID          int64  `json:"sessionId"`
	ProcessID          int    `json:"processId"`
	ThreadID           int64  `json:"threadId"`
	PortNum            int    `json:"portNum"`
	Background         bool   `json:"background"`
	OfsSource          string `json:"ofsSource"`
	NbRequests         int    `json:"nbRequests"`
	CreationDate       string `json:"creationDate"`
	ExecutionTime      int    `json:"executionTime"`
	IdleDetection      bool   `json:"idleDetection"`
	IdleTime           int    `json:"idleTime"`
	Expired            bool   `json:"expired"`
	ShutdownInProgress bool   `json:"shutdownInProgress"`
	Shutdown           bool   `json:"shutdown"`
}

type sessionsResponse struct {
	Sessions []SessionResponse `json:"sessions"`
}

func ProcessTsm(command, address string, timeout int, ofsSecret, tafjSecret *config.Secret, logger *log.Logger) error {
	logger.Println("start processing tsm command", command)
	ofsCommand := "{\"ofsRequest\": \"TSA.SERVICE,/I//0/0," + ofsSecret.User + "/" + ofsSecret.Password + ",TSM,SERVICE.CONTROL="
	client := http.Client{
		Timeout: 30 * time.Second,
	}
	ofsReq, _ := http.NewRequest("POST", "http://"+address+"/TAFJRestServices/resources/ofs", nil)
	ofsReq.Header.Set("Content-Type", "application/json")
	ofsReq.SetBasicAuth(tafjSecret.User, tafjSecret.Password)
	sessionsReq, _ := http.NewRequest("GET", "http://"+address+"/TAFJRestServices/resources/management/session", nil)
	sessionsReq.SetBasicAuth(tafjSecret.User, tafjSecret.Password)

	logger.Println("finished preparation for tsm command")

	if command == "stop" {
		ofsBody := ofsCommand + "STOP\"}"
		ofsReq.Body = io.NopCloser(bytes.NewBufferString(ofsBody))
		ofsReq.ContentLength = int64(len(ofsBody))
		res, err := client.Do(ofsReq)
		if err != nil {
			return err
		}
		var ofsResp ofsResponse
		json.NewDecoder(res.Body).Decode(&ofsResp)
		res.Body.Close()
		if !(strings.HasPrefix(ofsResp.OfsResponse, "TSM//1") || strings.HasSuffix(ofsResp.OfsResponse, "LIVE RECORD NOT CHANGED")) {
			return errors.New("error processing ofs: " + ofsResp.OfsResponse)
		}
		logger.Println("ofs request for TSM stop successfully sent")

		for range timeout {
			res, err := client.Do(sessionsReq)
			if err != nil {
				return err
			}
			var sessionsResp sessionsResponse
			err = json.NewDecoder(res.Body).Decode(&sessionsResp)
			res.Body.Close()
			if err != nil {
				return errors.New("error decoding sessions response" + err.Error())
			}

			if slices.IndexFunc(sessionsResp.Sessions, func(sessn SessionResponse) bool { return sessn.Background }) == -1 {
				logger.Println("all background sessions are stopped")
				break
			}

			time.Sleep(10 * time.Second)
		}

		logger.Println("exit stopping tsm")
	} else {
		ofsBody := ofsCommand + "START\"}"
		ofsReq.Body = io.NopCloser(bytes.NewBufferString(ofsBody))
		ofsReq.ContentLength = int64(len(ofsBody))
		res, err := client.Do(ofsReq)
		if err != nil {
			return err
		}
		var ofsResp ofsResponse
		json.NewDecoder(res.Body).Decode(&ofsResp)
		res.Body.Close()
		if !(strings.HasPrefix(ofsResp.OfsResponse, "TSM//1") || strings.HasSuffix(ofsResp.OfsResponse, "LIVE RECORD NOT CHANGED")) {
			return errors.New("error processing ofs: " + ofsResp.OfsResponse)
		}
		logger.Println("ofs request for TSM start successfully sent")

		data := url.Values{
			"message": {"START.TSM"},
		}
		startTsmReg, _ := http.NewRequest("POST", "http://"+address+"/TAFJRestServices/resources/management/topic", bytes.NewBufferString(data.Encode()))
		startTsmReg.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		startTsmReg.SetBasicAuth(tafjSecret.User, tafjSecret.Password)
		tsmRes, err := client.Do(startTsmReg)
		if err != nil {
			return err
		}
		defer tsmRes.Body.Close()
		if tsmRes.StatusCode != 200 {
			bodyBytes, err := io.ReadAll(tsmRes.Body)
			if err != nil {
				return errors.New("error reading response body for tsm start: " + err.Error())
			}
			return fmt.Errorf("request for tsm start returned status code %v with body %s", res.StatusCode, string(bodyBytes))
		}
		logger.Println("START.TSM successfully sent")

		var isStarted bool
		for range timeout {
			res, err := client.Do(sessionsReq)
			if err != nil {
				return err
			}
			var sessionsResp sessionsResponse
			err = json.NewDecoder(res.Body).Decode(&sessionsResp)
			res.Body.Close()
			if err != nil {
				return errors.New("error decoding sessions response" + err.Error())
			}

			if slices.IndexFunc(sessionsResp.Sessions, func(sessn SessionResponse) bool { return sessn.Background }) != -1 {
				logger.Println("background sessions are started")
				isStarted = true
				break
			}

			time.Sleep(10 * time.Second)
		}

		if isStarted {
			logger.Println("exit starting tsm")
		} else {
			return fmt.Errorf("tsm not started after %d", timeout)
		}
	}

	return nil
}
