#!/usr/bin/env bash
# ============================================================
# Sakura DCIM — Agent One-Click Deployment Script
# ============================================================
# Usage:
#   bash deploy-agent.sh [OPTIONS]
#
# Modes:
#   --mode binary   Install as systemd service (default)
#   --mode docker   Run as Docker container
#   --uninstall     Remove agent from this machine
#
# Examples:
#   # Interactive
#   bash deploy-agent.sh
#
#   # Full CLI (for CI / Ansible)
#   bash deploy-agent.sh \
#     --server https://panel.example.com \
#     --email admin@sakura-dcim.local --password admin123 \
#     --name "DC1-Agent-01" --location "Tokyo DC1" --mode binary
#
#   # Existing agent (skip registration)
#   bash deploy-agent.sh \
#     --server https://panel.example.com \
#     --agent-id UUID --token TOKEN --mode binary
# ============================================================

set -euo pipefail

# ── Colors ───────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; GRAY='\033[90m'; NC='\033[0m'

info()  { echo -e "  ${CYAN}[INFO]${NC} $*"; }
ok()    { echo -e "  ${GREEN}[OK]${NC}   $*"; }
warn()  { echo -e "  ${YELLOW}[WARN]${NC} $*"; }
err()   { echo -e "  ${RED}[FAIL]${NC} $*"; }
die()   { err "$*"; exit 1; }

banner() {
    echo -e "\n${GREEN}╔══════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║     Sakura DCIM Agent Deployment          ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════╝${NC}\n"
}

# ── Defaults ─────────────────────────────────────────────────
INSTALL_DIR="/opt/sakura-agent"
SERVICE_NAME="sakura-agent"
BINARY_NAME="sakura-agent"
DOCKER_IMAGE="sakura-dcim/agent:latest"

SERVER_URL=""
ADMIN_EMAIL=""
ADMIN_PASSWORD=""
AGENT_NAME=""
AGENT_LOCATION=""
AGENT_ID=""
AGENT_TOKEN=""
INSTALL_MODE="binary"
DO_UNINSTALL=false

# ── Parse Arguments ──────────────────────────────────────────
show_help() {
    cat <<'HELP'
Sakura DCIM Agent Deployment Script

Options:
  --server URL        Backend panel URL (e.g., https://panel.example.com)
  --email EMAIL       Admin email for auto-registration
  --password PASS     Admin password for auto-registration
  --name NAME         Agent display name (e.g., DC1-Agent-01)
  --location LOC      Agent location (e.g., Tokyo DC1)
  --agent-id UUID     Existing agent UUID (skip registration)
  --token TOKEN       Existing agent token (skip registration)
  --mode MODE         Install mode: binary (default) or docker
  --uninstall         Remove agent from this machine
  -h, --help          Show this help
HELP
    exit 0
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --server)       SERVER_URL="$2";      shift 2 ;;
        --email)        ADMIN_EMAIL="$2";     shift 2 ;;
        --password)     ADMIN_PASSWORD="$2";  shift 2 ;;
        --name)         AGENT_NAME="$2";      shift 2 ;;
        --location)     AGENT_LOCATION="$2";  shift 2 ;;
        --agent-id)     AGENT_ID="$2";        shift 2 ;;
        --token)        AGENT_TOKEN="$2";     shift 2 ;;
        --mode)         INSTALL_MODE="$2";    shift 2 ;;
        --uninstall)    DO_UNINSTALL=true;    shift   ;;
        -h|--help)      show_help ;;
        *) die "Unknown option: $1. Use --help for usage." ;;
    esac
done

# ── JSON helpers (no jq dependency) ──────────────────────────
# Extract a top-level string value from JSON: json_get '{"a":"b"}' a → b
json_get() {
    local json="$1" key="$2"
    echo "$json" | grep -o "\"${key}\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" | head -1 | sed "s/\"${key}\"[[:space:]]*:[[:space:]]*\"//;s/\"$//"
}

# Extract a nested .data.field value
json_get_data() {
    local json="$1" key="$2"
    # First extract the data object, then extract the field
    local data
    data=$(echo "$json" | grep -o '"data"[[:space:]]*:[[:space:]]*{[^}]*}' | head -1)
    if [[ -z "$data" ]]; then
        # Try nested data (e.g., data.agent)
        data=$(echo "$json" | grep -o '"data"[[:space:]]*:[[:space:]]*{[^{]*{[^}]*}[^}]*}' | head -1)
    fi
    echo "$data" | grep -o "\"${key}\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" | head -1 | sed "s/\"${key}\"[[:space:]]*:[[:space:]]*\"//;s/\"$//"
}

