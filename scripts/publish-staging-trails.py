"""Explicit, repeatable import of the reviewed routes in content/staging-trails to staging only."""

import hashlib
import http.cookiejar
import json
import os
from pathlib import Path
import urllib.error
import urllib.request
import uuid

API = "https://test.api.ermekten.com"
ROOT = Path(__file__).resolve().parents[1] / "content/staging-trails"
client = urllib.request.build_opener(
    urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar())
)


def request(path, method="GET", data=None, content_type="application/json"):
    body = json.dumps(data).encode() if isinstance(data, dict) else data
    req = urllib.request.Request(
        API + path, data=body, method=method,
        headers={"Content-Type": content_type, "Origin": "https://test.admin.ermekten.com"},
    )
    with client.open(req, timeout=60) as response:
        return json.load(response)


def upload(path, mime):
    boundary = uuid.uuid4().hex
    body = (
        f'--{boundary}\r\nContent-Disposition: form-data; name="file"; '
        f'filename="{path.name}"\r\nContent-Type: {mime}\r\n\r\n'
    ).encode() + path.read_bytes() + f"\r\n--{boundary}--\r\n".encode()
    return request("/uploads/file", "POST", body, f"multipart/form-data; boundary={boundary}")["url"]


def verify_file(url, path):
    if not url.startswith("/uploads/"):
        raise RuntimeError("Unexpected attachment URL")
    with client.open(API + url, timeout=60) as response:
        digest = hashlib.sha256(response.read()).digest()
    if digest != hashlib.sha256(path.read_bytes()).digest():
        raise RuntimeError(f"Attachment mismatch: {path.name}")


def main():
    request("/auth/login", "POST", {"password": os.environ["ADMIN_PASSWORD"]})
    try:
        existing = {post["slug"]: post for post in request("/trails/admin/all")}
        for file in sorted(ROOT.glob("*/post.json")):
            post = json.loads(file.read_text())
            old = existing.get(post["slug"])
            if old and old["name"] != post["name"]:
                raise RuntimeError(f"Slug belongs to another post: {post['slug']}")
            for field, name, mime in [
                ("gpxUrl", "route.gpx", "application/gpx+xml"),
                ("checklistUrl", "checklist.pdf", "application/pdf"),
            ]:
                post[field] = old.get(field) if old else None
                if not post[field]:
                    post[field] = upload(file.parent / name, mime)
                verify_file(post[field], file.parent / name)
            post["isPublished"] = True
            if old:
                request("/trails/" + old["id"], "PUT", post)
            else:
                request("/trails", "POST", post)
            live = request("/trails/" + post["slug"])
            for field in ["track", "markers", "gpxUrl", "checklistUrl", "isPublished"]:
                if live[field] != post[field]:
                    raise RuntimeError(f"Published data mismatch: {field}")
            print("Published and verified: https://test.ermekten.com/trails/" + post["slug"])
    finally:
        request("/auth/logout", "POST")


if __name__ == "__main__":
    main()
