#!/usr/bin/env python3

import argparse
import json
import os
import sqlite3
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional


SCHEMA_VERSION = "ai-workflow/session-trace-v1"

OPENCODE_DB_DEFAULT = Path.home() / ".local" / "share" / "opencode" / "opencode.db"
OPENCODE_FILE_TOOLS = {"edit", "write", "apply_patch"}


def project_path_to_claude_key(project_path: str) -> str:
    return project_path.replace("/", "-")


def ms_to_iso(value):
    if not isinstance(value, (int, float)):
        return None
    return datetime.fromtimestamp(value / 1000, tz=timezone.utc).isoformat().replace("+00:00", "Z")


def read_jsonl(file_path: Path):
    records = []
    with file_path.open("r", encoding="utf-8") as handle:
        for line_number, line in enumerate(handle, start=1):
            stripped = line.strip()
            if not stripped:
                continue
            try:
                records.append(json.loads(stripped))
            except json.JSONDecodeError as exc:
                raise ValueError(
                    f"Failed to parse JSONL line {line_number} in {file_path}: {exc}"
                ) from exc
    return records


def read_first_jsonl(file_path: Path):
    with file_path.open("r", encoding="utf-8") as handle:
        for line in handle:
            stripped = line.strip()
            if stripped:
                return json.loads(stripped)
    return None


def list_jsonl_files(root_dir: Path):
    if not root_dir.exists():
        return []
    return [path for path in root_dir.rglob("*.jsonl") if path.is_file()]


def detect_runtime(input_path: Optional[Path]):
    if input_path is None:
        return None
    normalized = input_path.as_posix()
    if "/.claude/" in normalized:
        return "claude"
    if "/.codex/" in normalized:
        return "codex"
    if "/opencode/" in normalized or normalized.endswith(".db"):
        return "opencode"
    return None


def pick_latest_file(files):
    if not files:
        return None
    return max(files, key=lambda item: item.stat().st_mtime_ns)


def find_latest_transcript(runtime: str, project_path: Optional[str]):
    home = Path.home()
    normalized_project = os.path.abspath(os.path.expanduser(project_path)) if project_path else None

    if runtime == "claude":
        files = list_jsonl_files(home / ".claude" / "projects")
        if normalized_project:
            project_key = project_path_to_claude_key(normalized_project)
            files = [path for path in files if project_key in path.as_posix()]
        return pick_latest_file(files)

    if runtime == "codex":
        files = list_jsonl_files(home / ".codex" / "sessions")
        if normalized_project:
            filtered = []
            for file_path in files:
                try:
                    first_record = read_first_jsonl(file_path) or {}
                except Exception:
                    continue
                payload = first_record.get("payload") or {}
                if payload.get("cwd") == normalized_project:
                    filtered.append(file_path)
            files = filtered
        return pick_latest_file(files)

    raise ValueError(f"Unsupported runtime for --latest: {runtime}")


def ensure_dir(dir_path: Path):
    dir_path.mkdir(parents=True, exist_ok=True)


def write_json(file_path: Path, value):
    ensure_dir(file_path.parent)
    with file_path.open("w", encoding="utf-8") as handle:
        json.dump(value, handle, indent=2, ensure_ascii=True)
        handle.write("\n")


def write_ndjson(file_path: Path, items):
    ensure_dir(file_path.parent)
    with file_path.open("w", encoding="utf-8") as handle:
        for item in items:
            handle.write(json.dumps(item, ensure_ascii=True))
            handle.write("\n")


def safe_json_parse(value):
    if not isinstance(value, str):
        return value
    try:
        return json.loads(value)
    except json.JSONDecodeError:
        return value


def sanitize_file_name(value):
    return "".join(char if char.isalnum() or char in "._-" else "-" for char in str(value or "unknown"))


def increment_counter(counter, key):
    safe_key = key or "unknown"
    counter[safe_key] = counter.get(safe_key, 0) + 1


def build_base_artifact(runtime: str, transcript_path: Path):
    return {
        "schema_version": SCHEMA_VERSION,
        "runtime": runtime,
        "source": {
            "transcript_path": str(transcript_path),
            "transcript_format": "jsonl",
            "extracted_at": __import__("datetime").datetime.utcnow().isoformat() + "Z",
            "extractor": "workflow-evaluation/extract_session_trace.py",
        },
        "session": {
            "id": None,
            "cwd": None,
            "started_at": None,
            "ended_at": None,
            "cli_version": None,
            "model": None,
            "git_branch": None,
            "metadata": {},
        },
        "session_trace": {
            "chat_history": [],
            "command_transcript": [],
            "tool_call_trace": [],
            "artifact_trail": [],
            "handoff_notes": [],
            "decision_log": [],
            "failure_retry_log": [],
        },
        "normalized_events": [],
        "stats": {
            "raw_event_types": {},
            "normalized_event_types": {},
        },
        "extraction_notes": [],
    }


