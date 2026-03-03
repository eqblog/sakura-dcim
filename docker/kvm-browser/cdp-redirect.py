#!/usr/bin/env python3
"""CDP-based redirect for KVM vConsole mode.

Monitors the Chromium page via Chrome DevTools Protocol.
Once the page URL or title changes (indicating login), navigates the
SAME tab to the target console URL using Page.navigate — this preserves
the session cookies set during authentication.

Uses only Python stdlib (raw WebSocket via socket).
"""

import json
import os
import socket
import struct
import sys
import time
import urllib.request

CDP_HOST = "127.0.0.1"
CDP_PORT = 9222
POLL_INTERVAL = 1.5  # seconds
TIMEOUT = 300  # 5 minutes


def cdp_get_targets():
    """Fetch target list from CDP HTTP endpoint."""
    try:
        with urllib.request.urlopen(f"http://{CDP_HOST}:{CDP_PORT}/json", timeout=3) as r:
            return json.loads(r.read())
    except Exception:
        return []


def ws_connect(url):
    """Minimal WebSocket client connect (RFC 6455)."""
    # Parse ws://host:port/path
    url = url.replace("ws://", "")
    host_port, path = url.split("/", 1)
    host, port = host_port.split(":")
    port = int(port)

    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(10)
    s.connect((host, port))

    # WebSocket handshake
    key = "dGhlIHNhbXBsZSBub25jZQ=="  # static key, fine for local CDP
    handshake = (
        f"GET /{path} HTTP/1.1\r\n"
        f"Host: {host}:{port}\r\n"
        f"Upgrade: websocket\r\n"
        f"Connection: Upgrade\r\n"
        f"Sec-WebSocket-Key: {key}\r\n"
        f"Sec-WebSocket-Version: 13\r\n"
        f"\r\n"
    )
    s.sendall(handshake.encode())

    # Read response headers
    response = b""
    while b"\r\n\r\n" not in response:
        chunk = s.recv(4096)
        if not chunk:
            raise ConnectionError("WebSocket handshake failed")
        response += chunk

    if b"101" not in response.split(b"\r\n")[0]:
        raise ConnectionError(f"WebSocket handshake rejected: {response[:200]}")

    return s


def ws_send(s, message):
    """Send a text frame (masked) over WebSocket."""
    payload = message.encode("utf-8")
    frame = bytearray()
    frame.append(0x81)  # FIN + text opcode

    length = len(payload)
    if length < 126:
        frame.append(0x80 | length)  # MASK bit set
    elif length < 65536:
        frame.append(0x80 | 126)
        frame.extend(struct.pack(">H", length))
    else:
        frame.append(0x80 | 127)
        frame.extend(struct.pack(">Q", length))

    # Mask key (all zeros for simplicity — local only)
    mask = b"\x00\x00\x00\x00"
    frame.extend(mask)
    frame.extend(payload)  # XOR with zero mask = payload unchanged
    s.sendall(bytes(frame))


def ws_recv(s, timeout=5):
    """Receive a WebSocket text frame."""
    s.settimeout(timeout)
    try:
        header = s.recv(2)
        if len(header) < 2:
            return None
        opcode = header[0] & 0x0F
        masked = (header[1] & 0x80) != 0
        length = header[1] & 0x7F

        if length == 126:
            length = struct.unpack(">H", s.recv(2))[0]
        elif length == 127:
            length = struct.unpack(">Q", s.recv(8))[0]

        if masked:
            mask_key = s.recv(4)

        data = b""
        while len(data) < length:
            chunk = s.recv(length - len(data))
            if not chunk:
                break
            data += chunk

        if masked:
            data = bytes(b ^ mask_key[i % 4] for i, b in enumerate(data))

        if opcode == 0x1:  # text
            return data.decode("utf-8", errors="replace")
        return None
    except (socket.timeout, OSError):
        return None


def navigate_tab(ws_url, target_url):
    """Use CDP Page.navigate to change URL of existing tab."""
    try:
        s = ws_connect(ws_url)
        cmd = json.dumps({
            "id": 1,
            "method": "Page.navigate",
            "params": {"url": target_url}
        })
        ws_send(s, cmd)
        resp = ws_recv(s, timeout=10)
        s.close()
        return resp
    except Exception as e:
        print(f"CDP navigate error: {e}", flush=True)
        return None


def main():
    redirect_url = os.environ.get("REDIRECT_URL", "").strip()
    if not redirect_url:
        print("CDP-redirect: No REDIRECT_URL set, exiting.", flush=True)
        return

    print(f"CDP-redirect: Will redirect to {redirect_url}", flush=True)

    # Wait for Chromium CDP to be available
    initial_url = None
    initial_title = None
    ws_debug_url = None

    for attempt in range(40):  # up to ~40 seconds
        time.sleep(1)
        targets = cdp_get_targets()
        pages = [t for t in targets if t.get("type") == "page"]
        if pages:
            initial_url = pages[0].get("url", "")
            initial_title = pages[0].get("title", "")
            ws_debug_url = pages[0].get("webSocketDebuggerUrl", "")
            if initial_url and "chrome" not in initial_url:
                break

    if not initial_url or not ws_debug_url:
        print("CDP-redirect: Could not get initial page info, aborting.", flush=True)
        return

    print(f"CDP-redirect: Initial URL={initial_url}", flush=True)
    print(f"CDP-redirect: Initial title={initial_title}", flush=True)

    # Poll until URL or title changes (login detected)
    deadline = time.time() + TIMEOUT
    while time.time() < deadline:
        time.sleep(POLL_INTERVAL)
        targets = cdp_get_targets()
        pages = [t for t in targets if t.get("type") == "page"]
        if not pages:
            continue

        current_url = pages[0].get("url", "")
        current_title = pages[0].get("title", "")
        ws_debug_url = pages[0].get("webSocketDebuggerUrl", ws_debug_url)

        url_changed = current_url and current_url != initial_url
        title_changed = (
            current_title
            and initial_title
            and current_title != initial_title
            # Ignore minor loading states
            and current_title not in ("", "about:blank")
        )

        if url_changed or title_changed:
            reason = "URL" if url_changed else "title"
            print(f"CDP-redirect: Login detected ({reason} changed)", flush=True)
            print(f"  URL:   {current_url}", flush=True)
            print(f"  Title: {current_title}", flush=True)

            # Brief pause to let post-login JS settle
            time.sleep(1.5)

            # Navigate existing tab to console URL (preserves session)
            print(f"CDP-redirect: Navigating to {redirect_url}", flush=True)
            result = navigate_tab(ws_debug_url, redirect_url)
            print(f"CDP-redirect: Navigate result: {result}", flush=True)
            return

    print("CDP-redirect: Timeout — no login detected.", flush=True)


if __name__ == "__main__":
    main()
