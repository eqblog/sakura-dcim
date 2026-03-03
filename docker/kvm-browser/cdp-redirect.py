#!/usr/bin/env python3
"""CDP-based auto-login & redirect for KVM vConsole mode.

Monitors the Chromium page via Chrome DevTools Protocol.

When AUTO_USER / AUTO_PASS are set (direct console mode):
  1. Waits for login page to fully load & settle (handles redirects/SPA routing)
  2. Auto-fills username/password and submits via JavaScript
  3. Waits for URL to change (login redirect detected), then navigates to REDIRECT_URL

When only REDIRECT_URL is set (web KVM manual-login mode):
  Waits for URL to settle, polls until URL/title changes, then navigates.

Uses only Python stdlib (raw WebSocket via socket).
"""

import json
import os
import socket
import struct
import time
import urllib.request

CDP_HOST = "127.0.0.1"
CDP_PORT = 9222
POLL_INTERVAL = 1.5  # seconds
TIMEOUT = 300  # 5 minutes
POST_LOGIN_SETTLE = 10  # seconds to wait after login redirect detected


# ── CDP HTTP helpers ──

def cdp_get_targets():
    """Fetch target list from CDP HTTP endpoint."""
    try:
        with urllib.request.urlopen(f"http://{CDP_HOST}:{CDP_PORT}/json", timeout=3) as r:
            return json.loads(r.read())
    except Exception:
        return []


def get_page_info():
    """Get first page target from CDP."""
    targets = cdp_get_targets()
    pages = [t for t in targets if t.get("type") == "page"]
    return pages[0] if pages else None


# ── Minimal WebSocket client (RFC 6455) ──

def ws_connect(url):
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
        if opcode == 0x1:
            return data.decode("utf-8", errors="replace")
        return None
    except (socket.timeout, OSError):
        return None


_cdp_id = 0


def cdp_call(s, method, params=None):
    global _cdp_id
    _cdp_id += 1
    msg = {"id": _cdp_id, "method": method}
    if params:
        msg["params"] = params
    ws_send(s, json.dumps(msg))
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


# ── High-level helpers ──

def navigate_tab(ws_url, target_url):
    """Use CDP Page.navigate to change URL of existing tab (preserves cookies)."""
    try:
        s = ws_connect(ws_url)
        resp = cdp_call(s, "Page.navigate", {"url": target_url})
        s.close()
        return resp
    except Exception as e:
        print(f"CDP navigate error: {e}", flush=True)
        return None


def navigate_current_page(target_url):
    """Navigate the current tab to target_url using fresh page info."""
    page = get_page_info()
    if not page:
        print("CDP-redirect: No page found for navigation", flush=True)
        return None
    ws_url = page.get("webSocketDebuggerUrl", "")
    if not ws_url:
        print("CDP-redirect: No WebSocket debug URL", flush=True)
        return None
    print(f"CDP-redirect: Navigating to {target_url}", flush=True)
    result = navigate_tab(ws_url, target_url)
    print(f"CDP-redirect: Navigate result: {result}", flush=True)
    return result


def wait_for_url_stable(timeout=20):
    """Wait until the page URL stops changing for 3 consecutive seconds.

    Handles initial redirects and SPA routing that change the URL after load.
    Returns the final stable page info dict, or None on timeout.
    """
    last_url = ""
    stable_since = time.time()
    deadline = time.time() + timeout

    while time.time() < deadline:
        time.sleep(1)
        page = get_page_info()
        if not page:
            continue
        current_url = page.get("url", "")
        if not current_url or "chrome" in current_url:
            continue

        if current_url == last_url:
            if time.time() - stable_since >= 3:
                return page
        else:
            last_url = current_url
            stable_since = time.time()

    return get_page_info()