def push_normalized_event(artifact, event):
    artifact["normalized_events"].append(event)
    increment_counter(artifact["stats"]["normalized_event_types"], event.get("kind"))


def push_if_text(chat_history, entry):
    text = entry.get("text")
    if isinstance(text, str) and text.strip():
        chat_history.append(entry)


def extract_claude_local_command(content: str):
    if "<command-name>" not in content:
        return None

    def extract(tag):
        start = f"<{tag}>"
        end = f"</{tag}>"
        if start not in content or end not in content:
            return None
        return content.split(start, 1)[1].split(end, 1)[0].strip()

    return {
        "command_name": extract("command-name"),
        "command_message": extract("command-message"),
        "command_args": extract("command-args"),
    }


def normalize_claude_blocks(content):
    if isinstance(content, str):
        return [{"type": "text", "text": content}]
    if isinstance(content, list):
        return content
    return []


def parse_claude_transcript(records, transcript_path: Path):
    artifact = build_base_artifact("claude", transcript_path)
    tool_calls = {}

    for index, record in enumerate(records):
        increment_counter(artifact["stats"]["raw_event_types"], record.get("type"))
        timestamp = record.get("timestamp") or ((record.get("snapshot") or {}).get("timestamp"))

        if artifact["session"]["id"] is None and record.get("sessionId"):
            artifact["session"]["id"] = record.get("sessionId")
        if artifact["session"]["cwd"] is None and record.get("cwd"):
            artifact["session"]["cwd"] = record.get("cwd")
        if artifact["session"]["started_at"] is None and timestamp:
            artifact["session"]["started_at"] = timestamp
        if timestamp:
            artifact["session"]["ended_at"] = timestamp
        if artifact["session"]["cli_version"] is None and record.get("version"):
            artifact["session"]["cli_version"] = record.get("version")
        if artifact["session"]["git_branch"] is None and record.get("gitBranch"):
            artifact["session"]["git_branch"] = record.get("gitBranch")

        record_type = record.get("type")
        if record_type in {"user", "assistant"}:
            message = record.get("message") or {}
            role = message.get("role") or record_type
            content = message.get("content")

            if isinstance(content, str):
                local_command = extract_claude_local_command(content)
                if local_command:
                    artifact["session_trace"]["command_transcript"].append({
                        "index": index,
                        "timestamp": timestamp,
                        "runtime": "claude",
                        "source": "local_command",
                        "role": role,
                        **local_command,
                    })
                    push_normalized_event(artifact, {
                        "index": index,
                        "timestamp": timestamp,
                        "kind": "command",
                        "role": role,
                        "text": local_command.get("command_message"),
                        "command_name": local_command.get("command_name"),
                    })
                else:
                    push_if_text(artifact["session_trace"]["chat_history"], {
                        "index": index,
                        "timestamp": timestamp,
                        "runtime": "claude",
                        "role": role,
                        "text": content,
                    })
                    push_normalized_event(artifact, {
                        "index": index,
                        "timestamp": timestamp,
                        "kind": "chat",
                        "role": role,
                        "text": content,
                    })

            for block in normalize_claude_blocks(content):
                block_type = block.get("type")
                if block_type == "text":
                    push_if_text(artifact["session_trace"]["chat_history"], {
                        "index": index,
                        "timestamp": timestamp,
                        "runtime": "claude",
                        "role": role,
                        "text": block.get("text", ""),
                    })
                    push_normalized_event(artifact, {
                        "index": index,
                        "timestamp": timestamp,
                        "kind": "chat",
                        "role": role,
                        "text": block.get("text", ""),
                    })
                elif block_type == "tool_use":
                    call_id = block.get("id")
                    tool_name = block.get("name")
                    tool_input = block.get("input")
                    tool_call = {
                        "index": index,
                        "timestamp": timestamp,
                        "runtime": "claude",
                        "direction": "call",
                        "call_id": call_id,
                        "tool_name": tool_name,
                        "input": tool_input,
                    }
                    artifact["session_trace"]["tool_call_trace"].append(tool_call)
                    if call_id:
                        tool_calls[call_id] = tool_call

                    command_value = None
                    if isinstance(tool_input, dict):
                        command_value = tool_input.get("command") or tool_input.get("cmd")
                    if command_value:
                        artifact["session_trace"]["command_transcript"].append({
                            "index": index,
                            "timestamp": timestamp,
                            "runtime": "claude",
                            "source": "tool_use",
                            "call_id": call_id,
                            "tool_name": tool_name,
                            "command": command_value,
                            "description": tool_input.get("description"),
                        })

                    push_normalized_event(artifact, {
                        "index": index,
                        "timestamp": timestamp,
                        "kind": "tool_call",
                        "role": role,
                        "tool_name": tool_name,
                        "call_id": call_id,
                        "command": command_value,
                    })
                elif block_type == "tool_result":
                    call_id = block.get("tool_use_id")
                    prior_call = tool_calls.get(call_id, {})
                    tool_name = prior_call.get("tool_name")
                    is_error = bool(block.get("is_error"))
                    tool_result = {
                        "index": index,
                        "timestamp": timestamp,
                        "runtime": "claude",
                        "direction": "result",
                        "call_id": call_id,
                        "tool_name": tool_name,
                        "output": block.get("content"),
                        "is_error": is_error,
                    }
                    artifact["session_trace"]["tool_call_trace"].append(tool_result)
                    artifact["session_trace"]["command_transcript"].append({
                        "index": index,
                        "timestamp": timestamp,
                        "runtime": "claude",
                        "source": "tool_result",
                        "call_id": call_id,
                        "tool_name": tool_name,
                        "output": block.get("content"),
                        "is_error": is_error,
                    })
                    if is_error:
                        artifact["session_trace"]["failure_retry_log"].append({
                            "index": index,
                            "timestamp": timestamp,
                            "runtime": "claude",
                            "source": "tool_result",
                            "call_id": call_id,
                            "tool_name": tool_name,
                            "error": block.get("content") or "Tool returned an error.",
                        })
                    push_normalized_event(artifact, {
                        "index": index,
                        "timestamp": timestamp,
                        "kind": "tool_result",
                        "role": role,
                        "tool_name": tool_name,
                        "call_id": call_id,
                        "is_error": is_error,
                    })

        elif record_type == "attachment":
            attachment = record.get("attachment") or {}
            artifact["session_trace"]["artifact_trail"].append({
                "index": index,
                "timestamp": timestamp,
                "runtime": "claude",
                "type": attachment.get("type"),
                "tool_use_id": attachment.get("toolUseID"),
                "command": attachment.get("command"),
                "exit_code": attachment.get("exitCode"),
            })
            if attachment.get("type") == "hook_success":
                artifact["session_trace"]["handoff_notes"].append({
                    "index": index,
                    "timestamp": timestamp,
                    "runtime": "claude",
                    "hook_name": attachment.get("hookName"),
                    "hook_event": attachment.get("hookEvent"),
                    "tool_use_id": attachment.get("toolUseID"),
                })
            push_normalized_event(artifact, {
                "index": index,
                "timestamp": timestamp,
                "kind": "artifact",
                "role": None,
                "artifact_type": attachment.get("type"),
            })

        elif record_type == "file-history-snapshot":
            snapshot = record.get("snapshot") or {}
            tracked_files = list((snapshot.get("trackedFileBackups") or {}).keys())
            artifact["session_trace"]["artifact_trail"].append({
                "index": index,
                "timestamp": timestamp,
                "runtime": "claude",
                "type": "file-history-snapshot",
                "tracked_files": tracked_files,
                "is_snapshot_update": bool(record.get("isSnapshotUpdate")),
            })
            push_normalized_event(artifact, {
                "index": index,
                "timestamp": timestamp,
                "kind": "artifact",
                "role": None,
                "artifact_type": "file-history-snapshot",
            })

        elif record_type == "system":
            if record.get("subtype") == "stop_hook_summary":
                artifact["session_trace"]["handoff_notes"].append({
                    "index": index,
                    "timestamp": timestamp,
                    "runtime": "claude",
                    "type": "stop_hook_summary",
                    "prevented_continuation": bool(record.get("preventedContinuation")),
                    "stop_reason": record.get("stopReason") or "",
                })
            push_normalized_event(artifact, {
                "index": index,
                "timestamp": timestamp,
                "kind": "system",
                "role": None,
                "subtype": record.get("subtype"),
            })

    artifact["session"]["metadata"]["project_key"] = (
        project_path_to_claude_key(artifact["session"]["cwd"])
        if artifact["session"]["cwd"]
        else None
    )
    artifact["extraction_notes"].append(
        "Claude transcript combines chat messages, tool calls, tool results, attachments, and file-history snapshots in one JSONL stream."
    )
    return artifact


