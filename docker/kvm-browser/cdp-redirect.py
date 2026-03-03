#!/usr/bin/env python3
"""CDP-based auto-login & redirect for KVM vConsole mode.

Monitors the Chromium page via Chrome DevTools Protocol.

When AUTO_USER / AUTO_PASS are set (direct console mode):
  1. Waits for the login page to load
  2. Auto-fills username/password and submits via JavaScript
  3. After login detected, navigates to REDIRECT_URL (vConsole)

When only REDIRECT_URL is set (legacy mode):
  Polls until URL or title changes, then navigates to REDIRECT_URL.

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
    url = url.replace("ws://", "")
    host_port, path = url.split("/", 1)
    host, port = host_port.split(":")
    port = int(port)

    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(10)
    s.connect((host, port))

    key = "dGhlIHNhbXBsZSBub25jZQ=="
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
    frame.append(0x81)

    length = len(payload)
    if length < 126:
        frame.append(0x80 | length)
    elif length < 65536:
        frame.append(0x80 | 126)
        frame.extend(struct.pack(">H", length))
    else:
        frame.append(0x80 | 127)
        frame.extend(struct.pack(">Q", length))

    mask = b"\x00\x00\x00\x00"
    frame.extend(mask)
    frame.extend(payload)
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


_cdp_id = 0


def cdp_call(s, method, params=None):
    """Send a CDP command and return the response."""
    global _cdp_id
    _cdp_id += 1
    msg = {"id": _cdp_id, "method": method}
    if params:
        msg["params"] = params
    ws_send(s, json.dumps(msg))
    # Read responses until we get ours
    deadline = time.time() + 15
    while time.time() < deadline:
        raw = ws_recv(s, timeout=5)
        if raw is None:
            continue
        try:
            resp = json.loads(raw)
            if resp.get("id") == _cdp_id:
                return resp
        except json.JSONDecodeError:
            pass
    return None


def navigate_tab(ws_url, target_url):
    """Use CDP Page.navigate to change URL of existing tab."""
    try:
        s = ws_connect(ws_url)
        resp = cdp_call(s, "Page.navigate", {"url": target_url})
        s.close()
        return resp
    except Exception as e:
        print(f"CDP navigate error: {e}", flush=True)
        return None


def auto_login(ws_url, username, password):
    """Use CDP to auto-fill login form and submit.

    Works with React/Angular SPAs by using native setter to bypass
    framework's synthetic event system.
    """
    # Escape special characters for JS string embedding
    safe_user = username.replace("\\", "\\\\").replace("'", "\\'")
    safe_pass = password.replace("\\", "\\\\").replace("'", "\\'")

    js = """
    (function() {
        function setVal(el, val) {
            var nativeSetter = Object.getOwnPropertyDescriptor(
                window.HTMLInputElement.prototype, 'value').set;
            nativeSetter.call(el, val);
            el.dispatchEvent(new Event('input', {bubbles: true}));
            el.dispatchEvent(new Event('change', {bubbles: true}));
        }

        // Find visible text-like input and password input
        var inputs = document.querySelectorAll('input');
        var userEl = null, passEl = null;
        for (var i = 0; i < inputs.length; i++) {
            var inp = inputs[i];
            var type = (inp.type || '').toLowerCase();
            // Skip hidden inputs
            if (inp.offsetParent === null && inp.offsetWidth === 0) continue;
            if (type === 'hidden') continue;
            if (type === 'password' && !passEl) {
                passEl = inp;
            } else if ((type === 'text' || type === 'email' || type === '') && !userEl) {
                userEl = inp;
            }
        }

        if (!userEl || !passEl) return 'not_found';

        userEl.focus();
        setVal(userEl, '""" + safe_user + """');
        passEl.focus();
        setVal(passEl, '""" + safe_pass + """');

        // Find submit button
        var btn = null;
        var buttons = document.querySelectorAll('button, input[type="submit"]');
        for (var j = 0; j < buttons.length; j++) {
            var b = buttons[j];
            if (b.offsetParent === null && b.offsetWidth === 0) continue;
            var text = (b.textContent || b.value || '').toLowerCase();
            if (text.match(/log\\s*in|sign\\s*in|submit|login/)) {
                btn = b;
                break;
            }
        }
        if (!btn) {
            var form = passEl.closest('form');
            if (form) btn = form.querySelector('button, input[type="submit"]');
        }
        if (btn) {
            btn.click();
            return 'clicked';
        }

        // Fallback: press Enter on password field
        passEl.dispatchEvent(new KeyboardEvent('keydown',
            {key: 'Enter', code: 'Enter', keyCode: 13, bubbles: true}));
        passEl.dispatchEvent(new KeyboardEvent('keypress',
            {key: 'Enter', code: 'Enter', keyCode: 13, bubbles: true}));
        passEl.dispatchEvent(new KeyboardEvent('keyup',
            {key: 'Enter', code: 'Enter', keyCode: 13, bubbles: true}));
        return 'enter_sent';
    })()
    """

    for attempt in range(10):
        try:
            s = ws_connect(ws_url)
            resp = cdp_call(s, "Runtime.evaluate", {"expression": js})
            s.close()

            result_val = ""
            if resp and "result" in resp:
                r = resp["result"].get("result", {})
                result_val = r.get("value", "")

            print(f"  auto-login attempt {attempt + 1}: {result_val}", flush=True)

            if result_val in ("clicked", "enter_sent"):
                return True
            # Form not rendered yet, retry
            time.sleep(1.5)
        except Exception as e:
            print(f"  auto-login attempt {attempt + 1} error: {e}", flush=True)
            time.sleep(1.5)

    return False


def wait_for_page_change(initial_url, initial_title, ws_debug_url_ref):
    """Poll until URL or title changes (login success detected)."""
    deadline = time.time() + TIMEOUT
    while time.time() < deadline:
        time.sleep(POLL_INTERVAL)
        targets = cdp_get_targets()
        pages = [t for t in targets if t.get("type") == "page"]
        if not pages:
            continue

        current_url = pages[0].get("url", "")
        current_title = pages[0].get("title", "")
        ws_debug_url_ref[0] = pages[0].get("webSocketDebuggerUrl", ws_debug_url_ref[0])

        url_changed = current_url and current_url != initial_url
        title_changed = (
            current_title
            and initial_title
            and current_title != initial_title
            and current_title not in ("", "about:blank")
        )

        if url_changed or title_changed:
            reason = "URL" if url_changed else "title"
            print(f"CDP-redirect: Login detected ({reason} changed)", flush=True)
            print(f"  URL:   {current_url}", flush=True)
            print(f"  Title: {current_title}", flush=True)
            return True

    return False


def main():
    redirect_url = os.environ.get("REDIRECT_URL", "").strip()
    auto_user = os.environ.get("AUTO_USER", "").strip()
    auto_pass = os.environ.get("AUTO_PASS", "").strip()

    if not redirect_url:
        print("CDP-redirect: No REDIRECT_URL set, exiting.", flush=True)
        return

    print(f"CDP-redirect: Will redirect to {redirect_url}", flush=True)
    if auto_user:
        print(f"CDP-redirect: Auto-login enabled (user={auto_user})", flush=True)

    # Wait for Chromium CDP to be available
    initial_url = None
    initial_title = None
    ws_debug_url = None

    for attempt in range(40):
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

    ws_ref = [ws_debug_url]

    # Auto-login if credentials provided
    if auto_user:
        # Wait a bit for the login form to render (SPAs need time)
        time.sleep(2)
        print("CDP-redirect: Attempting auto-login...", flush=True)
        success = auto_login(ws_debug_url, auto_user, auto_pass)
        if success:
            print("CDP-redirect: Auto-login submitted, waiting for page change...", flush=True)
        else:
            print("CDP-redirect: Auto-login failed (form not found), waiting for manual login...", flush=True)

    # Wait for login (URL or title change)
    if wait_for_page_change(initial_url, initial_title, ws_ref):
        # Brief pause to let post-login JS settle
        time.sleep(1.5)

        # Navigate existing tab to console URL (preserves session)
        print(f"CDP-redirect: Navigating to {redirect_url}", flush=True)
        result = navigate_tab(ws_ref[0], redirect_url)
        print(f"CDP-redirect: Navigate result: {result}", flush=True)
    else:
        print("CDP-redirect: Timeout — no login detected.", flush=True)


if __name__ == "__main__":
    main()