def wait_for_ready_state(timeout=30):
    """Wait until document.readyState == 'complete' on the current page.

    Called after Page.navigate to ensure the vConsole page is fully
    loaded before we let x11vnc start — so the user never sees a
    half-loaded page or a blank screen.
    """
    deadline = time.time() + timeout
    while time.time() < deadline:
        page = get_page_info()
        if page:
            ws_url = page.get("webSocketDebuggerUrl", "")
            if ws_url:
                try:
                    s = ws_connect(ws_url)
                    resp = cdp_call(s, "Runtime.evaluate",
                                    {"expression": "document.readyState"})
                    s.close()
                    val = ((resp or {}).get("result", {})
                                       .get("result", {})
                                       .get("value", ""))
                    if val == "complete":
                        return True
                except Exception:
                    pass
        time.sleep(1)
    return False


def try_launch_vconsole(ws_url, fallback_url):
    """Capture the virtual console URL by intercepting the BMC's launch button.

    Intercepts window.open() called by the 'Virtual Console' button on the
    BMC dashboard.  The URL passed to window.open() often includes a one-time
    session token required for the KVM WebSocket to authenticate — so we must
    use that URL instead of navigating to the vconsole page directly.

    If no button is found, or window.open() is never called (e.g. the SPA
    navigates internally), returns fallback_url so the caller can fall back.
    """
    js = r"""
    (function() {
        var captured = null;
        var origOpen = window.open;
        window.open = function(url, name, features) {
            if (url && !captured) captured = String(url);
            return {
                focus: function() {}, closed: false,
                location: {href: url || ''},
                document: {write: function() {}, close: function() {}}
            };
        };

        var found = null;

        // 1. Vendor-specific / id-based selectors
        var sels = [
            '#launch-virtual-console', '#virtualConsole', '#btnVirtualConsole',
            '#html5console', '#kvm-launch',
            '[data-automation*="onsole"]', '[data-testid*="onsole"]',
            '[data-id*="onsole"]', '[id*="Console"]',
            'a[href*="vconsole"]', 'a[href*="virtualconsole"]',
            '[onclick*="console"]', '[ng-click*="console"]',
        ];
        for (var i = 0; i < sels.length && !found; i++) {
            var el = document.querySelector(sels[i]);
            if (el && el.offsetWidth > 0) { el.click(); found = 'sel:' + sels[i]; }
        }

        // 2. Generic text search
        if (!found) {
            var pattern = /\b(virtual\s*console|launch\s*console|kvm\s*console|html5\s*console|remote\s*console)\b/i;
            var nodes = document.querySelectorAll(
                'button, a, [role="button"], [role="menuitem"], li, span');
            for (var j = 0; j < nodes.length && !found; j++) {
                var b = nodes[j];
                if (b.offsetWidth === 0 && b.offsetHeight === 0) continue;
                var txt = (b.textContent || b.title ||
                           b.getAttribute('aria-label') || '').trim();
                if (txt.length > 0 && txt.length < 60 && pattern.test(txt)) {
                    b.click();
                    found = 'txt:' + txt.substring(0, 40);
                }
            }
        }

        window.open = origOpen;
        return JSON.stringify({url: captured, clicked: found});
    })()
    """
    try:
        s = ws_connect(ws_url)
        resp = cdp_call(s, "Runtime.evaluate", {"expression": js})
        s.close()
        raw = (resp or {}).get("result", {}).get("result", {}).get("value", "")
        if raw:
            d = json.loads(raw)
            print(f"CDP-redirect: vconsole button → {d}", flush=True)
            url = d.get("url", "")
            if url and url.startswith("http"):
                return url
            if d.get("clicked"):
                # Button found and clicked, but no window.open() observed.
                # The SPA may have navigated internally — check current URL.
                time.sleep(2)
                page = get_page_info()
                if page:
                    cur = page.get("url", "")
                    if cur and "vconsole" in cur.lower():
                        return cur
    except Exception as e:
        print(f"CDP-redirect: try_launch_vconsole error: {e}", flush=True)
    return fallback_url


