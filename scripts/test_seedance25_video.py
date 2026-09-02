#!/usr/bin/env python3
"""
Seedance 2.5 video generation smoke test through New API.

This script asks for only the gateway Base URL and a New API virtual API key,
then submits one text-to-video task and polls until it reaches a terminal state.

Examples of accepted Base URL inputs:
  http://124.174.23.197:3000
  http://124.174.23.197:3000/v1
  http://124.174.23.197:3000/doubao

Do not use a Volcengine Ark real key here. Use the virtual key issued by New API.
"""

from __future__ import annotations

import argparse
import getpass
import json
import sys
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any
from urllib.parse import urlparse

try:
    import requests
except ImportError:  # pragma: no cover - friendly CLI fallback
    print("缺少 requests 依赖。请先执行：python -m pip install requests", file=sys.stderr)
    raise


MODEL_ID = "doubao-seedance-2-5-260628"
TERMINAL_STATUSES = {"succeeded", "failed", "expired", "cancelled"}
DEFAULT_PROMPT = (
    "A red paper airplane flying smoothly across a clean white studio background, "
    "fixed camera, no text, no logo"
)


def now_iso() -> str:
    return datetime.now(timezone.utc).astimezone().isoformat(timespec="seconds")


def normalize_base_url(raw_base_url: str) -> str:
    base_url = raw_base_url.strip().rstrip("/")
    if not base_url:
        raise ValueError("Base URL 不能为空")

    parsed = urlparse(base_url)
    if parsed.scheme not in {"http", "https"} or not parsed.netloc:
        raise ValueError("Base URL 必须形如 http://host:3000 或 https://domain")

    if base_url.endswith("/v1") or base_url.endswith("/doubao"):
        return base_url

    if base_url.endswith("/api/v3") or base_url.endswith("/doubao/api/v3"):
        raise ValueError("请填写网关根地址、/v1 或 /doubao，不要直接填到 /api/v3")

    return base_url + "/v1"


def build_task_endpoint(base_url: str) -> str:
    if base_url.endswith("/doubao"):
        return f"{base_url}/api/v3/contents/generations/tasks"
    return f"{base_url}/contents/generations/tasks"


def prompt_text(label: str, default: str | None = None) -> str:
    suffix = f" [{default}]" if default else ""
    value = input(f"{label}{suffix}: ").strip()
    return value or (default or "")


def extract_task_id(response_json: dict[str, Any]) -> str:
    task_id = response_json.get("id") or response_json.get("task_id")
    if not task_id:
        raise RuntimeError("提交响应中没有找到任务 ID，完整响应如下：\n" + json.dumps(response_json, ensure_ascii=False, indent=2))
    return str(task_id)


def extract_video_url(task_json: dict[str, Any]) -> str | None:
    content = task_json.get("content")
    if isinstance(content, dict):
        video_url = content.get("video_url")
        if isinstance(video_url, str):
            return video_url
        if isinstance(video_url, dict) and isinstance(video_url.get("url"), str):
            return video_url["url"]

    for key in ("video_url", "url"):
        value = task_json.get(key)
        if isinstance(value, str) and value.startswith(("http://", "https://")):
            return value

    output = task_json.get("output")
    if isinstance(output, dict):
        for key in ("video_url", "url"):
            value = output.get(key)
            if isinstance(value, str) and value.startswith(("http://", "https://")):
                return value

    return None


def submit_task(
    session: requests.Session,
    endpoint: str,
    api_key: str,
    prompt: str,
    resolution: str,
    duration: int,
    timeout: int,
) -> dict[str, Any]:
    payload = {
        "model": MODEL_ID,
        "content": [
            {
                "type": "text",
                "text": prompt,
            }
        ],
        "resolution": resolution,
        "duration": duration,
    }

    response = session.post(
        endpoint,
        headers={
            "Authorization": f"Bearer {api_key}",
            "Accept": "application/json",
            "Content-Type": "application/json; charset=utf-8",
        },
        json=payload,
        timeout=timeout,
    )
    response.raise_for_status()
    return response.json()


def query_task(session: requests.Session, query_endpoint: str, api_key: str, timeout: int) -> dict[str, Any]:
    response = session.get(
        query_endpoint,
        headers={
            "Authorization": f"Bearer {api_key}",
            "Accept": "application/json",
        },
        timeout=timeout,
    )
    response.raise_for_status()
    return response.json()