def extract_codex_text(content):
    if not isinstance(content, list):
        return []
    texts = []
    for part in content:
        if isinstance(part, dict) and part.get("type") in {"input_text", "output_text"}:
            text = str(part.get("text") or "")
            if text.strip():
                texts.append(text)
    return texts


def parse_codex_transcript(records, transcript_path: Path):
    artifact = build_base_artifact("codex", transcript_path)
    tool_calls = {}

    for index, record in enumerate(records):
        increment_counter(artifact["stats"]["raw_event_types"], record.get("type"))
        timestamp = record.get("timestamp")
        payload = record.get("payload") or {}
        record_type = record.get("type")

        if record_type == "session_meta":
            artifact["session"]["id"] = payload.get("session_id") or payload.get("id")
            artifact["session"]["cwd"] = payload.get("cwd")
            artifact["session"]["started_at"] = payload.get("timestamp")
            artifact["session"]["cli_version"] = payload.get("cli_version")
            artifact["session"]["metadata"]["originator"] = payload.get("originator")
            artifact["session"]["metadata"]["model_provider"] = payload.get("model_provider")
            artifact["session"]["metadata"]["source"] = payload.get("source")
            artifact["session"]["metadata"]["thread_source"] = payload.get("thread_source")
            artifact["session"]["metadata"]["history_mode"] = payload.get("history_mode")
            artifact["session"]["metadata"]["context_window"] = payload.get("context_window")

        elif record_type == "turn_context":
            artifact["session"]["cwd"] = payload.get("cwd") or artifact["session"]["cwd"]
            artifact["session"]["model"] = payload.get("model") or artifact["session"]["model"]
            artifact["session"]["metadata"]["timezone"] = payload.get("timezone")
            artifact["session"]["metadata"]["approval_policy"] = payload.get("approval_policy")
            artifact["session"]["metadata"]["personality"] = payload.get("personality")
            artifact["session"]["metadata"]["collaboration_mode"] = payload.get("collaboration_mode")

        if artifact["session"]["started_at"] is None and timestamp:
            artifact["session"]["started_at"] = timestamp
        if timestamp:
            artifact["session"]["ended_at"] = timestamp

        if record_type == "response_item":
            payload_type = payload.get("type")
            if payload_type == "message":
                for text in extract_codex_text(payload.get("content")):
                    push_if_text(artifact["session_trace"]["chat_history"], {
                        "index": index,
                        "timestamp": timestamp,
                        "runtime": "codex",
                        "role": payload.get("role"),
                        "text": text,
                    })
                    push_normalized_event(artifact, {
                        "index": index,
                        "timestamp": timestamp,
                        "kind": "chat",
                        "role": payload.get("role"),
                        "text": text,
                    })
            elif payload_type == "function_call":
                call_id = payload.get("call_id")
                input_value = safe_json_parse(payload.get("arguments"))
                tool_call = {
                    "index": index,
                    "timestamp": timestamp,
                    "runtime": "codex",
                    "direction": "call",
                    "call_id": call_id,
                    "tool_name": payload.get("name"),
                    "input": input_value,
                }
                artifact["session_trace"]["tool_call_trace"].append(tool_call)
                if call_id:
                    tool_calls[call_id] = tool_call

                command_value = None
                if isinstance(input_value, dict):
                    command_value = input_value.get("cmd") or input_value.get("command")
                if command_value:
                    artifact["session_trace"]["command_transcript"].append({
                        "index": index,
                        "timestamp": timestamp,
                        "runtime": "codex",
                        "source": "function_call",
                        "call_id": call_id,
                        "tool_name": payload.get("name"),
                        "command": command_value,
                    })

                push_normalized_event(artifact, {
                    "index": index,
                    "timestamp": timestamp,
                    "kind": "tool_call",
                    "role": None,
                    "tool_name": payload.get("name"),
                    "call_id": call_id,
                    "command": command_value,
                })
            elif payload_type == "function_call_output":
                call_id = payload.get("call_id")
                prior_call = tool_calls.get(call_id, {})
                tool_name = prior_call.get("tool_name")
                artifact["session_trace"]["tool_call_trace"].append({
                    "index": index,
                    "timestamp": timestamp,
                    "runtime": "codex",
                    "direction": "result",
                    "call_id": call_id,
                    "tool_name": tool_name,
                    "output": payload.get("output"),
                })
                artifact["session_trace"]["command_transcript"].append({
                    "index": index,
                    "timestamp": timestamp,
                    "runtime": "codex",
                    "source": "function_call_output",
                    "call_id": call_id,
                    "tool_name": tool_name,
                    "output": payload.get("output"),
                })
                push_normalized_event(artifact, {
                    "index": index,
                    "timestamp": timestamp,
                    "kind": "tool_result",
                    "role": None,
                    "tool_name": tool_name,
                    "call_id": call_id,
                })
            elif payload_type == "reasoning":
                artifact["session_trace"]["decision_log"].append({
                    "index": index,
                    "timestamp": timestamp,
                    "runtime": "codex",
                    "source": "reasoning",
                    "summary": payload.get("summary"),
                })
                push_normalized_event(artifact, {
                    "index": index,
                    "timestamp": timestamp,
                    "kind": "reasoning",
                    "role": None,
                })

        elif record_type == "event_msg":
            payload_type = payload.get("type")
            if payload_type == "agent_message":
                artifact["session_trace"]["handoff_notes"].append({
                    "index": index,
                    "timestamp": timestamp,
                    "runtime": "codex",
                    "source": "agent_message",
                    "phase": payload.get("phase"),
                    "message": payload.get("message"),
                })
            elif payload_type in {"task_started", "task_complete"}:
                artifact["session_trace"]["artifact_trail"].append({
                    "index": index,
                    "timestamp": timestamp,
                    "runtime": "codex",
                    "type": payload_type,
                    "turn_id": payload.get("turn_id"),
                    "duration_ms": payload.get("duration_ms"),
                })
            elif payload_type in {"web_search_call", "web_search_end"}:
                artifact["session_trace"]["tool_call_trace"].append({
                    "index": index,
                    "timestamp": timestamp,
                    "runtime": "codex",
                    "direction": "call" if payload_type == "web_search_call" else "result",
                    "call_id": payload.get("call_id"),
                    "tool_name": "web_search",
                    "input": payload.get("query") or payload.get("action"),
                })
            elif payload_type == "token_count":
                artifact["session_trace"]["artifact_trail"].append({
                    "index": index,
                    "timestamp": timestamp,
                    "runtime": "codex",
                    "type": "token_count",
                    "total_token_usage": (payload.get("info") or {}).get("total_token_usage"),
                })

            push_normalized_event(artifact, {
                "index": index,
                "timestamp": timestamp,
                "kind": "event",
                "role": None,
                "subtype": payload_type,
            })

        elif record_type == "world_state":
            artifact["session_trace"]["artifact_trail"].append({
                "index": index,
                "timestamp": timestamp,
                "runtime": "codex",
                "type": "world_state",
                "full": bool(payload.get("full")),
            })
            push_normalized_event(artifact, {
                "index": index,
                "timestamp": timestamp,
                "kind": "artifact",
                "role": None,
                "artifact_type": "world_state",
            })

    artifact["extraction_notes"].append(
        "Codex transcript combines response items, runtime context, world state, and event messages in one JSONL stream."
    )
    return artifact


