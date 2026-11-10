"""Exercise the built executable using disposable data and localhost only."""

import html
import http.cookiejar
import json
import os
import queue
import re
import subprocess
import sys
import tempfile
import threading
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

EXECUTABLE = str(Path(sys.argv[1]).resolve())


def multipart(fields, file=None):
    boundary = "logflow-smoke-boundary"
    parts = []
    for name, value in fields.items():
        parts.append(
            f'--{boundary}\r\nContent-Disposition: form-data; name="{name}"'
            f'\r\n\r\n{value}\r\n'.encode()
        )
    if file is not None:
        parts.append(
            f'--{boundary}\r\nContent-Disposition: form-data; name="logs"; '
            'filename="synthetic.jsonl"\r\nContent-Type: application/octet-stream'
            f'\r\n\r\n{file}\r\n'.encode()
        )
    parts.append(f"--{boundary}--\r\n".encode())
    return b"".join(parts), "multipart/form-data; boundary=" + boundary


def launch(env):
    process = subprocess.Popen(
        [EXECUTABLE], env=env, stdin=subprocess.DEVNULL,
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
        text=True, encoding="utf-8",
    )
    lines = queue.Queue()
    threading.Thread(target=lambda: lines.put(process.stdout.readline()), daemon=True).start()
    try:
        line = lines.get(timeout=30)
        match = re.search(r"http://\S+", line)
        if not match:
            raise RuntimeError("launcher failed to provide a URL")
        return process, match.group(0)
    except BaseException:
        process.kill()
        process.wait()
        raise


def stop(process):
    process.terminate()
    try:
        process.wait(timeout=15)
    except subprocess.TimeoutExpired:
        process.kill()
        process.wait()
    process.stdout.close()


def check_cli(env):
    replay = subprocess.run([EXECUTABLE, "-replay"], capture_output=True, text=True,
                            env=env, timeout=30, check=False)
    assert replay.returncode == 0
    scenarios = json.loads(replay.stdout)
    assert [r["ObservedAlerts"] for r in scenarios] == [1, 0, 0, 1]
    assert all(r["Complete"] and r["MatchesExpectation"] for r in scenarios)

    def event(second, message="Failed login attempt"):
        return json.dumps({
            "timestamp": f"2026-11-06T12:{second // 60:02d}:{second % 60:02d}Z",
            "level": "ERROR", "message": message, "source_ip": "192.0.2.10",
        }) + "\n"

    cases = [
        ("inclusive", "".join(event(t) for t in [0, 120, 240, 360, 480, 600]), 0, 1),
        ("below threshold", "".join(event(t) for t in range(5)), 0, 0),
        ("expired", "".join(event(t) for t in [0, 120, 240, 360, 480, 601]), 0, 0),
        ("generic errors", "".join(event(t, "database unavailable") for t in range(6)), 0, 0),
        ("bad JSON", "not-json\n", 2, 0),
        ("out of order", event(2) + event(1), 2, 0),
        ("line limit", "x" * (2 << 20), 2, 0),
    ]
    for name, source, code, alerts in cases:
        result = subprocess.run(
            [EXECUTABLE, "-"], input=source, capture_output=True,
            text=True, env=env, timeout=30, check=False,
        )
        report = json.loads(result.stdout)
        assert result.returncode == code, name
        assert report["Complete"] == (code == 0), name
        assert len(report["Alerts"]) == alerts, name
        assert report["Accepted"] + report["Rejected"] == report["Lines"], name
        if alerts:
            assert len(report["Alerts"][0]["Evidence"]) == 6, name
    result = subprocess.run(
        [EXECUTABLE, "-workers", "4"], stdin=subprocess.DEVNULL,
        capture_output=True, env=env, timeout=15, check=False,
    )
    assert result.returncode == 2 and not result.stdout, "implicit stdin accepted"


