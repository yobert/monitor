package main

import (
	"fmt"
	"github.com/PagerDuty/go-pagerduty"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type Service struct {
	URL string `json:"url"`
	Key string `json:"key"`
}

func main() {
	services, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	for _, service := range services {
		if service.Key == "" || service.URL == "" {
			log.Fatal("Service is missing key or url parameter")
		}
	}

	for _, service := range services {
		go watch(service)
	}

	select {}
}

func watch(service Service) {
	status := ""
	incident := ""

	for {
		newstatus := ""
		bad := false

		resp, err := http.Get(service.URL)
		if err != nil {
			newstatus = err.Error()
			bad = true
		} else {
			txt := ""
			body, err := ioutil.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				txt = err.Error()
				bad = true
			} else {
				ct := strings.ToLower(resp.Header.Get("Content-Type"))
				if strings.HasPrefix(ct, "text/plain") && (!strings.Contains(ct, "encoding") || strings.Contains(ct, "utf8") || strings.Contains(ct, "utf-8")) {
					txt = string(body)
				} else {
					txt = ct
				}
			}

			if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
				bad = true
			}

			if txt == "" {
				txt = "(no response body)"
			}
			if len(txt) > 1024 {
				txt = fmt.Sprintf("%s ... (trimmed %d bytes)", txt[:1024], len(txt)-1024)
			}
			newstatus = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, txt)
		}

		if newstatus != status {
			symbol := "✓"
			if bad {
				symbol = "✗"
			}
			log.Println(service.URL, symbol, newstatus)
		}

		if bad && incident == "" {
			event := pagerduty.Event{
				Type:        "trigger",
				ServiceKey:  service.Key,
				Description: newstatus,
			}

			resp, err := pagerduty.CreateEvent(event)
			if err != nil {
				log.Println(err)
				time.Sleep(time.Minute * 5) // If we can't log to pagerduty, just sleep for a few minutes and we'll try again.
				continue
			}

			incident = resp.IncidentKey
		} else if !bad && incident != "" {
			event := pagerduty.Event{
				Type:        "resolve",
				ServiceKey:  service.Key,
				Description: newstatus,
				IncidentKey: incident,
			}

			if _, err := pagerduty.CreateEvent(event); err != nil {
				log.Println(err)
				time.Sleep(time.Minute * 5) // If we can't log to pagerduty, just sleep for a few minutes and we'll try again.
				continue
			}

			incident = ""
		} else if bad {
			// Update existing incident with new status (maybe error changed?)
			event := pagerduty.Event{
				Type:        "trigger",
				ServiceKey:  service.Key,
				Description: newstatus,
				IncidentKey: incident,
			}

			if _, err := pagerduty.CreateEvent(event); err != nil {
				log.Println(err)
				time.Sleep(time.Minute * 5) // If we can't log to pagerduty, just sleep for a few minutes and we'll try again.
				continue
			}
		}

		status = newstatus
		time.Sleep(time.Second * 10)
	}
}
