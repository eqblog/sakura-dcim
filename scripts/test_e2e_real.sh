#!/bin/bash
# ══════════════════════════════════════════════════════════════════════
# Sakura DCIM — Real Environment E2E Test
# Tests the complete server lifecycle against a live backend with
# a mock agent simulating IPMI/PXE/KVM/Inventory hardware.
#
# Prerequisites:
#   - Backend running on localhost:8080
#   - PostgreSQL + Redis running
#   - DB seeded with admin user (admin@sakura-dcim.local / admin123)
#   - jq installed
#   - Go installed (to run mock-agent)
# ══════════════════════════════════════════════════════════════════════

set -euo pipefail

# ── Config ────────────────────────────────────────────────────────────
BASE="http://localhost:8080/api/v1"
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MOCK_PID=""

# ── Colors ────────────────────────────────────────────────────────────
RED='\033[31m'
GREEN='\033[32m'
YELLOW='\033[33m'
CYAN='\033[36m'
GRAY='\033[90m'
BOLD='\033[1m'
RESET='\033[0m'

PASSED=0
FAILED=0
TOTAL=0

pass() {
    PASSED=$((PASSED + 1))
    TOTAL=$((TOTAL + 1))
    echo -e "  ${GREEN}[PASS]${RESET} $1"
}

fail() {
    FAILED=$((FAILED + 1))
    TOTAL=$((TOTAL + 1))
    echo -e "  ${RED}[FAIL]${RESET} $1"
    if [ -n "${2:-}" ]; then
        echo -e "         ${GRAY}$2${RESET}"
    fi
}

section() {
    echo -e "\n${BOLD}${CYAN}── $1 ──${RESET}"
}

# ── Cleanup ───────────────────────────────────────────────────────────
cleanup() {
    if [ -n "$MOCK_PID" ]; then
        kill "$MOCK_PID" 2>/dev/null || true
        wait "$MOCK_PID" 2>/dev/null || true
    fi
}
trap cleanup EXIT

# ── Helper: API call ──────────────────────────────────────────────────
# Usage: api METHOD /path [json-body] [expected-status]
# Returns: response body
# Sets: LAST_STATUS (HTTP status code)
api() {
    local method="$1"
    local path="$2"
    local body="${3:-}"
    local expected="${4:-200}"

    local curl_args=(-s -w "\n%{http_code}" -X "$method")

    if [ -n "$TOKEN" ]; then
        curl_args+=(-H "Authorization: Bearer $TOKEN")
    fi

    if [ -n "$body" ]; then
        curl_args+=(-H "Content-Type: application/json" -d "$body")
    fi

    local response
    response=$(curl "${curl_args[@]}" "${BASE}${path}")

    LAST_STATUS=$(echo "$response" | tail -1)
    local body_resp
    body_resp=$(echo "$response" | sed '$d')

    echo "$body_resp"

    if [ "$LAST_STATUS" != "$expected" ]; then
        return 1
    fi
    return 0
}

# ── Helper: extract field from JSON ───────────────────────────────────
extract() {
    echo "$1" | jq -r "$2"
}

# ══════════════════════════════════════════════════════════════════════
echo -e "\n${BOLD}${GREEN}╔══════════════════════════════════════════════════╗${RESET}"
echo -e "${BOLD}${GREEN}║   Sakura DCIM — Real Environment E2E Test        ║${RESET}"
echo -e "${BOLD}${GREEN}╚══════════════════════════════════════════════════╝${RESET}\n"

# ── Check prerequisites ───────────────────────────────────────────────
if ! command -v jq &>/dev/null; then
    echo -e "${RED}Error: jq is required. Install it first.${RESET}"
    exit 1
fi

if ! curl -s "$BASE/../health" >/dev/null 2>&1; then
    echo -e "${RED}Error: Backend not running on localhost:8080${RESET}"
    echo -e "${GRAY}Start it with: cd backend && go run ./cmd/server${RESET}"
    exit 1
fi

echo -e "${GREEN}Backend is running.${RESET}"
TOKEN=""

# ══════════════════════════════════════════════════════════════════════
section "Phase 1: Authentication"
# ══════════════════════════════════════════════════════════════════════

RESP=$(api POST /auth/login '{"email":"admin@sakura-dcim.local","password":"admin123"}' 200) && {
    TOKEN=$(extract "$RESP" '.data.access_token')
    if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
        pass "Login → got access token"
    else
        fail "Login → no access_token in response"
    fi
} || fail "Login → HTTP $LAST_STATUS"

RESP=$(api GET /auth/me "" 200) && {
    EMAIL=$(extract "$RESP" '.data.email')
    if [ "$EMAIL" = "admin@sakura-dcim.local" ]; then
        pass "GetCurrentUser → $EMAIL"
    else
        fail "GetCurrentUser → unexpected email: $EMAIL"
    fi
} || fail "GetCurrentUser → HTTP $LAST_STATUS"

