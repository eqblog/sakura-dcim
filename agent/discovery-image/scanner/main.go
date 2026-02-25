// sakura-scanner: Standalone hardware scanner for discovery initramfs.
// Statically compiled (CGO_ENABLED=0) and placed inside the discovery initramfs.
// Scans CPU, memory, disks, NICs, BMC and outputs JSON to stdout.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type ScanResult struct {
	MACAddress  string         `json:"mac_address"`
	IPAddress   string         `json:"ip_address"`
	System      SystemInfo     `json:"system"`
	CPU         CPUInfo        `json:"cpu"`
	Memory      MemoryInfo     `json:"memory"`
	Disks       json.RawMessage `json:"disks"`
	Network     json.RawMessage `json:"network"`
	BMC         BMCInfo        `json:"bmc"`
	DiskSummary DiskSummary    `json:"disk_summary"`
	NICCount    int            `json:"nic_count"`
}

type SystemInfo struct {
	Vendor  string `json:"vendor"`
	Product string `json:"product"`
	Serial  string `json:"serial"`
}

type CPUInfo struct {
	Model   string `json:"model"`
	Cores   int    `json:"cores"`
	Sockets int    `json:"sockets"`
}

type MemoryInfo struct {
	TotalMB int64 `json:"total_mb"`
}

type BMCInfo struct {
	IP  string `json:"ip"`
	MAC string `json:"mac"`
}

type DiskSummary struct {
	Count   int   `json:"count"`
	TotalGB int64 `json:"total_gb"`
}

func main() {
	result := ScanResult{
		MACAddress: getPrimaryMAC(),
		IPAddress:  getPrimaryIP(),
		System:     scanSystem(),
		CPU:        scanCPU(),
		Memory:     scanMemory(),
		Disks:      scanDisks(),
		Network:    scanNetwork(),
		BMC:        scanBMC(),
	}

	// Compute disk summary from lsblk output
	result.DiskSummary = computeDiskSummary(result.Disks)
	result.NICCount = countNICs(result.Network)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "json encode error: %v\n", err)
		os.Exit(1)
	}
}

func runCmd(name string, args ...string) string {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func getPrimaryMAC() string {
	// Use ip link to find first non-lo interface MAC
	out := runCmd("ip", "-o", "link", "show")
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "link/ether") && !strings.Contains(line, "lo:") {
			parts := strings.Fields(line)
			for i, p := range parts {
				if p == "link/ether" && i+1 < len(parts) {
					return parts[i+1]
				}
			}
		}
	}
	return ""
}

func getPrimaryIP() string {
	out := runCmd("ip", "-4", "-o", "addr", "show")
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, " lo ") && strings.Contains(line, "inet") {
			re := regexp.MustCompile(`inet\s+(\d+\.\d+\.\d+\.\d+)`)
			if m := re.FindStringSubmatch(line); len(m) > 1 {
				return m[1]
			}
		}
	}
	return ""
}

func scanSystem() SystemInfo {
	return SystemInfo{
		Vendor:  runCmd("dmidecode", "-s", "system-manufacturer"),
		Product: runCmd("dmidecode", "-s", "system-product-name"),
		Serial:  runCmd("dmidecode", "-s", "system-serial-number"),
	}
}

func scanCPU() CPUInfo {
	info := CPUInfo{}

	// Model name from /proc/cpuinfo
	data, err := os.ReadFile("/proc/cpuinfo")
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "model name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					info.Model = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	}

	// Core count from nproc
	if n := runCmd("nproc"); n != "" {
		info.Cores, _ = strconv.Atoi(n)
	}

	// Socket count from dmidecode
	out := runCmd("dmidecode", "-t", "processor")
	info.Sockets = strings.Count(out, "Socket Designation:")
	if info.Sockets == 0 {
		info.Sockets = 1
	}

	return info
}

func scanMemory() MemoryInfo {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return MemoryInfo{}
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			re := regexp.MustCompile(`(\d+)`)
			if m := re.FindString(line); m != "" {
				kb, _ := strconv.ParseInt(m, 10, 64)
				return MemoryInfo{TotalMB: kb / 1024}
			}
		}
	}
	return MemoryInfo{}
}

func scanDisks() json.RawMessage {
	out := runCmd("lsblk", "-J", "-b", "-d", "-o", "NAME,SIZE,TYPE,MODEL,SERIAL,TRAN")
	if out == "" {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(out)
}

func scanNetwork() json.RawMessage {
	out := runCmd("ip", "-j", "addr", "show")
	if out == "" {
		return json.RawMessage(`[]`)
	}
	return json.RawMessage(out)
}

func scanBMC() BMCInfo {
	info := BMCInfo{}
	out := runCmd("ipmitool", "lan", "print")
	if out == "" {
		return info
	}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "IP Address":
			if val != "0.0.0.0" {
				info.IP = val
			}
		case "MAC Address":
			info.MAC = val
		}
	}
	return info
}

func computeDiskSummary(raw json.RawMessage) DiskSummary {
	var parsed struct {
		BlockDevices []struct {
			Name string `json:"name"`
			Size int64  `json:"size"`
			Type string `json:"type"`
		} `json:"blockdevices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return DiskSummary{}
	}
	summary := DiskSummary{}
	for _, d := range parsed.BlockDevices {
		if d.Type == "disk" {
			summary.Count++
			summary.TotalGB += d.Size / (1024 * 1024 * 1024)
		}
	}
	return summary
}

func countNICs(raw json.RawMessage) int {
	var ifaces []struct {
		IfName string `json:"ifname"`
	}
	if err := json.Unmarshal(raw, &ifaces); err != nil {
		return 0
	}
	count := 0
	for _, iface := range ifaces {
		if iface.IfName != "lo" {
			count++
		}
	}
	return count
}