def resolve_opencode_db_path(explicit_path: Optional[Path]):
    if explicit_path is not None:
        return Path(os.path.expanduser(str(explicit_path))).resolve()
    if OPENCODE_DB_DEFAULT.exists():
        return OPENCODE_DB_DEFAULT
    try:
        result = subprocess.run(
            ["opencode", "db", "path"],
            capture_output=True,
            text=True,
            check=True,
        )
    except (OSError, subprocess.CalledProcessError) as exc:
        raise FileNotFoundError(
            f"Opencode database not found at {OPENCODE_DB_DEFAULT} and `opencode db path` failed: {exc}"
        ) from exc
    return Path(os.path.expanduser(result.stdout.strip())).resolve()


def opencode_connect(db_path: Path):
    if not db_path.exists():
        raise FileNotFoundError(f"Opencode database not found: {db_path}")
    connection = sqlite3.connect(f"{db_path.as_uri()}?mode=ro", uri=True)
    connection.row_factory = sqlite3.Row
    return connection


def find_latest_opencode_session(connection, project_path: Optional[str]):
    query = "select id from session where time_archived is null"
    params = []
    if project_path:
        query += " and directory = ?"
        params.append(os.path.abspath(os.path.expanduser(project_path)))
    query += " order by time_updated desc limit 1"
    row = connection.execute(query, params).fetchone()
    return row["id"] if row else None


