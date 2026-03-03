FROM alpine:3.21

RUN apk add --no-cache \
    xvfb \
    x11vnc \
    xdpyinfo \
    chromium \
    font-noto \
    font-noto-cjk \
    dbus \
    ttf-freefont \
    mesa-dri-gallium \
    curl \
    && rm -rf /var/cache/apk/*

RUN adduser -D -u 1000 -G kvm kvmuser

ENV DISPLAY=:99 \
    SCREEN_WIDTH=1280 \
    SCREEN_HEIGHT=1024 \
    SCREEN_DEPTH=24 \
    VNC_PORT=5900 \
    TARGET_URL="" \
    REDIRECT_URL=""

COPY start.sh /start.sh
RUN chmod +x /start.sh

USER kvmuser
EXPOSE 5900

ENTRYPOINT ["/start.sh"]
