package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Config holds the application configuration
type Config struct {
	GoDaddyAPIKey        string `json:"godaddy_api_key"`
	GoDaddyAPISecret     string `json:"godaddy_api_secret"`
	Domain               string `json:"domain"`
	RecordName           string `json:"record_name"`
	CheckIntervalMinutes int    `json:"check_interval_minutes"`
	TTL                  int    `json:"ttl"`
}

// DNSRecord represents a GoDaddy DNS A record
type DNSRecord struct {
	Data string `json:"data"`
	TTL  int    `json:"ttl"`
}

const (
	ipCheckURL     = "https://api.ipify.org?format=text"
	godaddyAPIBase = "https://api.godaddy.com"
	logFile        = "ip_log.txt"
	lastIPFile     = "last_ip.txt"
)

func main() {
	// Load configuration
	config, err := loadConfig("config.json")
	if err != nil {
		logMessage(fmt.Sprintf("Error loading config: %v", err))
		logMessage("Please create config.json based on config.json.example")
		return
	}

	logMessage("GoDaddy DNS Updater started")
	logMessage(fmt.Sprintf("Monitoring domain: %s", config.Domain))
	logMessage(fmt.Sprintf("Record name: %s", config.RecordName))
	logMessage(fmt.Sprintf("Check interval: %d minutes", config.CheckIntervalMinutes))

	// Signal channel for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Main loop
	ticker := time.NewTicker(time.Duration(config.CheckIntervalMinutes) * time.Minute)
	defer ticker.Stop()

	// Run immediately on start
	checkAndUpdate(config)

	for {
		select {
		case <-ticker.C:
			checkAndUpdate(config)
		case sig := <-sigChan:
			logMessage(fmt.Sprintf("Received signal %v, shutting down...", sig))
			return
		}
	}
}

// loadConfig loads configuration from a JSON file
func loadConfig(filename string) (*Config, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}

	// Validate required fields
	if config.GoDaddyAPIKey == "" || config.GoDaddyAPIKey == "YOUR_API_KEY_HERE" {
		return nil, fmt.Errorf("godaddy_api_key not configured")
	}
	if config.GoDaddyAPISecret == "" || config.GoDaddyAPISecret == "YOUR_API_SECRET_HERE" {
		return nil, fmt.Errorf("godaddy_api_secret not configured")
	}
	if config.Domain == "" {
		return nil, fmt.Errorf("domain not configured")
	}
	if config.RecordName == "" {
		config.RecordName = "@"
	}
	if config.CheckIntervalMinutes <= 0 {
		config.CheckIntervalMinutes = 5
	}
	if config.TTL <= 0 {
		config.TTL = 600
	}

	return &config, nil
}

// getCurrentIPFromDNS attempts to get the IP by resolving the domain's DNS
func getCurrentIPFromDNS(domain string, recordName string) (string, error) {
	// Construct the FQDN (fully qualified domain name)
	var fqdn string
	if recordName == "@" || recordName == "" {
		fqdn = domain
	} else {
		fqdn = recordName + "." + domain
	}

	// Look up the A records for the domain
	ips, err := net.LookupIP(fqdn)
	if err != nil {
		return "", err
	}

	// Return the first IPv4 address found
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			return ipv4.String(), nil
		}
	}

	return "", fmt.Errorf("no IPv4 address found for %s", fqdn)
}

