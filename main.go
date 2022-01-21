package main

import (
	"bytes"
	"fmt"
	"github.com/PagerDuty/go-pagerduty"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type Service struct {
	URL       string `json:"url"`
	PagerDuty string `json:"pagerduty"`
	Ntfy      string `json:"ntfy"`
}

func main() {
	services, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	for _, service := range services {
		go watch(service)
	}

	select {}
}

func watch(service Service) {
	status := ""
	pagerdutyincident := ""
	ntfysh := false

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

		if service.Ntfy != "" {
			if bad != ntfysh || (bad && status != newstatus) {
				req, err := http.NewRequest("POST", "https://ntfy.sh/"+service.Ntfy, bytes.NewBufferString(newstatus))
				if err != nil {
					log.Println(err)
					time.Sleep(time.Minute * 5)
					continue
				}

				req.Header.Set("Title", service.URL)

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					log.Println(err)
					time.Sleep(time.Minute * 5)
					continue
				}
				resp.Body.Close()

				ntfysh = bad
			}
		}

		if service.PagerDuty != "" {
			if bad && pagerdutyincident == "" {
				event := pagerduty.Event{
					Type:        "trigger",
					ServiceKey:  service.PagerDuty,
					Description: newstatus,
				}

				resp, err := pagerduty.CreateEvent(event)
				if err != nil {
					log.Println(err)
					time.Sleep(time.Minute * 5) // If we can't log to pagerduty, just sleep for a few minutes and we'll try again.
					continue
				}

				pagerdutyincident = resp.IncidentKey
			} else if !bad && pagerdutyincident != "" {
				event := pagerduty.Event{
					Type:        "resolve",
					ServiceKey:  service.PagerDuty,
					Description: newstatus,
					IncidentKey: pagerdutyincident,
				}

				if _, err := pagerduty.CreateEvent(event); err != nil {
					log.Println(err)
					time.Sleep(time.Minute * 5) // If we can't log to pagerduty, just sleep for a few minutes and we'll try again.
					continue
				}

				pagerdutyincident = ""
			} else if bad && status != newstatus {
				// Update existing incident with new status (maybe error changed?)
				event := pagerduty.Event{
					Type:        "trigger",
					ServiceKey:  service.PagerDuty,
					Description: newstatus,
					IncidentKey: pagerdutyincident,
				}

				if _, err := pagerduty.CreateEvent(event); err != nil {
					log.Println(err)
					time.Sleep(time.Minute * 5) // If we can't log to pagerduty, just sleep for a few minutes and we'll try again.
					continue
				}
			}
		}

		status = newstatus
		time.Sleep(time.Second * 10)
	}
}