def auto_login(ws_url, username, password):
    """Auto-fill login form and submit via CDP Runtime.evaluate.

    Uses vendor-specific selectors (iDRAC, iLO, Supermicro, etc.) with
    generic fallback. React/Angular compatibility via native setter.
    """
    safe_user = username.replace("\\", "\\\\").replace("'", "\\'")
    safe_pass = password.replace("\\", "\\\\").replace("'", "\\'")

    js = """
    (function() {
        function setVal(el, val) {
            // Use native setter to bypass React/Angular controlled inputs
            var desc = Object.getOwnPropertyDescriptor(
                window.HTMLInputElement.prototype, 'value');
            if (desc && desc.set) {
                desc.set.call(el, val);
            } else {
                el.value = val;
            }
            el.dispatchEvent(new Event('input', {bubbles: true}));
            el.dispatchEvent(new Event('change', {bubbles: true}));
            // AngularJS compatibility
            if (typeof angular !== 'undefined') {
                try { angular.element(el).triggerHandler('input'); } catch(e) {}
            }
        }

        // ── Find username field: vendor-specific first, then generic ──
        var userEl = document.querySelector('#user')
                  || document.querySelector('#iDRAC_Username')
                  || document.querySelector('input[name="user"]')
                  || document.querySelector('input[name="username"]')
                  || document.querySelector('#username')
                  || document.querySelector('#login-user-name')
                  || document.querySelector('input[name="Login"]');

        // ── Find password field ──
        var passEl = document.querySelector('#password')
                  || document.querySelector('#iDRAC_Password')
                  || document.querySelector('input[name="password"]')
                  || document.querySelector('#pwd')
                  || document.querySelector('#login-password')
                  || document.querySelector('input[name="Password"]');

        // Generic fallback: find visible text + password inputs
        if (!userEl || !passEl) {
            var inputs = document.querySelectorAll('input');
            for (var i = 0; i < inputs.length; i++) {
                var inp = inputs[i];
                var type = (inp.type || '').toLowerCase();
                if (inp.offsetParent === null && inp.offsetWidth === 0) continue;
                if (type === 'hidden') continue;
                if (type === 'password' && !passEl) { passEl = inp; }
                else if ((type === 'text' || type === 'email' || type === '') && !userEl) {
                    userEl = inp;
                }
            }
        }

        if (!userEl || !passEl) return 'not_found';

        userEl.focus();
        setVal(userEl, '""" + safe_user + """');
        passEl.focus();
        setVal(passEl, '""" + safe_pass + """');

        // ── Find submit button: vendor-specific first ──
        var btn = document.querySelector('#btnOK')
               || document.querySelector('#login-btn')
               || document.querySelector('#btnLogin')
               || document.querySelector('button[type="submit"]')
               || document.querySelector('input[type="submit"]');

        // Generic: find button by visible text
        if (!btn) {
            var buttons = document.querySelectorAll(
                'button, input[type="submit"], input[type="button"], a.btn');
            for (var j = 0; j < buttons.length; j++) {
                var b = buttons[j];
                if (b.offsetParent === null && b.offsetWidth === 0) continue;
                var text = (b.textContent || b.value || '').trim().toLowerCase();
                if (/^(log\\s*in|sign\\s*in|submit|login|登录)$/.test(text)) {
                    btn = b;
                    break;
                }
            }
        }
        // Last resort: first button/submit inside the form
        if (!btn) {
            var form = passEl.closest('form');
            if (form) btn = form.querySelector('button, input[type="submit"]');
        }

        if (btn) {
            btn.click();
            return 'clicked';
        }

        // Absolute fallback: submit the form or press Enter
        var form = passEl.closest('form');
        if (form) { form.submit(); return 'form_submitted'; }

        passEl.dispatchEvent(new KeyboardEvent('keydown',
            {key: 'Enter', code: 'Enter', keyCode: 13, bubbles: true}));
        passEl.dispatchEvent(new KeyboardEvent('keypress',
            {key: 'Enter', code: 'Enter', keyCode: 13, bubbles: true}));
        passEl.dispatchEvent(new KeyboardEvent('keyup',
            {key: 'Enter', code: 'Enter', keyCode: 13, bubbles: true}));
        return 'enter_sent';
    })()
    """

    for attempt in range(12):
        try:
            s = ws_connect(ws_url)
            resp = cdp_call(s, "Runtime.evaluate", {"expression": js})
            s.close()

            result_val = ""
            if resp and "result" in resp:
                r = resp["result"].get("result", {})
                result_val = r.get("value", "")

            print(f"  auto-login attempt {attempt + 1}: {result_val}", flush=True)

            if result_val in ("clicked", "enter_sent", "form_submitted"):
                return True
            # Form not rendered yet, retry
            time.sleep(2)
        except Exception as e:
            print(f"  auto-login attempt {attempt + 1} error: {e}", flush=True)
            time.sleep(2)

    return False


