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
  --remote-debugging-port=9222 \
  --window-size=${SCREEN_WIDTH},${SCREEN_HEIGHT} \
  --window-position=0,0 \
  "${TARGET_URL}" &

# Launch CDP redirect script in background (handles vConsole auto-navigation)
if [ -n "$REDIRECT_URL" ]; then
  python3 /cdp-redirect.py &
fi

# Give Chromium a moment to start rendering before x11vnc captures screen
sleep 1

# Start VNC server
# -forever: keep running after clients disconnect
# -shared:  allow multiple concurrent VNC connections
# -nopw:    no password (relay handles auth)
exec x11vnc -display :99 -rfbport ${VNC_PORT} -nopw -listen 0.0.0.0 -xkb -forever -shared
