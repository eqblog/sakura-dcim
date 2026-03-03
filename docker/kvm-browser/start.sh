#!/bin/sh
set -e

# Start virtual framebuffer
Xvfb :99 -screen 0 ${SCREEN_WIDTH}x${SCREEN_HEIGHT}x${SCREEN_DEPTH} -ac +extension GLX +render -noreset &

# Wait for Xvfb to be ready
for i in $(seq 1 10); do
  xdpyinfo -display :99 >/dev/null 2>&1 && break
  sleep 0.5
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

# Auto-redirect to REDIRECT_URL after login (for direct vConsole mode).
# Polls CDP until the page URL changes from the login page (indicating
# successful authentication), then opens the console URL in a new tab
# and closes the old one so the session cookie is preserved.
if [ -n "$REDIRECT_URL" ]; then
  (
    # Wait for Chromium and CDP to become available
    sleep 6
    INITIAL_URL=""
    for i in $(seq 1 15); do
      INITIAL_URL=$(curl -s http://127.0.0.1:9222/json 2>/dev/null \
        | grep '"url"' | head -1 | sed 's/.*"url": *"//;s/".*//')
      [ -n "$INITIAL_URL" ] && break
      sleep 1
    done

    if [ -n "$INITIAL_URL" ]; then
      # Poll until URL changes (user authenticated) — up to 5 minutes
      ATTEMPTS=0
      while [ $ATTEMPTS -lt 150 ]; do
        sleep 2
        CURRENT=$(curl -s http://127.0.0.1:9222/json 2>/dev/null \
          | grep '"url"' | head -1 | sed 's/.*"url": *"//;s/".*//')
        if [ -n "$CURRENT" ] && [ "$CURRENT" != "$INITIAL_URL" ]; then
          # Login detected — navigate to console page
          sleep 2
          OLD_ID=$(curl -s http://127.0.0.1:9222/json 2>/dev/null \
            | grep '"id"' | head -1 | sed 's/.*"id": *"//;s/".*//')
          ENCODED=$(printf '%s' "$REDIRECT_URL" | sed 's/#/%23/g; s/ /%20/g')
          curl -s -X PUT "http://127.0.0.1:9222/json/new?${ENCODED}" >/dev/null 2>&1
          sleep 1
          [ -n "$OLD_ID" ] && \
            curl -s -X PUT "http://127.0.0.1:9222/json/close/${OLD_ID}" >/dev/null 2>&1
          break
        fi
        ATTEMPTS=$((ATTEMPTS + 1))
      done
    fi
  ) &
fi

# Give Chromium a moment to start rendering before x11vnc captures screen
sleep 2

# Start VNC server
# -forever: keep running after clients disconnect
# -shared:  allow multiple concurrent VNC connections
# -nopw:    no password (relay handles auth)
exec x11vnc -display :99 -rfbport ${VNC_PORT} -nopw -listen 0.0.0.0 -xkb -forever -shared
