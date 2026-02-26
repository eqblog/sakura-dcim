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

	// 6. Set channel access + user privilege (try multiple methods for BMC compatibility)
	privStr := strconv.Itoa(p.Privilege)
	channelAccessOK := false

	// Try LAN channels 1 and 2 (most BMCs use 1, some use 2)
	for _, ch := range []string{"1", "2"} {
		// Method 1: channel setaccess with combined args
		if _, err := e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass,
			"channel", "setaccess", ch, slotStr,
			fmt.Sprintf("callin=on ipmi=on link=on privilege=%s", privStr)); err == nil {
			e.logger.Info("channel setaccess succeeded", zap.String("channel", ch), zap.Int("slot", freeSlot))
			channelAccessOK = true
			break
		}
		// Method 2: channel setaccess with split args
		if _, err := e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass,
			"channel", "setaccess", ch, slotStr,
			"callin=on", "ipmi=on", "link=on", "privilege="+privStr); err == nil {
			e.logger.Info("channel setaccess (split args) succeeded", zap.String("channel", ch), zap.Int("slot", freeSlot))
			channelAccessOK = true
			break
		}
	}

	if !channelAccessOK {
		e.logger.Warn("channel setaccess failed on all channels, will rely on user priv",
			zap.Int("slot", freeSlot))
	}

	// 7. Set user privilege level (required by Dell iDRAC and some other BMCs)
	// This is separate from channel setaccess and is critical for iDRAC login.
	userPrivOK := false
	for _, ch := range []string{"1", "2"} {
		if _, err := e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass,
			"user", "priv", slotStr, privStr, ch); err == nil {
			e.logger.Info("user priv succeeded", zap.String("channel", ch), zap.Int("slot", freeSlot))
			userPrivOK = true
			break
		}
	}

	if !channelAccessOK && !userPrivOK {
		// Both methods failed — user won't be able to login. Rollback.
		e.logger.Error("failed to set any channel access or user privilege, rolling back user",
			zap.Int("slot", freeSlot))
		e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass, "user", "disable", slotStr)
		e.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass, "user", "set", "name", slotStr, "")
		return nil, fmt.Errorf("failed to grant channel access for user slot %d — tried channel setaccess and user priv on channels 1,2", freeSlot)
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