def read_opencode_session_row(connection, session_id: str):
    row = connection.execute("select * from session where id = ?", (session_id,)).fetchone()
    return dict(row) if row is not None else None


def read_opencode_parts(connection, session_id: str):
    return connection.execute(
        "select p.id as part_id, p.time_created as part_time, p.data as part_data, "
        "m.data as message_data "
        "from part p join message m on m.id = p.message_id "
        "where p.session_id = ? "
        "order by m.time_created, p.time_created, p.id",
        (session_id,),
    ).fetchall()


def parse_opencode_session(connection, session_id: str, db_path: Path):
    session_row = read_opencode_session_row(connection, session_id)
    if session_row is None:
        raise ValueError(f"Opencode session not found: {session_id}")

    artifact = build_base_artifact("opencode", db_path)
    artifact["source"]["transcript_format"] = "sqlite"
    artifact["source"]["session_id"] = session_id

    artifact["session"]["id"] = session_row.get("id")
    artifact["session"]["cwd"] = session_row.get("directory")
    artifact["session"]["started_at"] = ms_to_iso(session_row.get("time_created"))
    artifact["session"]["cli_version"] = session_row.get("version")
    artifact["session"]["model"] = safe_json_parse(session_row.get("model")) or None
    for key in ("title", "slug", "project_id", "workspace_id", "parent_id", "path", "agent", "cost"):
        if key in session_row:
            artifact["session"]["metadata"][key] = session_row.get(key)
    artifact["session"]["metadata"]["archived"] = session_row.get("time_archived") is not None
    artifact["session"]["metadata"]["session_totals"] = {
        "tokens_input": session_row.get("tokens_input"),
        "tokens_output": session_row.get("tokens_output"),
        "tokens_reasoning": session_row.get("tokens_reasoning"),
        "tokens_cache_read": session_row.get("tokens_cache_read"),
        "tokens_cache_write": session_row.get("tokens_cache_write"),
        "summary_additions": session_row.get("summary_additions"),
        "summary_deletions": session_row.get("summary_deletions"),
        "summary_files": session_row.get("summary_files"),
    }

    totals = {"input": 0, "output": 0, "reasoning": 0, "cost": 0}
    reasoning_parts = 0

    for index, row in enumerate(read_opencode_parts(connection, session_id)):
        part = safe_json_parse(row["part_data"]) or {}
        message = safe_json_parse(row["message_data"]) or {}
        if not isinstance(part, dict):
            continue
        part_type = part.get("type")
        role = message.get("role") if isinstance(message, dict) else None
        timestamp = ms_to_iso(row["part_time"])

        increment_counter(artifact["stats"]["raw_event_types"], part_type)
        if timestamp:
            artifact["session"]["ended_at"] = timestamp
        if artifact["session"]["model"] is None and isinstance(message, dict) and message.get("modelID"):
            artifact["session"]["model"] = message.get("modelID")
            artifact["session"]["metadata"]["provider"] = message.get("providerID")

        if part_type == "text":
            text = part.get("text")
            push_if_text(artifact["session_trace"]["chat_history"], {
                "index": index,
                "timestamp": timestamp,
                "runtime": "opencode",
                "role": role,
                "text": text,
            })
            if isinstance(text, str) and text.strip():
                push_normalized_event(artifact, {
                    "index": index,
                    "timestamp": timestamp,
                    "kind": "chat",
                    "role": role,
                    "text": text,
                })

        elif part_type == "reasoning":
            reasoning_parts += 1

        elif part_type == "tool":
            state = part.get("state") or {}
            call_id = part.get("callID")
            tool_name = part.get("tool")
            input_value = state.get("input")
            status = state.get("status")

            artifact["session_trace"]["tool_call_trace"].append({
                "index": index,
                "timestamp": timestamp,
                "runtime": "opencode",
                "direction": "call",
                "call_id": call_id,
                "tool_name": tool_name,
                "input": input_value,
            })

            command_value = None
            if isinstance(input_value, dict):
                command_value = input_value.get("command") or input_value.get("cmd")
            if command_value:
                artifact["session_trace"]["command_transcript"].append({
                    "index": index,
                    "timestamp": timestamp,
                    "runtime": "opencode",
                    "source": "tool_call",
                    "call_id": call_id,
                    "tool_name": tool_name,
                    "command": command_value,
                })

            push_normalized_event(artifact, {
                "index": index,
                "timestamp": timestamp,
                "kind": "tool_call",
                "role": None,
                "tool_name": tool_name,
                "call_id": call_id,
                "command": command_value,
            })

            if status in {"completed", "error"}:
                artifact["session_trace"]["tool_call_trace"].append({
                    "index": index,
                    "timestamp": timestamp,
                    "runtime": "opencode",
                    "direction": "result",
                    "call_id": call_id,
                    "tool_name": tool_name,
                    "status": status,
                    "output": state.get("output"),
                    "error": state.get("error"),
                })
                artifact["session_trace"]["command_transcript"].append({
                    "index": index,
                    "timestamp": timestamp,
                    "runtime": "opencode",
                    "source": "tool_result",
                    "call_id": call_id,
                    "tool_name": tool_name,
                    "status": status,
                    "output": state.get("output"),
                    "error": state.get("error"),
                })
                push_normalized_event(artifact, {
                    "index": index,
                    "timestamp": timestamp,
                    "kind": "tool_result",
                    "role": None,
                    "tool_name": tool_name,
                    "call_id": call_id,
                    "status": status,
                })
                if status == "error":
                    artifact["session_trace"]["failure_retry_log"].append({
                        "index": index,
                        "timestamp": timestamp,
                        "runtime": "opencode",
                        "call_id": call_id,
                        "tool_name": tool_name,
                        "error": state.get("error"),
                    })

            file_path = input_value.get("filePath") if isinstance(input_value, dict) else None
            if file_path and tool_name in OPENCODE_FILE_TOOLS:
                artifact["session_trace"]["artifact_trail"].append({
                    "index": index,
                    "timestamp": timestamp,
                    "runtime": "opencode",
                    "type": "file_change",
                    "tool_name": tool_name,
                    "path": file_path,
                })
                push_normalized_event(artifact, {
                    "index": index,
                    "timestamp": timestamp,
                    "kind": "artifact",
                    "role": None,
                    "artifact_type": "file_change",
                })

        elif part_type == "patch":
            artifact["session_trace"]["artifact_trail"].append({
                "index": index,
                "timestamp": timestamp,
                "runtime": "opencode",
                "type": "patch",
                "hash": part.get("hash"),
                "files": part.get("files"),
            })
            push_normalized_event(artifact, {
                "index": index,
                "timestamp": timestamp,
                "kind": "artifact",
                "role": None,
                "artifact_type": "patch",
            })

        elif part_type == "file":
            artifact["session_trace"]["artifact_trail"].append({
                "index": index,
                "timestamp": timestamp,
                "runtime": "opencode",
                "type": "file",
                "mime": part.get("mime"),
                "filename": part.get("filename"),
            })
            push_normalized_event(artifact, {
                "index": index,
                "timestamp": timestamp,
                "kind": "artifact",
                "role": None,
                "artifact_type": "file",
            })

        elif part_type == "step-finish":
            tokens = part.get("tokens") or {}
            for key in ("input", "output", "reasoning"):
                value = tokens.get(key)
                if isinstance(value, (int, float)):
                    totals[key] += value
            cost = part.get("cost")
            if isinstance(cost, (int, float)):
                totals["cost"] += cost

    artifact["session"]["metadata"]["token_totals"] = totals
    artifact["session"]["metadata"]["reasoning_parts"] = reasoning_parts

    artifact["extraction_notes"].append(
        "Opencode history is a SQLite database, not a JSONL transcript; parts are joined to their parent message for role and model."
    )
    artifact["extraction_notes"].append(
        "Reasoning parts are counted in raw_event_types and session.metadata.reasoning_parts, but their text is never emitted."
    )
    artifact["extraction_notes"].append(
        "Binary file parts keep mime and filename only; embedded data URLs are dropped."
    )
    artifact["extraction_notes"].append(
        "A tool part carries both call and result, so each completed or failed tool produces one call entry and one result entry at the same index."
    )
    return artifact


