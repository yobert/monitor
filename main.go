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
	Timeout   int    `json:"timeout"`  // timeout in seconds
	Interval  int    `json:"interval"` // monitor interval in seconds

	logged, status, pdstatus, ntfyshstatus Status

	pdincident string
	ntfysh     bool
}

type Status struct {
	Message string
	Bad     bool
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

	timeout := service.Timeout
	if timeout == 0 {
		timeout = 10
	}

	interval := service.Interval
	if interval == 0 {
		interval = 10
	}

	client := http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	for {
		newstatus := Status{}

		resp, err := client.Get(service.URL)
		if err != nil {
			newstatus.Message = err.Error()
			if strings.Contains(newstatus.Message, "Timeout") {
				newstatus.Message = fmt.Sprintf("Timed out after %v", client.Timeout)
			}
			newstatus.Bad = true
		} else {
			txt := ""
			body, err := ioutil.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				txt = err.Error()
				newstatus.Bad = true
			} else {
				ct := strings.ToLower(resp.Header.Get("Content-Type"))
				if strings.HasPrefix(ct, "text/plain") && (!strings.Contains(ct, "encoding") || strings.Contains(ct, "utf8") || strings.Contains(ct, "utf-8")) {
					txt = string(body)
				} else {
					txt = ct
				}

				if resp.TLS != nil {
					for _, cert := range resp.TLS.PeerCertificates {
						expires := cert.NotAfter.Sub(time.Now())
						if expires < time.Hour*24*7 {
							newstatus.Bad = true
							expires = expires.Truncate(time.Hour)
							txt = fmt.Sprintf("Certificate %#v expires in ", cert.Subject.String())
							if expires > time.Hour*24 {
								txt += fmt.Sprintf("%.0f days", expires.Hours()/24)
							} else {
								txt += expires.Truncate(time.Hour * 24).String()
							}
						}
					}
				}
			}
			if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
				newstatus.Bad = true
			}

			if txt == "" {
				txt = "(no response body)"
			}
			if len(txt) > 1024 {
				txt = fmt.Sprintf("%s ... (trimmed %d bytes)", txt[:1024], len(txt)-1024)
			}
			newstatus.Message = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, txt)
		}

		if newstatus != service.logged {
			symbol := "✓"
			if newstatus.Bad {
				symbol = "✗"
			}
			log.Println(symbol, service.URL, newstatus.Message)
			service.logged = newstatus
		}

		// Single debounce.
		if newstatus != service.status {
			service.status = newstatus
		} else {
			alertpagerduty(&service)
			alertntfysh(&service)
		}

		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func alertntfysh(service *Service) {
	if service.Ntfy == "" {
		return
	}

	status := service.status
	old := service.ntfyshstatus

	if status == old {
		return
	}

	if status.Bad == false && old.Bad == false {
		// Ignore changes in only success status text
		return
	}

	req, err := http.NewRequest("POST", "https://ntfy.sh/"+service.Ntfy, bytes.NewBufferString(status.Message))
	if err != nil {
		log.Println(err)
		time.Sleep(time.Minute * 5)
		return
	}

	req.Header.Set("Title", service.URL)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		time.Sleep(time.Minute * 5)
		return
	}
	resp.Body.Close()

	service.ntfyshstatus = status
}

func alertpagerduty(service *Service) {
	if service.PagerDuty == "" {
		return
	}

	status := service.status
	old := service.pdstatus
	incident := service.pdincident

	if status == old {
		return
	}

	if status.Bad == false && old.Bad == false && incident == "" {
		// Ignore changes in only success status text
		return
	}

	event := pagerduty.Event{
		Type:        "trigger",
		ServiceKey:  service.PagerDuty,
		Description: status.Message,
		IncidentKey: incident,
	}

	if !status.Bad {
		event.Type = "resolve"
	}

	resp, err := pagerduty.CreateEvent(event)
	if err != nil {
		log.Println(err)
		time.Sleep(time.Minute * 5)
		return
	}

	if status.Bad && incident == "" {
		service.pdincident = resp.IncidentKey
	}
	if !status.Bad && incident != "" {
		service.pdincident = ""
	}
	service.pdstatus = status
	return
}