# ══════════════════════════════════════════════════════════════════════
section "Phase 2: Agent Creation"
# ══════════════════════════════════════════════════════════════════════

AGENT_RESP=$(api POST /agents '{"name":"Mock-DC-Tokyo","location":"Tokyo, Japan","capabilities":["ipmi","pxe","kvm","snmp","inventory"]}' 201) && {
    AGENT_ID=$(extract "$AGENT_RESP" '.data.agent.id')
    AGENT_TOKEN=$(extract "$AGENT_RESP" '.data.token')
    if [ -n "$AGENT_ID" ] && [ "$AGENT_ID" != "null" ]; then
        pass "CreateAgent → ID: $AGENT_ID"
    else
        fail "CreateAgent → no agent ID in response"
    fi
} || fail "CreateAgent → HTTP $LAST_STATUS"

# ══════════════════════════════════════════════════════════════════════
section "Phase 3: Start Mock Agent"
# ══════════════════════════════════════════════════════════════════════

cd "$PROJECT_ROOT/backend"
go run ./cmd/mock-agent \
    -server "ws://localhost:8080/api/v1/agents/ws" \
    -id "$AGENT_ID" \
    -token "$AGENT_TOKEN" &
MOCK_PID=$!
sleep 3

if kill -0 "$MOCK_PID" 2>/dev/null; then
    pass "MockAgent started (PID: $MOCK_PID)"
else
    fail "MockAgent failed to start"
    exit 1
fi

# ══════════════════════════════════════════════════════════════════════
section "Phase 4: Server CRUD"
# ══════════════════════════════════════════════════════════════════════

SERVER_RESP=$(api POST /servers "{
    \"hostname\": \"e2e-web-prod-01\",
    \"label\": \"E2E Production Server\",
    \"primary_ip\": \"10.0.1.100\",
    \"ipmi_ip\": \"10.0.0.100\",
    \"ipmi_user\": \"ADMIN\",
    \"ipmi_pass\": \"S3cretIPMI!\",
    \"agent_id\": \"$AGENT_ID\",
    \"tags\": [\"e2e\", \"production\"],
    \"notes\": \"Created by E2E test\"
}" 201) && {
    SERVER_ID=$(extract "$SERVER_RESP" '.data.id')
    pass "CreateServer → ID: $SERVER_ID"
} || fail "CreateServer → HTTP $LAST_STATUS"

RESP=$(api GET /servers "" 200) && {
    TOTAL_SERVERS=$(extract "$RESP" '.data.total')
    pass "ListServers → $TOTAL_SERVERS server(s)"
} || fail "ListServers → HTTP $LAST_STATUS"

RESP=$(api GET "/servers/$SERVER_ID" "" 200) && {
    HN=$(extract "$RESP" '.data.hostname')
    pass "GetServer → hostname: $HN"
} || fail "GetServer → HTTP $LAST_STATUS"

RESP=$(api PUT "/servers/$SERVER_ID" '{"label":"E2E Updated Server"}' 200) && {
    LBL=$(extract "$RESP" '.data.label')
    pass "UpdateServer → label: $LBL"
} || fail "UpdateServer → HTTP $LAST_STATUS"

# ══════════════════════════════════════════════════════════════════════
section "Phase 5: IPMI Power Control"
# ══════════════════════════════════════════════════════════════════════

RESP=$(api GET "/servers/$SERVER_ID/power" "" 200) && {
    STATUS=$(extract "$RESP" '.data.status')
    pass "PowerStatus → $STATUS"
} || fail "PowerStatus → HTTP $LAST_STATUS"

RESP=$(api POST "/servers/$SERVER_ID/power" '{"action":"off"}' 200) && {
    pass "PowerOff → ok"
} || fail "PowerOff → HTTP $LAST_STATUS"

RESP=$(api POST "/servers/$SERVER_ID/power" '{"action":"on"}' 200) && {
    pass "PowerOn → ok"
} || fail "PowerOn → HTTP $LAST_STATUS"

RESP=$(api POST "/servers/$SERVER_ID/power" '{"action":"cycle"}' 200) && {
    pass "PowerCycle → ok"
} || fail "PowerCycle → HTTP $LAST_STATUS"

# ══════════════════════════════════════════════════════════════════════
section "Phase 6: IPMI Sensors"
# ══════════════════════════════════════════════════════════════════════

RESP=$(api GET "/servers/$SERVER_ID/sensors" "" 200) && {
    COUNT=$(extract "$RESP" '.data.sensors | length')
    pass "ReadSensors → $COUNT sensors"
} || fail "ReadSensors → HTTP $LAST_STATUS"

# ══════════════════════════════════════════════════════════════════════
section "Phase 7: Inventory Scan"
# ══════════════════════════════════════════════════════════════════════