// getCurrentIPFromAPI fetches the current public IP address from external API
func getCurrentIPFromAPI() (string, error) {
	resp, err := http.Get(ipCheckURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}

// getCurrentIP gets the machine's current public IP from external API,
// falling back to DNS resolution if the API is unavailable
func getCurrentIP(config *Config) (string, error) {
	ip, err := getCurrentIPFromAPI()
	if err == nil {
		logMessage(fmt.Sprintf("Public IP from API: %s", ip))
		return ip, nil
	}

	logMessage(fmt.Sprintf("API call failed: %v, falling back to DNS resolution", err))
	ip, err = getCurrentIPFromDNS(config.Domain, config.RecordName)
	if err != nil {
		return "", err
	}

	logMessage(fmt.Sprintf("IP resolved from DNS: %s", ip))
	return ip, nil
}

// getLastIP reads the last known IP from file
func getLastIP() string {
	data, err := os.ReadFile(lastIPFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// saveLastIP saves the current IP to file
func saveLastIP(ip string) error {
	return os.WriteFile(lastIPFile, []byte(ip), 0644)
}

// getGoDaddyDNS fetches the current DNS A record from GoDaddy
func getGoDaddyDNS(config *Config) (string, error) {
	// Build the API URL
	url := fmt.Sprintf("%s/v1/domains/%s/records/A/%s",
		godaddyAPIBase, config.Domain, config.RecordName)

	// Create the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Add headers
	req.Header.Set("Authorization", fmt.Sprintf("sso-key %s:%s",
		config.GoDaddyAPIKey, config.GoDaddyAPISecret))

	// Send the request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var records []DNSRecord
	err = json.Unmarshal(body, &records)
	if err != nil {
		return "", err
	}

	// Return the first record's IP (there should only be one A record with this name)
	if len(records) > 0 {
		return records[0].Data, nil
	}

	return "", fmt.Errorf("no DNS record found")
}

// updateGoDaddyDNS updates the DNS A record via GoDaddy API
func updateGoDaddyDNS(config *Config, ip string) error {
	// Prepare the DNS record
	records := []DNSRecord{
		{
			Data: ip,
			TTL:  config.TTL,
		},
	}

	jsonData, err := json.Marshal(records)
	if err != nil {
		return err
	}

	// Build the API URL
	url := fmt.Sprintf("%s/v1/domains/%s/records/A/%s",
		godaddyAPIBase, config.Domain, config.RecordName)

	// Create the request
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("sso-key %s:%s",
		config.GoDaddyAPIKey, config.GoDaddyAPISecret))

	// Send the request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// checkAndUpdate checks the current IP and updates DNS if changed
func checkAndUpdate(config *Config) {
	// Get current public IP (tries DNS first, then API fallback)
	currentIP, err := getCurrentIP(config)
	if err != nil {
		logMessage(fmt.Sprintf("Error getting current IP: %v", err))
		return
	}

	logMessage(fmt.Sprintf("Current IP: %s", currentIP))

	// Get current DNS record from GoDaddy
	dnsIP, err := getGoDaddyDNS(config)
	if err != nil {
		logMessage(fmt.Sprintf("Error fetching GoDaddy DNS record: %v", err))
		// Fallback to local file if API call fails
		dnsIP = getLastIP()
		logMessage(fmt.Sprintf("Using cached IP from file: %s", dnsIP))
	} else {
		logMessage(fmt.Sprintf("GoDaddy DNS IP: %s", dnsIP))
	}

	// Check if IP has changed
	if currentIP == dnsIP {
		logMessage("IP matches GoDaddy DNS, no update needed")
		// Update local cache even if no DNS update needed
		saveLastIP(currentIP)
		return
	}

	logMessage(fmt.Sprintf("IP changed from %s to %s", dnsIP, currentIP))

	// Update GoDaddy DNS
	err = updateGoDaddyDNS(config, currentIP)
	if err != nil {
		logMessage(fmt.Sprintf("Error updating GoDaddy DNS: %v", err))
		return
	}

	logMessage("Successfully updated GoDaddy DNS")

	// Save the new IP
	err = saveLastIP(currentIP)
	if err != nil {
		logMessage(fmt.Sprintf("Error saving last IP: %v", err))
	}
}

// logMessage logs a message with timestamp to both console and file
func logMessage(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[%s] %s\n", timestamp, message)

	// Print to console
	fmt.Print(logLine)

	// Append to log file
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(logLine)
}
