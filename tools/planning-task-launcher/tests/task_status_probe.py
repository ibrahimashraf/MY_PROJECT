#!/usr/bin/env python3
"""Read only one launcher-created task status without printing task contents."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from urllib.parse import urlencode

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "scripts"))
from planning_task_launcher import LauncherError, api_request, load_config  # noqa: E402


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", required=True)
    parser.add_argument("--task-id", required=True)
    parser.add_argument("--mode", choices=("detail", "messages"), default="detail")
    args = parser.parse_args()
    try:
        _, config, _ = load_config(args.config)
        if args.mode == "detail":
            response = api_request(config, "GET", f"/v2/task.detail?{urlencode({'task_id': args.task_id})}")
            task = response.get("task", {})
            status = task.get("status") if isinstance(task, dict) else None
            print(json.dumps({"ok": True, "task_id": args.task_id, "latest_status": status, "query": "detail"}, sort_keys=True))
            return 0
        query = urlencode({"task_id": args.task_id, "order": "desc", "limit": 20})
        response = api_request(config, "GET", f"/v2/task.listMessages?{query}")
        statuses = [item.get("status_update", {}).get("agent_status") for item in response.get("messages", []) if isinstance(item, dict) and item.get("type") == "status_update"]
        print(json.dumps({"ok": True, "task_id": args.task_id, "latest_status": statuses[0] if statuses else None, "status_event_count": len(statuses), "query": "messages"}, sort_keys=True))
        return 0
    except LauncherError as error:
        print(json.dumps({"ok": False, "reason": str(error)}, sort_keys=True))
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