# Check if JSON has "success":true
json_success() {
    echo "$1" | grep -q '"success"[[:space:]]*:[[:space:]]*true'
}

# ── Uninstall ────────────────────────────────────────────────
do_uninstall() {
    banner
    info "Uninstalling Sakura DCIM Agent..."

    # Systemd service
    if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
        info "Stopping service..."
        systemctl stop "$SERVICE_NAME"
    fi
    if systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
        systemctl disable "$SERVICE_NAME" 2>/dev/null
    fi
    if [[ -f "/etc/systemd/system/${SERVICE_NAME}.service" ]]; then
        rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
        systemctl daemon-reload
        ok "Systemd service removed"
    fi

    # Docker container
    if docker ps -a --format '{{.Names}}' 2>/dev/null | grep -q "^${SERVICE_NAME}$"; then
        docker stop "$SERVICE_NAME" 2>/dev/null || true
        docker rm "$SERVICE_NAME" 2>/dev/null || true
        ok "Docker container removed"
    fi

    # Install directory
    if [[ -d "$INSTALL_DIR" ]]; then
        rm -rf "$INSTALL_DIR"
        ok "Removed $INSTALL_DIR"
    fi

    echo ""
    ok "Agent uninstalled successfully."
    echo ""
    exit 0
}

if $DO_UNINSTALL; then
    do_uninstall
fi

# ── Main Flow ────────────────────────────────────────────────
banner

# ── Step 1: Environment Detection ────────────────────────────
info "Detecting environment..."

# Root check
if [[ $EUID -ne 0 ]] && [[ "$INSTALL_MODE" == "binary" ]]; then
    die "Binary mode requires root. Run with: sudo bash $0 $*"
fi

# OS detection
OS_ID="unknown"
OS_VERSION=""
if [[ -f /etc/os-release ]]; then
    . /etc/os-release
    OS_ID="${ID,,}"
    OS_VERSION="$VERSION_ID"
elif [[ -f /etc/redhat-release ]]; then
    OS_ID="rhel"
fi

# Architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  GO_ARCH="amd64" ;;
    aarch64) GO_ARCH="arm64" ;;
    armv7l)  GO_ARCH="arm"   ;;
    *)       GO_ARCH="$ARCH" ;;
esac

# Package manager
PKG_MGR="unknown"
case "$OS_ID" in
    ubuntu|debian)               PKG_MGR="apt" ;;
    centos|rhel|rocky|alma|fedora) PKG_MGR="yum" ;;
    alpine)                      PKG_MGR="apk" ;;
esac

info "OS: ${OS_ID} ${OS_VERSION} (${ARCH})"
info "Package manager: ${PKG_MGR}"

# curl check
if ! command -v curl &>/dev/null; then
    die "curl is required but not installed. Please install curl first."
fi

# ── Step 2: Interactive Prompts (if needed) ──────────────────
if [[ -z "$SERVER_URL" ]]; then
    echo ""
    read -rp "  Backend URL (e.g., https://panel.example.com): " SERVER_URL
fi
[[ -z "$SERVER_URL" ]] && die "Backend URL is required."
# Strip trailing slash
SERVER_URL="${SERVER_URL%/}"

# Determine API base URL
API_BASE="${SERVER_URL}/api/v1"

# Need registration?
NEED_REGISTER=false
if [[ -z "$AGENT_ID" || -z "$AGENT_TOKEN" ]]; then
    NEED_REGISTER=true

    if [[ -z "$ADMIN_EMAIL" ]]; then
        read -rp "  Admin email: " ADMIN_EMAIL
    fi
    if [[ -z "$ADMIN_PASSWORD" ]]; then
        read -srp "  Admin password: " ADMIN_PASSWORD
        echo ""
    fi
    if [[ -z "$AGENT_NAME" ]]; then
        HOSTNAME_DEFAULT=$(hostname -s 2>/dev/null || echo "agent-01")
        read -rp "  Agent name [${HOSTNAME_DEFAULT}]: " AGENT_NAME
        AGENT_NAME="${AGENT_NAME:-$HOSTNAME_DEFAULT}"
    fi
    if [[ -z "$AGENT_LOCATION" ]]; then
        read -rp "  Agent location [Default]: " AGENT_LOCATION
        AGENT_LOCATION="${AGENT_LOCATION:-Default}"
    fi
