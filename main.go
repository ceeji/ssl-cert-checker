package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// Config represents the YAML configuration file structure.
type Config struct {
	Domains          []DomainInfo `yaml:"domains"`
	DaysBeforeExpire int          `yaml:"days_before_expire"`
	WebhookURL       string       `yaml:"webhook_url"`
}

type DomainInfo struct {
	Name             string `yaml:"name"`
	Domain           string `yaml:"domain"`
	IgnoreServerName bool   `yaml:"ignore_server_name"`
}

func loadConfig(filename string) (Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	return config, err
}

func checkDomain(domain DomainInfo, daysBeforeExpire int) (string, error) {
	serverName := domain.Domain
	if domain.IgnoreServerName {
		serverName = ""
	}
	config := &tls.Config{
		InsecureSkipVerify: false,      // Ensure certificate verification is enabled
		ServerName:         serverName, // Ensure the domain matches the certificate
	}

	fullDomain := domain.Domain
	if !strings.Contains(fullDomain, ":") {
		fullDomain += ":443"
	}

	conn, err := tls.Dial("tcp", fullDomain, config)
	if err != nil {
		return fmt.Sprintf("无法连接到服务器 %s: %v", domain.Domain, err), nil
	}
	defer conn.Close()

	for _, chain := range conn.ConnectionState().VerifiedChains {
		for _, cert := range chain {
			if time.Now().AddDate(0, 0, daysBeforeExpire).After(cert.NotAfter) {
				days := math.Round(cert.NotAfter.Sub(time.Now()).Hours() / float64(24))
				return fmt.Sprintf("证书将在 %.0f 日内过期", days), nil
			}
		}
	}

	// If we reach here, no issues detected.
	return "", nil
}

func sendAlert(webhookURL string, message string) error {
	msg := fmt.Sprintf(`{"msgtype": "markdown", "markdown": {"content": "%s"}}`, message)
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
		log.Printf("checking domain %s (%s)", domain.Name, domain.Domain)
		issue, err := checkDomain(domain, config.DaysBeforeExpire)
		if err != nil {
			issues = append(issues, fmt.Sprintf("- **%s**(%s): %v", domain.Name, domain.Domain, err))
		} else if issue != "" {
			issues = append(issues, fmt.Sprintf("- **%s**(%s): %v", domain.Name, domain.Domain, issue))
		}
	}

	if len(issues) > 0 {
		message := fmt.Sprintf("# 域名证书检查报告\n\n- 检查域名: **%d**\n- 问题域名: **%d**\n## 错误详情\n\n%s",
			len(config.Domains), len(issues), strings.Join(issues, "\n"))
		log.Println(message)
		err := sendAlert(config.WebhookURL, message)
		if err != nil {
			fmt.Printf("Error sending alert: %v\n", err)
		}
	} else {
		fmt.Println("All domains are OK.")
	}
}