def extract_opencode(args, input_path: Optional[Path]):
    db_path = resolve_opencode_db_path(input_path)
    connection = opencode_connect(db_path)
    try:
        session_id = args.session_id
        if not session_id:
            if not args.latest:
                raise ValueError("Provide --session-id or --latest for the opencode runtime.")
            session_id = find_latest_opencode_session(connection, args.project)
            if not session_id:
                raise ValueError("No opencode session found for the requested project.")
        return parse_opencode_session(connection, session_id, db_path), db_path
    finally:
        connection.close()


def write_artifact_set(artifact, output_dir: Path):
    ensure_dir(output_dir)
    write_json(output_dir / "session-trace.json", artifact)
    write_json(output_dir / "metadata.json", {
        "schema_version": artifact["schema_version"],
        "runtime": artifact["runtime"],
        "source": artifact["source"],
        "session": artifact["session"],
        "stats": artifact["stats"],
        "extraction_notes": artifact["extraction_notes"],
    })
    write_ndjson(output_dir / "chat-history.ndjson", artifact["session_trace"]["chat_history"])
    write_ndjson(output_dir / "command-transcript.ndjson", artifact["session_trace"]["command_transcript"])
    write_ndjson(output_dir / "tool-call-trace.ndjson", artifact["session_trace"]["tool_call_trace"])
    write_ndjson(output_dir / "artifact-trail.ndjson", artifact["session_trace"]["artifact_trail"])
    write_json(output_dir / "handoff-notes.json", artifact["session_trace"]["handoff_notes"])
    write_json(output_dir / "decision-log.json", artifact["session_trace"]["decision_log"])
    write_json(output_dir / "failure-retry-log.json", artifact["session_trace"]["failure_retry_log"])


