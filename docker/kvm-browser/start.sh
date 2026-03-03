#!/bin/sh
set -e

# Start virtual framebuffer
Xvfb :99 -screen 0 ${SCREEN_WIDTH}x${SCREEN_HEIGHT}x${SCREEN_DEPTH} -ac +extension GLX +render -noreset &

# Wait for Xvfb to be ready
for i in $(seq 1 10); do
  xdpyinfo -display :99 >/dev/null 2>&1 && break
  sleep 0.3
done

# Start Chromium in kiosk mode with remote debugging for post-login navigation
chromium-browser \
  --no-sandbox \
  --disable-gpu \
  --disable-dev-shm-usage \
  --no-first-run \
  --no-default-browser-check \
  --disable-translate \
  --disable-extensions \
  --disable-sync \
  --disable-features=TranslateUI \
  --ignore-certificate-errors \
  --allow-running-insecure-content \
  --kiosk \
  --remote-debugging-port=9222 \
  --window-size=${SCREEN_WIDTH},${SCREEN_HEIGHT} \
  --window-position=0,0 \
  "${TARGET_URL}" &

if [ -n "$AUTO_USER" ] && [ -n "$REDIRECT_URL" ]; then
  # ── Direct Console mode ──
  # Run CDP auto-login + redirect SYNCHRONOUSLY before starting x11vnc.
  # This ensures VNC is only exposed AFTER the browser has navigated to
  # the vConsole page — the user never sees the BMC login screen.
  # "|| true" prevents script exit on non-zero return (graceful fallback).
  echo "start.sh: auto-login mode — running cdp-redirect.py synchronously"
  python3 /cdp-redirect.py || true
  echo "start.sh: cdp-redirect.py complete, starting VNC server"
elif [ -n "$REDIRECT_URL" ]; then
  # ── Web KVM mode ──
  # Run in background: user logs in manually via the VNC viewer,
  # CDP script monitors and navigates to vConsole after manual login.
  python3 /cdp-redirect.py &
fi

# Start VNC server
# -forever: keep running after clients disconnect
# -shared:  allow multiple concurrent VNC connections
# -nopw:    no password (relay handles auth)
exec x11vnc -display :99 -rfbport ${VNC_PORT} -nopw -listen 0.0.0.0 -xkb -forever -shared