RESP=$(api POST "/servers/$SERVER_ID/inventory/scan" "" 200) && {
    pass "TriggerInventoryScan → ok"
} || fail "TriggerInventoryScan → HTTP $LAST_STATUS"

echo -e "  ${GRAY}Waiting 3s for async inventory.result event...${RESET}"
sleep 3

RESP=$(api GET "/servers/$SERVER_ID/inventory" "" 200) && {
    COMPONENTS=$(extract "$RESP" '.data.components | length')
    pass "ViewInventory → $COMPONENTS component(s)"
} || fail "ViewInventory → HTTP $LAST_STATUS"

# ══════════════════════════════════════════════════════════════════════
section "Phase 8: Network — Switches & Ports"
# ══════════════════════════════════════════════════════════════════════

SWITCH_RESP=$(api POST /switches "{
    \"name\": \"E2E-TOR-Switch-01\",
    \"ip\": \"10.0.0.1\",
    \"vendor\": \"cisco_ios\",
    \"model\": \"Nexus 9300\",
    \"snmp_community\": \"public\",
    \"snmp_version\": \"v2c\",
    \"ssh_user\": \"admin\",
    \"ssh_pass\": \"switchpass\",
    \"ssh_port\": 22,
    \"agent_id\": \"$AGENT_ID\"
}" 201) && {
    SWITCH_ID=$(extract "$SWITCH_RESP" '.data.id')
    pass "CreateSwitch → ID: $SWITCH_ID"
} || fail "CreateSwitch → HTTP $LAST_STATUS"

PORT_RESP=$(api POST "/switches/$SWITCH_ID/ports" "{
    \"port_index\": 1,
    \"port_name\": \"Ethernet1/1\",
    \"speed_mbps\": 10000,
    \"vlan_id\": 100,
    \"admin_status\": \"up\",
    \"server_id\": \"$SERVER_ID\",
    \"description\": \"e2e-web-prod-01 uplink\"
}" 201) && {
    PORT_ID=$(extract "$PORT_RESP" '.data.id')
    pass "CreateSwitchPort → ID: $PORT_ID"
} || fail "CreateSwitchPort → HTTP $LAST_STATUS"

RESP=$(api POST "/switches/$SWITCH_ID/ports/$PORT_ID/provision" "" 200) && {
    pass "ProvisionPort → ok"
} || fail "ProvisionPort → HTTP $LAST_STATUS"

RESP=$(api GET "/switches/$SWITCH_ID/ports/$PORT_ID/status" "" 200) && {
    pass "GetPortStatus → ok"
} || fail "GetPortStatus → HTTP $LAST_STATUS"

# ══════════════════════════════════════════════════════════════════════
section "Phase 9: IP Address Management"
# ══════════════════════════════════════════════════════════════════════

POOL_RESP=$(api POST /ip-pools '{"network":"10.0.1.0/24","gateway":"10.0.1.1","description":"E2E Production VLAN"}' 201) && {
    POOL_ID=$(extract "$POOL_RESP" '.data.id')
    pass "CreateIPPool → ID: $POOL_ID"
} || fail "CreateIPPool → HTTP $LAST_STATUS"

for ADDR in "10.0.1.10" "10.0.1.11" "10.0.1.12"; do
    api POST "/ip-pools/$POOL_ID/addresses" "{\"address\":\"$ADDR\",\"status\":\"available\"}" 201 >/dev/null && {
        pass "CreateAddress → $ADDR"
    } || fail "CreateAddress → $ADDR HTTP $LAST_STATUS"
done

RESP=$(api GET "/ip-pools/$POOL_ID/addresses" "" 200) && {
    pass "ListAddresses → ok"
} || fail "ListAddresses → HTTP $LAST_STATUS"

RESP=$(api POST "/ip-pools/$POOL_ID/assign" "{\"server_id\":\"$SERVER_ID\"}" 200) && {
    ASSIGNED_IP=$(extract "$RESP" '.data.address')
    pass "AutoAssignIP → $ASSIGNED_IP"
} || fail "AutoAssignIP → HTTP $LAST_STATUS"

# ══════════════════════════════════════════════════════════════════════
section "Phase 10: OS Reinstallation"
# ══════════════════════════════════════════════════════════════════════

OS_RESP=$(api POST /os-profiles '{
    "name": "E2E Ubuntu 22.04",
    "os_family": "ubuntu",
    "version": "22.04",
    "arch": "amd64",
    "kernel_url": "http://archive.ubuntu.com/ubuntu/dists/jammy/main/installer-amd64/current/legacy-images/netboot/ubuntu-installer/amd64/linux",
    "initrd_url": "http://archive.ubuntu.com/ubuntu/dists/jammy/main/installer-amd64/current/legacy-images/netboot/ubuntu-installer/amd64/initrd.gz",
    "boot_args": "auto=true priority=critical",
    "template_type": "preseed",
    "template": "d-i debian-installer/locale string en_US",
    "is_active": true
}' 201) && {
    OS_PROFILE_ID=$(extract "$OS_RESP" '.data.id')
    pass "CreateOSProfile → ID: $OS_PROFILE_ID"
} || fail "CreateOSProfile → HTTP $LAST_STATUS"