def parse_args():
    parser = argparse.ArgumentParser(description="Normalize Claude Code, Codex, or Opencode session history for workflow evaluation.")
    parser.add_argument("--input", help="Raw local transcript path, or the opencode database path.")
    parser.add_argument("--runtime", choices=["claude", "codex", "opencode"], help="Runtime name.")
    parser.add_argument("--session-id", help="Opencode session id. Required for the opencode runtime unless --latest is used.")
    parser.add_argument("--latest", action="store_true", help="Pick the latest transcript or session for the runtime.")
    parser.add_argument("--project", help="Filter latest transcript or session by project/cwd path.")
    parser.add_argument("--output-dir", help="Target output directory.")
    parser.add_argument("--stdout", action="store_true", help="Print session-trace.json to stdout.")
    return parser.parse_args()


def main():
    args = parse_args()
    input_path = Path(os.path.expanduser(args.input)).resolve() if args.input else None
    runtime = args.runtime or detect_runtime(input_path)
    if runtime is None and args.session_id:
        runtime = "opencode"

    if runtime == "opencode":
        artifact, input_path = extract_opencode(args, input_path)
    else:
        if input_path is None and args.latest:
            if runtime is None:
                raise ValueError("--runtime is required when using --latest.")
            input_path = find_latest_transcript(runtime, args.project)

        if input_path is None:
            raise ValueError("Provide --input or use --latest.")
        if runtime is None:
            runtime = detect_runtime(input_path)
        if runtime not in {"claude", "codex"}:
            raise ValueError(f"Unsupported runtime: {runtime}")
        if not input_path.exists():
            raise FileNotFoundError(f"Transcript not found: {input_path}")

        records = read_jsonl(input_path)
        artifact = parse_claude_transcript(records, input_path) if runtime == "claude" else parse_codex_transcript(records, input_path)

    if artifact["session"]["id"] is None:
        artifact["session"]["id"] = input_path.stem
        artifact["extraction_notes"].append("Session id was inferred from the transcript file name.")

    default_output = Path.cwd() / "docs" / "ai" / "session-traces" / runtime / sanitize_file_name(artifact["session"]["id"])
    output_dir = Path(os.path.expanduser(args.output_dir)).resolve() if args.output_dir else default_output

    write_artifact_set(artifact, output_dir)

    summary = {
        "runtime": artifact["runtime"],
        "transcript_path": str(input_path),
        "output_dir": str(output_dir),
        "session_id": artifact["session"]["id"],
        "cwd": artifact["session"]["cwd"],
        "started_at": artifact["session"]["started_at"],
        "counts": {
            "chat_history": len(artifact["session_trace"]["chat_history"]),
            "command_transcript": len(artifact["session_trace"]["command_transcript"]),
            "tool_call_trace": len(artifact["session_trace"]["tool_call_trace"]),
            "artifact_trail": len(artifact["session_trace"]["artifact_trail"]),
            "handoff_notes": len(artifact["session_trace"]["handoff_notes"]),
            "decision_log": len(artifact["session_trace"]["decision_log"]),
            "failure_retry_log": len(artifact["session_trace"]["failure_retry_log"]),
            "normalized_events": len(artifact["normalized_events"]),
        },
    }
    print(json.dumps(summary, indent=2, ensure_ascii=True))

    if args.stdout:
        print(json.dumps(artifact, indent=2, ensure_ascii=True))


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:  # pragma: no cover
        print(str(exc), file=sys.stderr)
        sys.exit(1)
