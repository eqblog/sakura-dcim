#!/bin/sh
set -e

# Start virtual framebuffer
Xvfb :99 -screen 0 ${SCREEN_WIDTH}x${SCREEN_HEIGHT}x${SCREEN_DEPTH} -ac +extension GLX +render -noreset &

# Wait for Xvfb to be ready
for i in $(seq 1 10); do
  xdpyinfo -display :99 >/dev/null 2>&1 && break
  sleep 0.5
done

# Start Chromium in kiosk mode
chromium-browser \
  --no-sandbox \
  --disable-gpu \
  --disable-software-rasterizer \
  --disable-dev-shm-usage \
  --no-first-run \
  --no-default-browser-check \
  --disable-translate \
  --disable-extensions \
  --disable-background-networking \
  --disable-sync \
  --disable-features=TranslateUI \
  --ignore-certificate-errors \
  --kiosk \
  --window-size=${SCREEN_WIDTH},${SCREEN_HEIGHT} \
  --window-position=0,0 \
  "${TARGET_URL}" &

# Give Chromium a moment to start rendering before x11vnc captures screen
sleep 2

# Start VNC server
# -forever: keep running after clients disconnect
# -shared:  allow multiple concurrent VNC connections
# -nopw:    no password (relay handles auth)
# NOTE: -ncache removed to reduce memory (saves ~80MB with 1280x1024)
exec x11vnc -display :99 -rfbport ${VNC_PORT} -nopw -listen 0.0.0.0 -xkb -forever -shared