LAYOUT_RESP=$(api POST /disk-layouts '{
    "name": "E2E Standard Layout",
    "description": "Boot + Root",
    "layout": {"partitions": [{"mount":"/boot","size":"1G","fs":"ext4"},{"mount":"/","size":"100%FREE","fs":"ext4"}]}
}' 201) && {
    LAYOUT_ID=$(extract "$LAYOUT_RESP" '.data.id')
    pass "CreateDiskLayout → ID: $LAYOUT_ID"
} || fail "CreateDiskLayout → HTTP $LAST_STATUS"

RESP=$(api POST "/servers/$SERVER_ID/reinstall" "{
    \"os_profile_id\": \"$OS_PROFILE_ID\",
    \"disk_layout_id\": \"$LAYOUT_ID\",
    \"raid_level\": \"auto\",
    \"root_password\": \"E2ESecurePass123!\",
    \"ssh_keys\": [\"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI... e2e@test\"]
}" 200) && {
    pass "StartReinstall → ok"
} || fail "StartReinstall → HTTP $LAST_STATUS"

echo -e "  ${GRAY}Waiting 8s for PXE install simulation...${RESET}"
sleep 8

RESP=$(api GET "/servers/$SERVER_ID/reinstall/status" "" 200) && {
    INSTALL_STATUS=$(extract "$RESP" '.data.status')
    INSTALL_PROGRESS=$(extract "$RESP" '.data.progress')
    if [ "$INSTALL_STATUS" = "completed" ]; then
        pass "ReinstallStatus → $INSTALL_STATUS ($INSTALL_PROGRESS%)"
    else
        fail "ReinstallStatus → expected 'completed', got '$INSTALL_STATUS' ($INSTALL_PROGRESS%)"
    fi
} || fail "ReinstallStatus → HTTP $LAST_STATUS"

# ══════════════════════════════════════════════════════════════════════
section "Phase 11: KVM Console"
# ══════════════════════════════════════════════════════════════════════

KVM_RESP=$(api POST "/servers/$SERVER_ID/kvm" "" 200) && {
    SESSION_ID=$(extract "$KVM_RESP" '.data.session_id')
    WS_URL=$(extract "$KVM_RESP" '.data.ws_url')
    pass "StartKVM → session: $SESSION_ID"
    echo -e "  ${GRAY}         ws_url: $WS_URL${RESET}"
} || fail "StartKVM → HTTP $LAST_STATUS"

RESP=$(api DELETE "/servers/$SERVER_ID/kvm?session=$SESSION_ID" "" 200) && {
    pass "StopKVM → ok"
} || fail "StopKVM → HTTP $LAST_STATUS"

# ══════════════════════════════════════════════════════════════════════
section "Phase 12: Audit Logs"
# ══════════════════════════════════════════════════════════════════════

RESP=$(api GET "/audit-logs?page=1&page_size=50" "" 200) && {
    LOG_TOTAL=$(extract "$RESP" '.data.total')
    pass "ViewAuditLogs → $LOG_TOTAL entries"
} || fail "ViewAuditLogs → HTTP $LAST_STATUS"

# ══════════════════════════════════════════════════════════════════════
section "Phase 13: Cleanup"
# ══════════════════════════════════════════════════════════════════════

RESP=$(api DELETE "/servers/$SERVER_ID" "" 200) && {
    pass "DeleteServer → ok"
} || fail "DeleteServer → HTTP $LAST_STATUS"

RESP=$(api GET "/servers/$SERVER_ID" "" 404) && {
    pass "VerifyDeleted → 404"
} || fail "VerifyDeleted → expected 404, got HTTP $LAST_STATUS"

# ══════════════════════════════════════════════════════════════════════
# Summary
# ══════════════════════════════════════════════════════════════════════

echo -e "\n${BOLD}══════════════════════════════════════════════════${RESET}"
if [ "$FAILED" -eq 0 ]; then
    echo -e "${BOLD}${GREEN}  Results: $PASSED passed, $FAILED failed out of $TOTAL tests${RESET}"
    echo -e "${GREEN}  All tests passed!${RESET}"
else
    echo -e "${BOLD}${RED}  Results: $PASSED passed, $FAILED failed out of $TOTAL tests${RESET}"
fi
echo -e "${BOLD}══════════════════════════════════════════════════${RESET}\n"

# Stop mock agent
cleanup
MOCK_PID=""

exit $FAILED
