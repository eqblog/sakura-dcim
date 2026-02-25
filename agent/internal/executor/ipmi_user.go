package executor

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

// TempUserCreatePayload is sent by the backend to create a temporary IPMI user.
type TempUserCreatePayload struct {
	IPMIIP    string `json:"ipmi_ip"`
	IPMIUser  string `json:"ipmi_user"`
	IPMIPass  string `json:"ipmi_pass"`
	BMCType   string `json:"bmc_type"`
	Privilege int    `json:"privilege"` // IPMI privilege: 3=Operator, 4=Admin
}

// TempUserDeletePayload is sent by the backend to remove a temporary IPMI user.
type TempUserDeletePayload struct {
	IPMIIP   string `json:"ipmi_ip"`
	IPMIUser string `json:"ipmi_user"`
	IPMIPass string `json:"ipmi_pass"`
	BMCType  string `json:"bmc_type"`
	UserSlot int    `json:"user_slot"`
}

// HandleCreateTempUser creates a temporary IPMI user on the BMC.
// It finds a free user slot (3–15), sets a random username/password,
// enables the user, and grants channel access with the requested privilege.
func (e *IPMIExecutor) HandleCreateTempUser(raw json.RawMessage) (interface{}, error) {
	var p TempUserCreatePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}
	if p.IPMIIP == "" || p.IPMIUser == "" || p.IPMIPass == "" {
		return nil, fmt.Errorf("ipmi_ip, ipmi_user, ipmi_pass are required")
	}
	if p.Privilege < 2 || p.Privilege > 4 {
		p.Privilege = 3 // default to Operator
	}

	// 1. List existing users to find a free slot
	listOut, err := e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass, "user", "list")
	if err != nil {
		return nil, fmt.Errorf("user list failed: %w", err)
	}

	freeSlot := findFreeUserSlot(listOut)
	if freeSlot < 0 {
		return nil, fmt.Errorf("no free IPMI user slot available (slots 3-15 all occupied)")
	}

	// 2. Generate random credentials
	username := generateTempUsername()
	password := generateTempPassword()
	slotStr := strconv.Itoa(freeSlot)

	e.logger.Info("creating temp IPMI user",
		zap.String("ip", p.IPMIIP),
		zap.Int("slot", freeSlot),
		zap.String("username", username),
	)

	// 3. Set username
	if _, err := e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass, "user", "set", "name", slotStr, username); err != nil {
		return nil, fmt.Errorf("set username (slot %d): %w", freeSlot, err)
	}

	// 4. Set password
	if _, err := e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass, "user", "set", "password", slotStr, password); err != nil {
		// Rollback: clear the username we just set
		e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass, "user", "set", "name", slotStr, "")
		return nil, fmt.Errorf("set password (slot %d): %w", freeSlot, err)
	}

	// 5. Enable user
	if _, err := e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass, "user", "enable", slotStr); err != nil {
		e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass, "user", "set", "name", slotStr, "")
		return nil, fmt.Errorf("enable user (slot %d): %w", freeSlot, err)
	}

	// 6. Set channel access (channel 1 = LAN)
	privStr := strconv.Itoa(p.Privilege)
	accessArgs := fmt.Sprintf("callin=on ipmi=on link=on privilege=%s", privStr)
	if _, err := e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass,
		"channel", "setaccess", "1", slotStr, accessArgs); err != nil {
		// Non-fatal: some BMCs have different channel numbers or syntax.
		// Try alternate syntax (split arguments).
		e.logger.Warn("channel setaccess failed with combined args, trying split args",
			zap.Int("slot", freeSlot), zap.Error(err))
		if _, err2 := e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass,
			"channel", "setaccess", "1", slotStr,
			"callin=on", "ipmi=on", "link=on", "privilege="+privStr); err2 != nil {
			e.logger.Warn("channel setaccess also failed with split args, user may have limited access",
				zap.Int("slot", freeSlot), zap.Error(err2))
		}
	}

	e.logger.Info("temp IPMI user created",
		zap.String("ip", p.IPMIIP),
		zap.Int("slot", freeSlot),
		zap.String("username", username),
	)

	return map[string]interface{}{
		"username":  username,
		"password":  password,
		"user_slot": freeSlot,
	}, nil
}

// HandleDeleteTempUser removes a temporary IPMI user from the BMC.
func (e *IPMIExecutor) HandleDeleteTempUser(raw json.RawMessage) (interface{}, error) {
	var p TempUserDeletePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}
	if p.IPMIIP == "" || p.IPMIUser == "" || p.IPMIPass == "" {
		return nil, fmt.Errorf("ipmi_ip, ipmi_user, ipmi_pass are required")
	}
	if p.UserSlot < 3 || p.UserSlot > 15 {
		return nil, fmt.Errorf("invalid user_slot %d (must be 3-15)", p.UserSlot)
	}

	slotStr := strconv.Itoa(p.UserSlot)

	e.logger.Info("deleting temp IPMI user",
		zap.String("ip", p.IPMIIP),
		zap.Int("slot", p.UserSlot),
	)

	// Disable user first
	if _, err := e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass, "user", "disable", slotStr); err != nil {
		e.logger.Warn("disable user failed", zap.Int("slot", p.UserSlot), zap.Error(err))
	}

	// Clear username (effectively deletes the user)
	if _, err := e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass, "user", "set", "name", slotStr, ""); err != nil {
		e.logger.Warn("clear username failed", zap.Int("slot", p.UserSlot), zap.Error(err))
		return nil, fmt.Errorf("clear username (slot %d): %w", p.UserSlot, err)
	}

	e.logger.Info("temp IPMI user deleted",
		zap.String("ip", p.IPMIIP),
		zap.Int("slot", p.UserSlot),
	)

	return map[string]string{"status": "deleted"}, nil
}

// findFreeUserSlot parses `ipmitool user list` output and returns the first
// free slot between 3 and 15. Returns -1 if none available.
//
// Typical output:
//
//	ID  Name             Callin  Link Auth  IPMI Msg   Channel Priv Limit
//	1                    false   false      true       ADMINISTRATOR
//	2   admin            true    true       true       ADMINISTRATOR
//	3                    true    false      false      NO ACCESS
//	4   operator         true    true       true       OPERATOR
//	...
func findFreeUserSlot(output string) int {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		id, err := strconv.Atoi(fields[0])
		if err != nil {
			continue // header line or non-numeric
		}
		if id < 3 || id > 15 {
			continue // skip slot 1 (null) and 2 (admin)
		}

		// A slot is free if:
		// - The name field is empty (ID is directly followed by true/false)
		// - Or the name is "(Empty User)" on some BMCs
		// When name is empty, fields[1] will be "true" or "false" (the Callin column)
		name := fields[1]
		if name == "true" || name == "false" {
			// Empty name — this slot is free
			return id
		}
		if strings.EqualFold(name, "(Empty") {
			// "(Empty User)" — free slot
			return id
		}
	}
	return -1
}

// generateTempUsername returns a username like "kvm-a1b2c3d4".
func generateTempUsername() string {
	b := make([]byte, 4)
	rand.Read(b)
	return "kvm-" + hex.EncodeToString(b)
}

// generateTempPassword returns a 16-character random alphanumeric password.
func generateTempPassword() string {
	const charset = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 16)
	rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}