with tempfile.TemporaryDirectory(prefix="logflow-", dir=os.environ.get("LOGFLOW_SMOKE_TEMP")) as root:
    env = dict(os.environ, LOGFLOW_DATA_DIR=root, LOGFLOW_NO_BROWSER="1")
    check_cli(env)
    browser = urllib.request.build_opener(
        urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar())
    )

    def get(url):
        with browser.open(url, timeout=30) as response:
            return response.read().decode()

    def post(url, fields, multi=False, file=None):
        if multi:
            data, content_type = multipart(fields, file)
        else:
            data = urllib.parse.urlencode(fields).encode()
            content_type = "application/x-www-form-urlencoded"
        request = urllib.request.Request(url, data=data, headers={"Content-Type": content_type})
        with browser.open(request, timeout=30) as response:
            return response.read().decode()

    process, url = launch(env)
    try:
        page = get(url)
        assert "Try the guided demo" in page and "Use my data" in page
        assert 'data-theme="light"' in page and "ZgotmplZ" not in page
        token = urllib.parse.parse_qs(urllib.parse.urlparse(url).query)["token"][0]

        def route(path):
            return urllib.parse.urljoin(url, path) + "?" + urllib.parse.urlencode({"token": token})

        page = post(route("/theme"), {"theme": "dark", "return": "/welcome"})
        assert 'data-theme="dark"' in page and "Light mode" in page
        setup = get(route("/setup"))
        assert "Set up your first task" in setup and 'data-theme="dark"' in setup
        assert not list(Path(root).glob("*/onboarding.json")), "setup marked guide complete"
        sample = get(route("/onboarding/sample") + "&name=example-logs.jsonl")
        records = [json.loads(line) for line in sample.splitlines()]
        assert len(records) == 8 and all(r["message"] == "Failed login attempt" for r in records)
        page = post(url + "&onboarding=demo", {"action": "demo"}, multi=True)
        assert 'id="results"' in page and 'href="#results"' in page
        assert "Your first result is ready" in page and "Guided demo - synthetic data" in page
        assert "192.0.2.10" in page and "ZgotmplZ" not in page
        assert 'data-theme="dark"' in page and 'name="secret"' not in page
        link = re.search(r'href="(/export[^\"]+)"', page)
        assert link, "missing export"
        export_url = urllib.parse.urljoin(url, html.unescape(link.group(1)))
        value = json.loads(get(export_url))
        assert value["Demo"] and value["Report"]["Complete"]
        assert len(value["Report"]["Alerts"]) == 1
        assert len(value["Report"]["Alerts"][0]["Evidence"]) == 6
        assert len(value["Report"]["Alerts"][0]["Sensitivity"]) == 9
        assert "How sensitive is this finding?" in page
        replays = json.loads(get(route("/replay")))
        assert [r["ObservedAlerts"] for r in replays] == [1, 0, 0, 1]
        assert all(r["MatchesExpectation"] for r in replays)
        assert list(Path(root).glob("*/onboarding.json")), "completion not remembered"
        assert "Try the guided demo" in get(route("/welcome"))
        assert json.loads(get(export_url)) == value, "guide restart modified result"
        page = post(url, {"action": "analyze", "threshold": "6", "window": "10m"}, multi=True, file=sample)
        uploaded = json.loads(get(export_url))
        assert not uploaded["Demo"] and uploaded["Report"]["Complete"]
        assert len(uploaded["Report"]["Alerts"]) == 1 and "Create reviewed Jira investigation" in page
        page = post(route("/theme"), {"theme": "light", "return": "/"})
        assert 'data-theme="light"' in page
        assert json.loads(get(export_url)) == uploaded, "theme change modified result"
        try:
            get(url.split("?")[0])
            raise AssertionError("missing token allowed")
        except urllib.error.HTTPError as error:
            assert error.code == 403
        duplicate = subprocess.run(
            [EXECUTABLE], stdin=subprocess.DEVNULL, capture_output=True,
            env=env, timeout=15, check=False,
        )
        assert duplicate.returncode == 2, "same-directory second launcher allowed"
    finally:
        stop(process)
    # terminate() is a forced stop on Windows. Only remove this disposable app's lock,
    # after the process is known to have exited, to test reopening saved data.
    for lock in Path(root).glob("*/application.lock"):
        lock.unlink()
    process, url = launch(env)
    try:
        page = get(url)
        assert "Try the guided demo" not in page and "Welcome back" in page
        assert "192.0.2.10" in page and 'data-theme="light"' in page
    finally:
        stop(process)

print("LogFlow: packaged CLI, welcome, themes, upload, demo, evidence, export, lock and restart PASS")