fi

info "Backend: ${SERVER_URL}"
info "Install mode: ${INSTALL_MODE}"

# ── Step 3: Agent Registration ───────────────────────────────
if $NEED_REGISTER; then
    info "Authenticating with backend..."

    # Login
    LOGIN_RESP=$(curl -sS -w "\n%{http_code}" \
        -X POST "${API_BASE}/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"${ADMIN_EMAIL}\",\"password\":\"${ADMIN_PASSWORD}\"}" \
        2>&1) || die "Cannot reach backend at ${SERVER_URL}"

    HTTP_CODE=$(echo "$LOGIN_RESP" | tail -1)
    LOGIN_BODY=$(echo "$LOGIN_RESP" | sed '$d')

    if [[ "$HTTP_CODE" != "200" ]]; then
        die "Login failed (HTTP ${HTTP_CODE}): $(json_get "$LOGIN_BODY" error)"
    fi

    JWT_TOKEN=$(json_get_data "$LOGIN_BODY" "access_token")
    if [[ -z "$JWT_TOKEN" ]]; then
        die "Failed to extract access token from login response"
    fi
    ok "Login successful"

    # Register agent
    info "Registering agent: ${AGENT_NAME}..."

    REGISTER_RESP=$(curl -sS -w "\n%{http_code}" \
        -X POST "${API_BASE}/agents" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${JWT_TOKEN}" \
        -d "{\"name\":\"${AGENT_NAME}\",\"location\":\"${AGENT_LOCATION}\",\"capabilities\":[\"ipmi\",\"pxe\",\"kvm\",\"inventory\",\"snmp\"]}" \
        2>&1)

    HTTP_CODE=$(echo "$REGISTER_RESP" | tail -1)
    REGISTER_BODY=$(echo "$REGISTER_RESP" | sed '$d')

    if [[ "$HTTP_CODE" != "201" ]]; then
        die "Agent registration failed (HTTP ${HTTP_CODE}): $(json_get "$REGISTER_BODY" error)"
    fi

    AGENT_ID=$(json_get_data "$REGISTER_BODY" "id")
    AGENT_TOKEN=$(json_get "$REGISTER_BODY" "token")

    if [[ -z "$AGENT_ID" || -z "$AGENT_TOKEN" ]]; then
        die "Failed to extract agent_id or token from registration response"
    fi

    ok "Agent registered: ${AGENT_NAME}"
    echo -e "       Agent ID:  ${BOLD}${AGENT_ID}${NC}"
    TOKEN_MASKED="${AGENT_TOKEN:0:4}$(printf '*%.0s' $(seq 1 $((${#AGENT_TOKEN}-8))))${AGENT_TOKEN: -4}"
    echo -e "       Token:     ${GRAY}${TOKEN_MASKED}${NC}"
    echo -e "       ${YELLOW}Save this token! It will NOT be shown again.${NC}"
    echo ""
fi

# ── Build WebSocket URL ──────────────────────────────────────
# https://xxx → wss://xxx/api/v1/agents/ws
# http://xxx  → ws://xxx/api/v1/agents/ws
WS_URL="${SERVER_URL}"
WS_URL="${WS_URL/https:\/\//wss://}"
WS_URL="${WS_URL/http:\/\//ws://}"
WS_URL="${WS_URL}/api/v1/agents/ws"

# ── Step 4: Install ──────────────────────────────────────────

install_binary() {
    # 4a. Install system dependencies
    info "Installing system dependencies..."
    case "$PKG_MGR" in
        apt)
            apt-get update -qq
            apt-get install -y -qq ipmitool lshw dmidecode ca-certificates >/dev/null 2>&1
            ;;
        yum)
            yum install -y -q ipmitool lshw dmidecode ca-certificates >/dev/null 2>&1
            ;;
        apk)
            apk add --no-cache ipmitool lshw dmidecode ca-certificates >/dev/null 2>&1
            ;;
        *)
            warn "Unknown package manager. Please install manually: ipmitool lshw dmidecode"
            ;;
    esac
    ok "Dependencies installed (ipmitool, lshw, dmidecode)"

    # 4b. Create install directory
    mkdir -p "$INSTALL_DIR"

    # 4c. Download or locate binary
    BINARY_URL="${SERVER_URL}/downloads/${BINARY_NAME}-linux-${GO_ARCH}"
    info "Downloading agent binary..."
    HTTP_CODE=$(curl -sS -o "${INSTALL_DIR}/${BINARY_NAME}" -w "%{http_code}" "$BINARY_URL" 2>/dev/null || echo "000")

    if [[ "$HTTP_CODE" != "200" ]]; then
        warn "Binary download not available (HTTP ${HTTP_CODE})"

        # Check if binary exists locally (development mode)
        LOCAL_BINARY=""
        for p in "./agent/bin/${BINARY_NAME}" "../agent/bin/${BINARY_NAME}" "/tmp/${BINARY_NAME}"; do
            if [[ -f "$p" ]]; then
                LOCAL_BINARY="$p"
                break
            fi
        done

        if [[ -n "$LOCAL_BINARY" ]]; then
            info "Using local binary: ${LOCAL_BINARY}"
            cp "$LOCAL_BINARY" "${INSTALL_DIR}/${BINARY_NAME}"
        else
            # Try to build from source if Go is available
            if command -v go &>/dev/null; then
                warn "Attempting source build..."
                SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
                PROJECT_ROOT="${SCRIPT_DIR}/.."
                if [[ -f "${PROJECT_ROOT}/agent/cmd/agent/main.go" ]]; then
                    (cd "${PROJECT_ROOT}/agent" && CGO_ENABLED=0 GOOS=linux GOARCH="${GO_ARCH}" go build -o "${INSTALL_DIR}/${BINARY_NAME}" ./cmd/agent)
                    ok "Built from source"
                else
                    die "Cannot find agent source code. Please build manually:\n         cd agent && CGO_ENABLED=0 go build -o ${INSTALL_DIR}/${BINARY_NAME} ./cmd/agent"
                fi
            else
                die "Binary not available and Go not installed.\n         Build on another machine: cd agent && CGO_ENABLED=0 GOOS=linux GOARCH=${GO_ARCH} go build -o ${BINARY_NAME} ./cmd/agent\n         Then copy to ${INSTALL_DIR}/${BINARY_NAME} and re-run this script with --agent-id and --token"
            fi
        fi
    else
        ok "Binary downloaded"
    fi

    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

    # 4d. Write config.yaml
    cat > "${INSTALL_DIR}/config.yaml" <<YAML
