#!/bin/sh
set -e

# Start virtual framebuffer
Xvfb :99 -screen 0 ${SCREEN_WIDTH}x${SCREEN_HEIGHT}x${SCREEN_DEPTH} -ac +extension GLX +render -noreset &
sleep 1

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
sleep 2

# Start VNC server (no password, listen on all interfaces)
exec x11vnc -display :99 -rfbport ${VNC_PORT} -nopw -listen 0.0.0.0 -xkb -forever -shared -ncache 10