def wait_for_page_change(baseline_url, baseline_title, ws_debug_url_ref, timeout=None):
    """Poll until URL or title changes from the baseline (login detected)."""
    if timeout is None:
        timeout = TIMEOUT
    deadline = time.time() + timeout
    while time.time() < deadline:
        time.sleep(POLL_INTERVAL)
        page = get_page_info()
        if not page:
            continue

        current_url = page.get("url", "")
        current_title = page.get("title", "")
        ws_debug_url_ref[0] = page.get("webSocketDebuggerUrl", ws_debug_url_ref[0])

        url_changed = current_url and current_url != baseline_url
        title_changed = (
            current_title
            and baseline_title
            and current_title != baseline_title
            and current_title not in ("", "about:blank")
        )

        if url_changed or title_changed:
            reason = "URL" if url_changed else "title"
            print(f"CDP-redirect: Login detected ({reason} changed)", flush=True)
            print(f"  from: {baseline_url}  /  {baseline_title}", flush=True)
            print(f"  to:   {current_url}  /  {current_title}", flush=True)
            return True

    return False


def wait_for_post_login(baseline_url, baseline_title, ws_debug_url_ref, timeout=90):
    """After clicking login, wait for BMC to redirect away from the login page.

    Uses a shorter timeout than the main TIMEOUT since we know login was
    submitted. Falls back gracefully: if no redirect is seen within `timeout`
    seconds, the caller should navigate anyway in case cookies were set silently.
    """
    deadline = time.time() + timeout
    while time.time() < deadline:
        time.sleep(POLL_INTERVAL)
        page = get_page_info()
        if not page:
            continue

        current_url = page.get("url", "")
        current_title = page.get("title", "")
        ws_debug_url_ref[0] = page.get("webSocketDebuggerUrl", ws_debug_url_ref[0])

        url_changed = current_url and current_url != baseline_url
        title_changed = (
            current_title
            and baseline_title
            and current_title != baseline_title
            and current_title not in ("", "about:blank")
        )

        if url_changed or title_changed:
            reason = "URL" if url_changed else "title"
            print(f"CDP-redirect: Post-login redirect detected ({reason})", flush=True)
            print(f"  from: {baseline_url}", flush=True)
            print(f"  to:   {current_url}", flush=True)
            return True

    print(f"CDP-redirect: No post-login redirect in {timeout}s "
          f"(BMC may be slow or login silent)", flush=True)
    return False