# Sakura DCIM Agent Configuration
# Auto-generated by deploy-agent.sh on $(date -Iseconds)
server_url: "${WS_URL}"
agent_id: "${AGENT_ID}"
token: "${AGENT_TOKEN}"
YAML
    chmod 600 "${INSTALL_DIR}/config.yaml"
    ok "Config written  -> ${INSTALL_DIR}/config.yaml"

    # 4e. Create systemd service
    cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<SERVICE
[Unit]
Description=Sakura DCIM Agent
Documentation=https://github.com/sakura-dcim/sakura-dcim
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/${BINARY_NAME} -config ${INSTALL_DIR}/config.yaml
Restart=always
RestartSec=5
LimitNOFILE=65536
Environment=LANG=en_US.UTF-8

# Security hardening
NoNewPrivileges=false
ProtectSystem=full
ProtectHome=true
ReadWritePaths=${INSTALL_DIR}

[Install]
WantedBy=multi-user.target
SERVICE
    ok "Systemd service created"

    # 4f. Enable and start
    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME" --quiet
    systemctl start "$SERVICE_NAME"
    ok "Service started: systemctl status ${SERVICE_NAME}"
}

install_docker() {
    # Check Docker
    if ! command -v docker &>/dev/null; then
        die "Docker is not installed. Install Docker first: https://docs.docker.com/engine/install/"
    fi

    # Stop existing container
    if docker ps -a --format '{{.Names}}' | grep -q "^${SERVICE_NAME}$"; then
        info "Stopping existing container..."
        docker stop "$SERVICE_NAME" 2>/dev/null || true
        docker rm "$SERVICE_NAME" 2>/dev/null || true
    fi

    # Create config directory
    mkdir -p "$INSTALL_DIR"

    # Write config.yaml
    cat > "${INSTALL_DIR}/config.yaml" <<YAML
# Sakura DCIM Agent Configuration
# Auto-generated by deploy-agent.sh on $(date -Iseconds)
server_url: "${WS_URL}"
agent_id: "${AGENT_ID}"
token: "${AGENT_TOKEN}"
YAML
    chmod 600 "${INSTALL_DIR}/config.yaml"
    ok "Config written  -> ${INSTALL_DIR}/config.yaml"

    # Build or pull image
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    PROJECT_ROOT="${SCRIPT_DIR}/.."
    DOCKERFILE="${PROJECT_ROOT}/docker/agent.Dockerfile"

    if [[ -f "$DOCKERFILE" ]]; then
        info "Building Docker image from source..."
        docker build -t "$DOCKER_IMAGE" -f "$DOCKERFILE" "${PROJECT_ROOT}/agent" >/dev/null 2>&1
        ok "Docker image built: ${DOCKER_IMAGE}"
    else
        info "Pulling Docker image: ${DOCKER_IMAGE}..."
        if ! docker pull "$DOCKER_IMAGE" 2>/dev/null; then
            die "Cannot pull ${DOCKER_IMAGE} and Dockerfile not found locally."
        fi
        ok "Docker image pulled"
    fi

    # Run container
    docker run -d \
        --name "$SERVICE_NAME" \
        --restart always \
        --network host \
        --privileged \
        -v "${INSTALL_DIR}/config.yaml:/app/config.yaml:ro" \
        "$DOCKER_IMAGE" >/dev/null

    ok "Container started: docker logs -f ${SERVICE_NAME}"
}

