package device

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"strings"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	log "github.com/sirupsen/logrus"
)

// Device represents device information
type Device struct {
	UUID            string `json:"uuid"`
	OS              string `json:"os"`
	Arch            string `json:"arch"`
	LAN             string `json:"lan"`
	WAN             string `json:"wan"`
	MAC             string `json:"mac"`
	CPU             string `json:"cpu"`
	Cores           int    `json:"cores"`
	RAM             uint64 `json:"ram"`
	Uptime          uint64 `json:"uptime"`
	Hostname        string `json:"hostname"`
	Username        string `json:"username"`
	Version         string `json:"version"`
	WatchdogVersion string `json:"watchdog_version"`
	HasProxy        bool   `json:"has_proxy"`
}

// GetDeviceInfo collects comprehensive device information
func GetDeviceInfo() (*Device, error) {
	// Generate or get machine ID
	id, err := machineid.ID()
	if err != nil {
		log.WithError(err).Warn("Failed to get machine ID, generating random ID")
		secBuffer := make([]byte, 16)
		if _, err := rand.Read(secBuffer); err != nil {
			return nil, err
		}
		id = hex.EncodeToString(secBuffer)
	}

	// Get network information
	localIP, err := GetLocalIP()
	if err != nil {
		log.WithError(err).Warn("Failed to get local IP")
		localIP = "unknown"
	}

	macAddr, err := GetMacAddress()
	if err != nil {
		log.WithError(err).Warn("Failed to get MAC address")
		macAddr = "unknown"
	}

	wanIP, err := GetWanIP()
	if err != nil {
		log.WithError(err).Warn("Failed to get WAN IP")
		wanIP = "unknown"
	}

	// Get CPU information
	cpuInfo, cores, err := GetCPUInfo()
	if err != nil {
		log.WithError(err).Warn("Failed to get CPU info")
		cpuInfo = "unknown"
		cores = 0
	}

	// Get RAM information
	ramInfo, err := GetRAMInfo()
	if err != nil {
		log.WithError(err).Warn("Failed to get RAM info")
		ramInfo = 0
	}

	// Get system uptime
	uptime, err := host.Uptime()
	if err != nil {
		log.WithError(err).Warn("Failed to get uptime")
		uptime = 0
	}

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		log.WithError(err).Warn("Failed to get hostname")
		hostname = "unknown"
	}

	// Get username
	currentUser, err := user.Current()
	username := "unknown"
	if err != nil {
		log.WithError(err).Warn("Failed to get current user")
	} else {
		username = currentUser.Username
		// Remove domain prefix if present (Windows)
		if slashIndex := strings.Index(username, `\`); slashIndex > -1 && slashIndex+1 < len(username) {
			username = username[slashIndex+1:]
		}
	}

	// Check for proxy
	hasProxy := HasProxy()

	return &Device{
		UUID:     id,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		LAN:      localIP,
		WAN:      wanIP,
		MAC:      macAddr,
		CPU:      cpuInfo,
		RAM:      ramInfo,
		Cores:    cores,
		Uptime:   uptime,
		Hostname: hostname,
		Username: username,
		HasProxy: hasProxy,
	}, nil
}

// GetLocalIP returns the local IP address
func GetLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// GetWanIP returns the external IP address
func GetWanIP() (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get("https://www.icanhazip.com")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(ip)), nil
}

// GetMacAddress returns the MAC address of the first network interface
func GetMacAddress() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		if iface.HardwareAddr != nil && len(iface.HardwareAddr) > 0 {
			// Skip loopback and virtual interfaces
			if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
				return strings.ToUpper(iface.HardwareAddr.String()), nil
			}
		}
	}

	return "", errors.New("no valid MAC address found")
}

// GetCPUInfo returns CPU model name and core count
func GetCPUInfo() (string, int, error) {
	info, err := cpu.Info()
	if err != nil {
		return "", 0, err
	}
	if len(info) == 0 {
		return "", 0, errors.New("no CPU info available")
	}

	count, err := cpu.Counts(true)
	if err != nil {
		return info[0].ModelName, 0, err
	}

	return info[0].ModelName, count, nil
}

// GetRAMInfo returns total RAM in bytes
func GetRAMInfo() (uint64, error) {
	stat, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}
	return stat.Total, nil
}

// HasProxy checks if the system is configured to use a proxy
func HasProxy() bool {
	// Check common proxy environment variables
	proxyVars := []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"}
	for _, envVar := range proxyVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	// TODO: Add platform-specific proxy detection
	// For Windows: check registry settings
	// For macOS: check system preferences
	// For Linux: check various config files

	return false
}
