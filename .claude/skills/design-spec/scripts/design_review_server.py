#!/usr/bin/env python3
"""Serve a design plan through the fixed review viewer and persist local approval artifacts."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import signal
import socket
import subprocess
import sys
import threading
import time
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any
from urllib.error import URLError
from urllib.request import urlopen

from validate_design_plan import PlanError, validate_plan


KEBAB_CASE = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")
VIEWER_PATH = Path(__file__).resolve().parent.parent / "assets" / "review-viewer.html"
IDLE_TIMEOUT_SECONDS = 1800


def iso_now() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="milliseconds").replace("+00:00", "Z")


def atomic_write_json(path: Path, payload: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp = path.with_suffix(path.suffix + ".tmp")
    tmp.write_text(json.dumps(payload, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    tmp.replace(path)


def resolve_repo_path(repo_root: Path, value: str) -> Path:
    candidate = Path(value)
    if candidate.is_absolute():
        raise ValueError("path must be repository-relative")
    resolved_root = repo_root.resolve()
    resolved = (resolved_root / candidate).resolve()
    if resolved != resolved_root and resolved_root not in resolved.parents:
        raise ValueError("path escapes repository root")
    return resolved


def repo_relative(repo_root: Path, path: Path) -> str:
    return path.resolve().relative_to(repo_root.resolve()).as_posix()


def load_plan(plan_path: Path, repo_root: Path) -> dict[str, Any]:
    validate_plan(plan_path, repo_root)
    return json.loads(plan_path.read_text(encoding="utf-8"))


def feature_slug_for(plan_path: Path) -> str:
    """Resolve the slug for lifecycle commands without requiring a readable plan.

    Stopping a runner must keep working after the plan is deleted or broken,
    otherwise an orphan process can only be killed by hand.
    """
    try:
        slug = json.loads(plan_path.read_text(encoding="utf-8")).get("feature_slug")
        if isinstance(slug, str) and KEBAB_CASE.fullmatch(slug):
            return slug
    except (OSError, json.JSONDecodeError, AttributeError):
        pass
    if not KEBAB_CASE.fullmatch(plan_path.stem):
        raise ValueError(f"cannot resolve a feature slug from {plan_path}")
    return plan_path.stem


def require_string(value: Any, field: str) -> str:
    if not isinstance(value, str) or not value.strip():
        raise ValueError(f"{field} must be a non-empty string")
    return value.strip()


def require_string_list(value: Any, field: str) -> list[str]:
    if not isinstance(value, list):
        raise ValueError(f"{field} must be an array")
    return [require_string(item, f"{field}[{index}]") for index, item in enumerate(value)]


def plan_targets(plan: dict[str, Any]) -> set[str]:
    targets = {"general"}
    for field in ("decisions", "business_rules", "implementation_constraints", "assumptions", "risks"):
        for item in plan.get(field) or []:
            targets.add(item["id"])
    return targets


def validate_approval_payload(payload: dict[str, Any], plan: dict[str, Any]) -> dict[str, str]:
    if payload.get("schema_version") != 3:
        raise ValueError("schema_version must equal 3")
    if payload.get("event") != "design-spec-approval":
        raise ValueError("event must equal design-spec-approval")
    if payload.get("approved") is not True:
        raise ValueError("approved must equal true")
    if payload.get("approval_meaning") != "direction-approved":
        raise ValueError("approval_meaning must equal direction-approved")
    if payload.get("feature_slug") != plan["feature_slug"]:
        raise ValueError("feature_slug does not match the design plan")
    if str(payload.get("design_revision") or "") != str(plan["design_revision"]):
        raise ValueError("design_revision does not match the design plan")
    require_string(payload.get("submitted_at"), "submitted_at")
    require_string_list(payload.get("extra_constraints") or [], "extra_constraints")

    answers = payload.get("answers")
    if not isinstance(answers, dict):
        raise ValueError("answers must be an object")

    expected = {decision["id"]: decision for decision in plan.get("decisions") or []}
    if set(answers) != set(expected):
        missing = sorted(set(expected) - set(answers))
        extra = sorted(set(answers) - set(expected))
        raise ValueError(f"answers must cover every decision exactly once: missing {missing}, unknown {extra}")

    resolved: dict[str, str] = {}
    for decision_id, answer in answers.items():
        answer_text = require_string(answer, f"answers['{decision_id}']")
        allowed = {option["answer"] for option in expected[decision_id]["options"]}
        if answer_text not in allowed:
            raise ValueError(f"answers['{decision_id}'] is not one of the options offered in the design plan")
        resolved[decision_id] = answer_text
    return resolved


def validate_feedback_payload(payload: dict[str, Any], plan: dict[str, Any]) -> list[dict[str, str]]:
    if payload.get("schema_version") != 3:
        raise ValueError("schema_version must equal 3")
    if payload.get("event") != "design-change-request":
        raise ValueError("event must equal design-change-request")
    if payload.get("feature_slug") != plan["feature_slug"]:
        raise ValueError("feature_slug does not match the design plan")
    if str(payload.get("design_revision") or "") != str(plan["design_revision"]):
        raise ValueError("design_revision does not match the design plan")

    comments = payload.get("comments")
    if not isinstance(comments, list) or not comments:
        raise ValueError("comments must be a non-empty array")

    known = plan_targets(plan)
    result: list[dict[str, str]] = []
    for index, comment in enumerate(comments):
        if not isinstance(comment, dict):
            raise ValueError(f"comments[{index}] must be an object")
        target = require_string(comment.get("target"), f"comments[{index}].target")
        if target not in known:
            raise ValueError(f"comments[{index}].target '{target}' is not an id present in the design plan")
        result.append({"target": target, "text": require_string(comment.get("text"), f"comments[{index}].text")})
    return result


def make_manifest(
    repo_root: Path,
    plan_path: Path,
    plan: dict[str, Any],
    payload: dict[str, Any],
    answers: dict[str, str],
) -> dict[str, Any]:
    questions = {decision["id"]: decision["question"] for decision in plan.get("decisions") or []}
    return {
        "schema_version": 3,
        "feature_slug": plan["feature_slug"],
        "design_revision": str(plan["design_revision"]),
        "design_path": repo_relative(repo_root, plan_path),
        "design_sha256": hashlib.sha256(plan_path.read_bytes()).hexdigest(),
        "approved_at": payload["submitted_at"],
        "approval_source": "local-runner",
        "approval_meaning": "direction-approved",
        "goal": plan["goal"],
        "output_preview": plan["preview"],
        "scope": plan["scope"],
        "decisions": [
            {
                "id": decision_id,
                "question": questions[decision_id],
                "answer": answers[decision_id],
                "source": "human",
            }
            for decision_id in sorted(answers)
        ],
        "business_rules": [
            {"id": rule["id"], "statement": rule["statement"], "source": "agent-proposed-not-objected"}
            for rule in plan.get("business_rules") or []
        ],
        "implementation_constraints": [
            {
                "id": constraint["id"],
                "statement": constraint["statement"],
                "source": constraint.get("source", "agent")
                if constraint.get("source", "agent") != "agent"
                else "agent-proposed-not-objected",
            }
            for constraint in plan.get("implementation_constraints") or []
        ],
        "ai_discretion": list(plan.get("ai_discretion") or []),
        "assumptions": list(plan.get("assumptions") or []),
        "risks": list(plan.get("risks") or []),
        "evidence": list(plan.get("evidence") or []),
        "constraints": list(payload.get("extra_constraints") or []),
        "unresolved_non_blocking": [],
    }


def run_manifest_validator(repo_root: Path, manifest_path: Path) -> None:
    validator = Path(__file__).with_name("validate_design_decisions.py")
    result = subprocess.run(
        [sys.executable, str(validator), str(manifest_path), "--repo-root", str(repo_root)],
        text=True,
        capture_output=True,
        check=False,
    )
    if result.returncode != 0:
        raise ValueError((result.stderr or result.stdout or "manifest validation failed").strip())


class ReviewHandler(BaseHTTPRequestHandler):
    repo_root: Path
    plan_path: Path
    plan_rel: str
    last_activity: float = 0.0

    def handle_one_request(self) -> None:
        type(self).last_activity = time.monotonic()
        super().handle_one_request()

    def current_plan(self) -> dict[str, Any]:
        return load_plan(self.plan_path, self.repo_root)

    def log_message(self, format: str, *args: Any) -> None:
        sys.stderr.write("%s - %s\n" % (self.address_string(), format % args))

    def send_bytes(self, status: int, content_type: str, body: bytes) -> None:
        self.send_response(status)
        self.send_header("content-type", content_type)
        self.send_header("content-length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def send_json(self, status: int, payload: dict[str, Any]) -> None:
        self.send_bytes(status, "application/json; charset=utf-8", json.dumps(payload, ensure_ascii=False).encode("utf-8"))

    def read_json_body(self) -> dict[str, Any]:
        length = int(self.headers.get("content-length") or "0")
        if length <= 0 or length > 1_000_000:
            raise ValueError("request body size is invalid")
        payload = json.loads(self.rfile.read(length).decode("utf-8"))
        if not isinstance(payload, dict):
            raise ValueError("request body must be a JSON object")
        return payload

    def do_GET(self) -> None:
        try:
            if self.path in {"/", "/design", "/index.html"}:
                self.send_bytes(200, "text/html; charset=utf-8", VIEWER_PATH.read_bytes())
                return
            if self.path == "/api/plan":
                self.current_plan()
                self.send_bytes(200, "application/json; charset=utf-8", self.plan_path.read_bytes())
                return
            if self.path == "/api/status":
                plan = self.current_plan()
                self.send_json(
                    200,
                    {
                        "status": "ok",
                        "feature_slug": plan["feature_slug"],
                        "design_revision": str(plan["design_revision"]),
                        "design_path": self.plan_rel,
                    },
                )
                return
            self.send_json(404, {"error": "not found"})
        except (PlanError, ValueError) as error:
            self.send_json(400, {"error": str(error)})

    def do_POST(self) -> None:
        try:
            if self.path == "/api/design-approval":
                self.handle_approval()
                return
            if self.path == "/api/design-feedback":
                self.handle_feedback()
                return
            self.send_json(404, {"error": "not found"})
        except (json.JSONDecodeError, PlanError, ValueError) as error:
            self.send_json(400, {"error": str(error)})
        except Exception as error:
            self.send_json(500, {"error": str(error)})

    def handle_approval(self) -> None:
        payload = self.read_json_body()
        plan = self.current_plan()
        answers = validate_approval_payload(payload, plan)
        manifest_path = self.repo_root / "docs" / "ai" / "design-decisions" / f"{plan['feature_slug']}.json"
        manifest = make_manifest(self.repo_root, self.plan_path, plan, payload, answers)
        atomic_write_json(manifest_path, manifest)
        try:
            run_manifest_validator(self.repo_root, manifest_path)
        except Exception:
            manifest_path.unlink(missing_ok=True)
            raise
        self.send_json(
            200,
            {"status": "approved", "design_decisions_path": repo_relative(self.repo_root, manifest_path)},
        )

    def handle_feedback(self) -> None:
        payload = self.read_json_body()
        plan = self.current_plan()
        comments = validate_feedback_payload(payload, plan)
        feedback_path = self.repo_root / "docs" / "ai" / "design-feedback" / f"{plan['feature_slug']}.json"
        atomic_write_json(
            feedback_path,
            {
                "schema_version": 3,
                "event": "design-change-request",
                "feature_slug": plan["feature_slug"],
                "design_revision": str(plan["design_revision"]),
                "design_path": self.plan_rel,
                "design_sha256": hashlib.sha256(self.plan_path.read_bytes()).hexdigest(),
                "received_at": iso_now(),
                "comments": comments,
            },
        )
        self.send_json(
            200,
            {"status": "changes-requested", "design_feedback_path": repo_relative(self.repo_root, feedback_path)},
        )


def state_path(repo_root: Path, feature_slug: str) -> Path:
    return repo_root / "docs" / "ai" / "design-runtime" / f"{feature_slug}.server.json"


def clear_state(repo_root: Path, feature_slug: str, pid: int) -> None:
    """Drop the state file once this runner exits, so status never reports a dead server."""
    path = state_path(repo_root, feature_slug)
    state = read_state(path)
    if state and int(state.get("pid") or 0) == pid:
        path.unlink(missing_ok=True)


def start_idle_watchdog(server: ThreadingHTTPServer, handler: type[ReviewHandler], timeout: float) -> None:
    """Shut the runner down after `timeout` seconds without a single request.

    The viewer sends a heartbeat while the review page is open, so an open tab
    keeps the runner alive and a closed one lets it exit on its own.
    """

    def watch() -> None:
        interval = min(30.0, max(1.0, timeout / 10))
        while True:
            time.sleep(interval)
            if time.monotonic() - handler.last_activity >= timeout:
                sys.stderr.write(f"idle for {timeout:.0f}s with no request; shutting down\n")
                server.shutdown()
                return

    threading.Thread(target=watch, daemon=True).start()


def read_state(path: Path) -> dict[str, Any] | None:
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (FileNotFoundError, json.JSONDecodeError):
        return None
    return data if isinstance(data, dict) else None


def pid_running(pid: int) -> bool:
    if pid <= 0:
        return False
    try:
        os.kill(pid, 0)
    except OSError:
        return False
    return True


def choose_port(host: str) -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind((host, 0))
        return int(sock.getsockname()[1])


def wait_for_status(url: str, timeout: float = 5.0) -> bool:
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            with urlopen(url + "/api/status", timeout=0.5) as response:
                return response.status == 200
        except URLError:
            time.sleep(0.1)
    return False


def fetch_status(url: str, timeout: float = 1.0) -> dict[str, Any] | None:
    try:
        with urlopen(url + "/api/status", timeout=timeout) as response:
            payload = json.loads(response.read().decode("utf-8"))
    except (OSError, URLError, json.JSONDecodeError):
        return None
    return payload if isinstance(payload, dict) else None


def start_command(args: argparse.Namespace) -> int:
    repo_root = args.repo_root.resolve()
    plan_path = resolve_repo_path(repo_root, args.design_path)
    plan = load_plan(plan_path, repo_root)
    feature_slug = plan["feature_slug"]
    design_revision = str(plan["design_revision"])
    plan_rel = repo_relative(repo_root, plan_path)
    state = state_path(repo_root, feature_slug)
    existing = read_state(state)
    if existing and pid_running(int(existing.get("pid") or 0)):
        status = fetch_status(str(existing.get("url") or ""))
        matches = status and all(
            (
                status.get("feature_slug") == feature_slug,
                str(status.get("design_revision") or "") == design_revision,
                status.get("design_path") == plan_rel,
            )
        )
        if matches:
            print(json.dumps({"status": "running", **existing}, ensure_ascii=False))
            return 0
        os.kill(int(existing["pid"]), signal.SIGTERM)

    port = args.port or choose_port(args.host)
    state.parent.mkdir(parents=True, exist_ok=True)
    log_path = state.with_suffix(".log")
    command = [
        sys.executable,
        str(Path(__file__).resolve()),
        "--repo-root",
        str(repo_root),
        "serve",
        plan_rel,
        "--host",
        args.host,
        "--port",
        str(port),
        "--idle-timeout",
        str(args.idle_timeout),
    ]
    with log_path.open("ab") as log:
        process = subprocess.Popen(command, stdin=subprocess.DEVNULL, stdout=log, stderr=log, start_new_session=True)
    url = f"http://{args.host}:{port}"
    payload = {
        "pid": process.pid,
        "url": url,
        "host": args.host,
        "port": port,
        "feature_slug": feature_slug,
        "design_revision": design_revision,
        "design_path": plan_rel,
        "state_path": repo_relative(repo_root, state),
        "log_path": repo_relative(repo_root, log_path),
        "idle_timeout_seconds": args.idle_timeout,
        "started_at": iso_now(),
    }
    atomic_write_json(state, payload)
    if not wait_for_status(url):
        raise SystemExit(f"server did not become healthy; see {log_path}")
    print(json.dumps({"status": "started", **payload}, ensure_ascii=False))
    return 0


def status_command(args: argparse.Namespace) -> int:
    repo_root = args.repo_root.resolve()
    plan_path = resolve_repo_path(repo_root, args.design_path)
    feature_slug = feature_slug_for(plan_path)
    state = read_state(state_path(repo_root, feature_slug)) or {}
    running = pid_running(int(state.get("pid") or 0))
    manifest = repo_root / "docs" / "ai" / "design-decisions" / f"{feature_slug}.json"
    feedback = repo_root / "docs" / "ai" / "design-feedback" / f"{feature_slug}.json"
    print(
        json.dumps(
            {
                "status": "running" if running else "stopped",
                "server": state,
                "design_decisions_path": repo_relative(repo_root, manifest) if manifest.exists() else None,
                "design_feedback_path": repo_relative(repo_root, feedback) if feedback.exists() else None,
            },
            ensure_ascii=False,
        )
    )
    return 0


def stop_command(args: argparse.Namespace) -> int:
    repo_root = args.repo_root.resolve()
    plan_path = resolve_repo_path(repo_root, args.design_path)
    feature_slug = feature_slug_for(plan_path)
    state = read_state(state_path(repo_root, feature_slug)) or {}
    pid = int(state.get("pid") or 0)
    if pid_running(pid):
        os.kill(pid, signal.SIGTERM)
    print(json.dumps({"status": "stopped", "feature_slug": feature_slug, "pid": pid}, ensure_ascii=False))
    return 0


def serve_command(args: argparse.Namespace) -> int:
    repo_root = args.repo_root.resolve()
    plan_path = resolve_repo_path(repo_root, args.design_path)
    load_plan(plan_path, repo_root)
    if not VIEWER_PATH.is_file():
        raise SystemExit(f"review viewer asset is missing: {VIEWER_PATH}")

    class Handler(ReviewHandler):
        pass

    Handler.repo_root = repo_root
    Handler.plan_path = plan_path
    Handler.plan_rel = repo_relative(repo_root, plan_path)
    Handler.last_activity = time.monotonic()

    server = ThreadingHTTPServer((args.host, args.port), Handler)
    if args.idle_timeout > 0:
        start_idle_watchdog(server, Handler, float(args.idle_timeout))
    signal.signal(signal.SIGTERM, lambda *_: threading.Thread(target=server.shutdown, daemon=True).start())
    try:
        server.serve_forever()
    finally:
        clear_state(repo_root, feature_slug_for(plan_path), os.getpid())
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo-root", type=Path, default=Path.cwd())
    subparsers = parser.add_subparsers(dest="command", required=True)
    for name in ("start", "status", "stop", "serve"):
        command = subparsers.add_parser(name)
        command.add_argument("design_path", help="Repository-relative path to docs/ai/designs/{slug}.json")
        command.add_argument("--host", default="127.0.0.1")
        command.add_argument("--port", type=int, default=0)
        command.add_argument(
            "--idle-timeout",
            type=int,
            default=IDLE_TIMEOUT_SECONDS,
            help="Seconds without any request before the runner exits; 0 disables the timeout",
        )
        command.set_defaults(func=globals()[f"{name}_command"])
    return parser


def main() -> int:
    args = build_parser().parse_args()
    return int(args.func(args))


if __name__ == "__main__":
    raise SystemExit(main())
