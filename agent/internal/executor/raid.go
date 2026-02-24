package executor

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// RAIDExecutor manages hardware and software RAID configuration.
type RAIDExecutor struct {
	logger *zap.Logger
}

func NewRAIDExecutor(logger *zap.Logger) *RAIDExecutor {
	return &RAIDExecutor{logger: logger}
}

type RAIDConfigPayload struct {
	RAIDLevel string   `json:"raid_level"` // none, raid0, raid1, raid5, raid10
	Disks     []string `json:"disks"`      // device paths e.g. ["/dev/sda", "/dev/sdb"]
	ServerID  string   `json:"server_id"`
}

type RAIDStatusPayload struct {
	ServerID string `json:"server_id"`
}

// HandleRAIDConfigure sets up RAID using mdadm (software) or storcli (hardware).
func (e *RAIDExecutor) HandleRAIDConfigure(raw json.RawMessage) (interface{}, error) {
	var p RAIDConfigPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	e.logger.Info("configuring RAID",
		zap.String("server_id", p.ServerID),
		zap.String("raid_level", p.RAIDLevel),
		zap.Strings("disks", p.Disks),
	)

	if p.RAIDLevel == "none" || p.RAIDLevel == "" {
		return map[string]string{"status": "skipped", "message": "no RAID requested"}, nil
	}

	if len(p.Disks) == 0 {
		return nil, fmt.Errorf("no disks provided for RAID configuration")
	}

	// Try hardware RAID first (storcli), fall back to software RAID (mdadm)
	if e.hasStorcli() {
		return e.configureHardwareRAID(&p)
	}
	return e.configureSoftwareRAID(&p)
}

// HandleRAIDStatus checks current RAID status.
func (e *RAIDExecutor) HandleRAIDStatus(raw json.RawMessage) (interface{}, error) {
	var p RAIDStatusPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	// Check software RAID
	out, err := exec.Command("cat", "/proc/mdstat").CombinedOutput()
	if err == nil && strings.Contains(string(out), "md") {
		return map[string]string{
			"type":   "software",
			"status": strings.TrimSpace(string(out)),
		}, nil
	}

	// Check hardware RAID
	if e.hasStorcli() {
		out, err := exec.Command("storcli", "/c0", "show").CombinedOutput()
		if err == nil {
			return map[string]string{
				"type":   "hardware",
				"status": strings.TrimSpace(string(out)),
			}, nil
		}
	}

	return map[string]string{"type": "none", "status": "no RAID detected"}, nil
}

func (e *RAIDExecutor) hasStorcli() bool {
	_, err := exec.LookPath("storcli")
	if err != nil {
		_, err = exec.LookPath("storcli64")
	}
	return err == nil
}

func (e *RAIDExecutor) configureSoftwareRAID(p *RAIDConfigPayload) (interface{}, error) {
	mdLevel := ""
	switch p.RAIDLevel {
	case "raid0":
		mdLevel = "0"
	case "raid1":
		mdLevel = "1"
	case "raid5":
		mdLevel = "5"
	case "raid10":
		mdLevel = "10"
	default:
		return nil, fmt.Errorf("unsupported RAID level: %s", p.RAIDLevel)
	}

	// Stop any existing array
	exec.Command("mdadm", "--stop", "/dev/md0").Run()

	// Zero superblocks on all disks
	for _, disk := range p.Disks {
		exec.Command("mdadm", "--zero-superblock", disk).Run()
	}

	// Create RAID array
	args := []string{
		"--create", "/dev/md0",
		"--level=" + mdLevel,
		fmt.Sprintf("--raid-devices=%d", len(p.Disks)),
		"--run",
	}
	args = append(args, p.Disks...)

	cmd := exec.Command("mdadm", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("mdadm create failed: %s: %w", string(out), err)
	}

	e.logger.Info("software RAID created",
		zap.String("level", p.RAIDLevel),
		zap.Int("disks", len(p.Disks)),
	)

	return map[string]string{
		"status":  "created",
		"type":    "software",
		"device":  "/dev/md0",
		"level":   p.RAIDLevel,
		"output":  strings.TrimSpace(string(out)),
	}, nil
}

func (e *RAIDExecutor) configureHardwareRAID(p *RAIDConfigPayload) (interface{}, error) {
	storcli := "storcli"
	if _, err := exec.LookPath("storcli64"); err == nil {
		storcli = "storcli64"
	}

	raidLevel := ""
	switch p.RAIDLevel {
	case "raid0":
		raidLevel = "0"
	case "raid1":
		raidLevel = "1"
	case "raid5":
		raidLevel = "5"
	case "raid10":
		raidLevel = "10"
	default:
		return nil, fmt.Errorf("unsupported RAID level: %s", p.RAIDLevel)
	}

	// Clear existing virtual drives
	exec.Command(storcli, "/c0/vall", "del", "force").Run()

	// Create new virtual drive
	diskList := strings.Join(p.Disks, ",")
	cmd := exec.Command(storcli, "/c0", "add", "vd",
		fmt.Sprintf("r%s", raidLevel),
		fmt.Sprintf("drives=%s", diskList))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("storcli create failed: %s: %w", string(out), err)
	}

	e.logger.Info("hardware RAID created",
		zap.String("level", p.RAIDLevel),
		zap.Int("disks", len(p.Disks)),
	)

	return map[string]string{
		"status": "created",
		"type":   "hardware",
		"level":  p.RAIDLevel,
		"output": strings.TrimSpace(string(out)),
	}, nil
}