case "$INSTALL_MODE" in
    binary) install_binary ;;
    docker) install_docker ;;
    *)      die "Unknown install mode: ${INSTALL_MODE}. Use 'binary' or 'docker'." ;;
esac

# ── Step 5: Verify Connection ────────────────────────────────
echo ""
info "Verifying agent connection (waiting up to 15s)..."

ONLINE=false
for i in $(seq 1 15); do
    sleep 1

    if $NEED_REGISTER; then
        VERIFY_RESP=$(curl -sS \
            -H "Authorization: Bearer ${JWT_TOKEN}" \
            "${API_BASE}/agents/${AGENT_ID}" 2>/dev/null || echo "")
    else
        # Without JWT we can only check if the process is running
        if [[ "$INSTALL_MODE" == "binary" ]]; then
            if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
                ONLINE=true
                break
            fi
        else
            if docker ps --format '{{.Names}}' | grep -q "^${SERVICE_NAME}$"; then
                ONLINE=true
                break
            fi
        fi
        continue
    fi

    if [[ -n "$VERIFY_RESP" ]]; then
        STATUS=$(json_get_data "$VERIFY_RESP" "status")
        if [[ "$STATUS" == "online" ]]; then
            ONLINE=true
            break
        fi
    fi
    printf "\r  ${GRAY}[....]${NC} Waiting... ${i}s"
done
echo ""

if $ONLINE; then
    ok "Agent is ${GREEN}ONLINE${NC}"
else
    warn "Agent may still be connecting. Check logs:"
    if [[ "$INSTALL_MODE" == "binary" ]]; then
        echo -e "       ${GRAY}journalctl -u ${SERVICE_NAME} -f${NC}"
    else
        echo -e "       ${GRAY}docker logs -f ${SERVICE_NAME}${NC}"
    fi
fi

# ── Step 6: Summary ──────────────────────────────────────────
echo ""
echo -e "${GREEN}╔══════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  ${BOLD}Deployment Complete!${NC}${GREEN}                    ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════╝${NC}"
echo ""
echo -e "  Agent ID:   ${BOLD}${AGENT_ID}${NC}"
echo -e "  Mode:       ${INSTALL_MODE}"
echo -e "  Config:     ${INSTALL_DIR}/config.yaml"
if [[ "$INSTALL_MODE" == "binary" ]]; then
    echo -e "  Logs:       journalctl -u ${SERVICE_NAME} -f"
    echo -e "  Status:     systemctl status ${SERVICE_NAME}"
    echo -e "  Restart:    systemctl restart ${SERVICE_NAME}"
else
    echo -e "  Logs:       docker logs -f ${SERVICE_NAME}"
    echo -e "  Status:     docker ps | grep ${SERVICE_NAME}"
    echo -e "  Restart:    docker restart ${SERVICE_NAME}"
fi
echo -e "  Uninstall:  bash $0 --uninstall"
echo ""