def save_result(result: dict[str, Any], output_dir: Path, task_id: str) -> Path:
    output_dir.mkdir(parents=True, exist_ok=True)
    safe_task_id = "".join(ch if ch.isalnum() or ch in "-_" else "_" for ch in task_id)
    path = output_dir / f"seedance25-{safe_task_id}.json"

    sanitized = dict(result)
    video_url = extract_video_url(sanitized)
    sanitized["video_url_present"] = bool(video_url)
    if video_url:
        sanitized["video_url"] = video_url

    path.write_text(json.dumps(sanitized, ensure_ascii=False, indent=2), encoding="utf-8")
    return path


def main() -> int:
    parser = argparse.ArgumentParser(description="通过 New API 中转站测试 Seedance 2.5 文生视频")
    parser.add_argument("--base-url", help="New API Base URL，例如 http://124.174.23.197:3000 或 http://124.174.23.197:3000/v1")
    parser.add_argument("--api-key", help="New API 虚拟 API Key。不建议写在命令行，推荐留空后交互输入")
    parser.add_argument("--prompt", default=None, help="视频提示词")
    parser.add_argument("--resolution", choices=["480p", "720p", "1080p"], default=None, help="输出分辨率")
    parser.add_argument("--duration", type=int, default=None, help="视频时长，Seedance 2.5 常用 5 秒")
    parser.add_argument("--poll-interval", type=int, default=5, help="轮询间隔秒数，默认 5")
    parser.add_argument("--timeout", type=int, default=1800, help="总等待超时秒数，默认 1800")
    parser.add_argument("--request-timeout", type=int, default=60, help="单次 HTTP 请求超时秒数，默认 60")
    parser.add_argument("--output-dir", default="artifacts/poc", help="结果 JSON 保存目录，默认 artifacts/poc")
    args = parser.parse_args()

    base_url_input = args.base_url or prompt_text("请输入 New API Base URL", "http://124.174.23.197:3000/v1")
    base_url = normalize_base_url(base_url_input)
    endpoint = build_task_endpoint(base_url)

    api_key = args.api_key or getpass.getpass("请输入 New API 虚拟 API Key（输入时不会显示）: ").strip()
    if not api_key:
        raise ValueError("API Key 不能为空")

    prompt = args.prompt or prompt_text("请输入视频提示词", DEFAULT_PROMPT)
    resolution = args.resolution or prompt_text("请输入分辨率：480p / 720p / 1080p", "720p")
    if resolution not in {"480p", "720p", "1080p"}:
        raise ValueError("分辨率只能是 480p、720p 或 1080p")

    duration = args.duration
    if duration is None:
        duration = int(prompt_text("请输入视频时长秒数", "5"))
    if duration < 4 or duration > 30:
        raise ValueError("视频时长建议在 4 到 30 秒之间")

    print(f"\n提交任务：{endpoint}")
    print(f"模型：{MODEL_ID}，分辨率：{resolution}，时长：{duration}s")

    session = requests.Session()
    started_at = now_iso()
    submitted = submit_task(session, endpoint, api_key, prompt, resolution, duration, args.request_timeout)
    task_id = extract_task_id(submitted)
    query_endpoint = f"{endpoint}/{task_id}"
    print(f"任务已创建：{task_id}")

    query_retry_count = 0
    deadline = time.monotonic() + args.timeout
    task = submitted

    while True:
        if time.monotonic() >= deadline:
            raise TimeoutError(f"任务轮询超时：{task_id}")

        time.sleep(max(args.poll_interval, 0))
        try:
            task = query_task(session, query_endpoint, api_key, args.request_timeout)
        except requests.RequestException as exc:
            query_retry_count += 1
            print(f"查询暂时失败，第 {query_retry_count} 次重试：{exc}")
            continue

        status = str(task.get("status", "")).lower()
        print(f"[{now_iso()}] status={status or 'unknown'}")
        if status in TERMINAL_STATUSES:
            break

    task["local_test_meta"] = {
        "started_at": started_at,
        "finished_at": now_iso(),
        "model": MODEL_ID,
        "resolution": resolution,
        "duration": duration,
        "query_retry_count": query_retry_count,
    }

    output_path = save_result(task, Path(args.output_dir), task_id)
    video_url = extract_video_url(task)
    usage = task.get("usage") if isinstance(task.get("usage"), dict) else {}

    print("\n任务终态：", task.get("status"))
    if usage:
        print("usage：", json.dumps(usage, ensure_ascii=False))
    print("视频 URL：", video_url or "未返回/未识别")
    print("结果已保存：", output_path.resolve())

    return 0 if str(task.get("status", "")).lower() == "succeeded" else 2


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except KeyboardInterrupt:
        print("\n已取消。注意：如果任务已经提交成功，取消本地脚本不会取消上游生成任务。", file=sys.stderr)
        raise SystemExit(130)
    except Exception as exc:
        print(f"\n错误：{exc}", file=sys.stderr)
        raise SystemExit(1)
