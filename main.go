package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// Config represents the YAML configuration file structure.
type Config struct {
	Domains          []string `yaml:"domains"`
	DaysBeforeExpire int      `yaml:"days_before_expire"`
	WebhookURL       string   `yaml:"webhook_url"`
}

func loadConfig(filename string) (Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	return config, err
}

func checkDomain(domain string, daysBeforeExpire int) (string, error) {
	conn, err := tls.Dial("tcp", domain+":443", nil)
	if err != nil {
		return "", fmt.Errorf("could not connect to %s: %v", domain, err)
	}
	defer conn.Close()

	for _, chain := range conn.ConnectionState().VerifiedChains {
		for _, cert := range chain {
			if time.Now().AddDate(0, 0, daysBeforeExpire).After(cert.NotAfter) {
				return fmt.Sprintf("%s: Certificate expires in less than %d days", domain, daysBeforeExpire), nil
			}
		}
	}

	// If we reach here, no issues detected.
	return "", nil
}

func sendAlert(webhookURL string, message string) error {
	msg := fmt.Sprintf(`{"msgtype": "text", "text": {"content": "%s"}}`, message)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer([]byte(msg)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send alert, response code: %d", resp.StatusCode)
	}

	return nil
}

func main() {
	config, err := loadConfig("config.yml")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	var issues []string
	for _, domain := range config.Domains {
		issue, err := checkDomain(domain, config.DaysBeforeExpire)
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", domain, err))
		} else if issue != "" {
			issues = append(issues, issue)
		}
	}

	if len(issues) > 0 {
		message := fmt.Sprintf("Total domains checked: %d\nDomains with issues: %d\nDetails:\n%s",
			len(config.Domains), len(issues), strings.Join(issues, "\n"))
		err := sendAlert(config.WebhookURL, message)
		if err != nil {
			fmt.Printf("Error sending alert: %v\n", err)
		}
	} else {
		fmt.Println("All domains are OK.")
	}
}