# ── Main ──

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

    # 1. Wait for Chromium CDP to become available
    for attempt in range(40):
        time.sleep(1)
        page = get_page_info()
        if page:
            url = page.get("url", "")
            if url and "chrome" not in url:
                break

    # 2. Wait for page URL to SETTLE (handles redirects + SPA routing).
    #    This is critical: iDRAC redirects restgui/start.html → login.html,
    #    React SPAs change hash to #/login, etc.  We must wait for all of
    #    that to finish BEFORE capturing the baseline or attempting login.
    print("CDP-redirect: Waiting for page URL to settle...", flush=True)
    page = wait_for_url_stable(timeout=20)
    if not page:
        print("CDP-redirect: Could not get stable page info, aborting.", flush=True)
        return

    settled_url = page.get("url", "")
    settled_title = page.get("title", "")
    ws_debug_url = page.get("webSocketDebuggerUrl", "")

    print(f"CDP-redirect: Settled URL   = {settled_url}", flush=True)
    print(f"CDP-redirect: Settled title = {settled_title}", flush=True)

    if auto_user and ws_debug_url:
        # ── Auto-login mode ──
        # The page URL is now stable (login form should be rendered).
        print("CDP-redirect: Attempting auto-login...", flush=True)
        success = auto_login(ws_debug_url, auto_user, auto_pass)

        if success:
            # ── Dynamic wait: detect when BMC redirects away from login page ──
            # Replaces the old fixed LOGIN_SETTLE_SECS=8 delay which was too
            # short for BMCs that take >8s to authenticate and redirect.
            ws_ref = [ws_debug_url]
            print("CDP-redirect: Login submitted — waiting for post-login redirect...", flush=True)
            wait_for_post_login(settled_url, settled_title, ws_ref, timeout=90)

            # Settle pause: give the BMC session time to fully initialise.
            # During this window the dashboard SPA finishes loading its state
            # (including any KVM session tokens) before we interact with it.
            time.sleep(POST_LOGIN_SETTLE)
            fresh = get_page_info()
            nav_ws = fresh.get("webSocketDebuggerUrl", ws_ref[0]) if fresh else ws_ref[0]

            # Wait for the dashboard page to be fully rendered so the
            # Virtual Console button is clickable.
            print("CDP-redirect: Waiting for dashboard to render...", flush=True)
            wait_for_ready_state(timeout=15)
            time.sleep(1)

            # ── Key fix: capture the real console URL via button interception ──
            # iDRAC (and other BMCs) call window.open(url) where url includes
            # a one-time session token.  Directly navigating to vconsole/index.html
            # skips this token → "Connecting Viewer..." hangs indefinitely.
            # try_launch_vconsole() intercepts that window.open call to get the
            # token-bearing URL.  Falls back to redirect_url if button not found.
            fresh = get_page_info()
            nav_ws = fresh.get("webSocketDebuggerUrl", nav_ws) if fresh else nav_ws
            print("CDP-redirect: Intercepting Virtual Console button...", flush=True)
            console_url = try_launch_vconsole(nav_ws, redirect_url)
            print(f"CDP-redirect: Navigating to: {console_url}", flush=True)

            result = navigate_tab(nav_ws, console_url)
            print(f"CDP-redirect: Navigate result: {result}", flush=True)

            # Wait for the vConsole page to fully load, then give the KVM
            # viewer time to establish its WebSocket video session.
            print("CDP-redirect: Waiting for vConsole page to load...", flush=True)
            time.sleep(2)  # let Page.navigate complete
            loaded = wait_for_ready_state(timeout=20)
            print(f"CDP-redirect: vConsole ready state: {'complete' if loaded else 'timeout'}",
                  flush=True)
            # Extra settle for KVM WebSocket to initialise video session
            print("CDP-redirect: Waiting for KVM viewer to initialise...", flush=True)
            time.sleep(5)
        else:
            # Form not found — fall back to watching for manual login
            print("CDP-redirect: Auto-login failed, watching for manual login...",
                  flush=True)
            ws_ref = [ws_debug_url]
            if wait_for_page_change(settled_url, settled_title, ws_ref):
                time.sleep(1.5)
                navigate_tab(ws_ref[0], redirect_url)
            else:
                print("CDP-redirect: Timeout — no login detected.", flush=True)
    else:
        # ── Manual login mode (web KVM: user logs in via VNC) ──
        ws_ref = [ws_debug_url]
        if wait_for_page_change(settled_url, settled_title, ws_ref):
            time.sleep(1.5)
            print(f"CDP-redirect: Navigating to {redirect_url}", flush=True)
            result = navigate_tab(ws_ref[0], redirect_url)
            print(f"CDP-redirect: Navigate result: {result}", flush=True)
        else:
            print("CDP-redirect: Timeout — no login detected.", flush=True)


if __name__ == "__main__":
    main()
