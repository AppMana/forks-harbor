"""Exercise a disposable registry bound to localhost; never contacts production."""
import hashlib
import json
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

base = sys.argv[1]
assert urllib.parse.urlsplit(base).hostname == "127.0.0.1"
repo = "/v2/security-canary/image"

def request(method, path, data=None, headers=None):
    with urllib.request.urlopen(urllib.request.Request(base + path, data=data, headers=headers or {}, method=method), timeout=15) as response:
        return response.status, dict(response.headers), response.read()

for attempt in range(30):
    try:
        assert request("GET", "/v2/")[0] == 200
        break
    except (urllib.error.URLError, TimeoutError):
        if attempt == 29:
            raise
        time.sleep(1)
config = json.dumps({"architecture": "amd64", "os": "linux", "rootfs": {"type": "layers", "diff_ids": []}, "config": {}}, separators=(",", ":")).encode()
digest = "sha256:" + hashlib.sha256(config).hexdigest()
status, headers, _ = request("POST", repo + "/blobs/uploads/", b"")
assert status == 202
location = urllib.parse.urlsplit(headers["Location"])
assert location.netloc == urllib.parse.urlsplit(base).netloc
path = location.path + "?" + location.query + "&digest=" + digest
assert request("PUT", path, config, {"Content-Type": "application/octet-stream"})[0] == 201
assert request("GET", repo + "/blobs/" + digest)[2] == config
media = "application/vnd.oci.image.manifest.v1+json"
manifest = json.dumps({"schemaVersion": 2, "mediaType": media, "config": {"mediaType": "application/vnd.oci.image.config.v1+json", "size": len(config), "digest": digest}, "layers": []}, separators=(",", ":")).encode()
manifest_digest = "sha256:" + hashlib.sha256(manifest).hexdigest()
assert request("PUT", repo + "/manifests/canary", manifest, {"Content-Type": media})[0] == 201
_, headers, body = request("GET", repo + "/manifests/canary", headers={"Accept": media})
assert body == manifest and headers["Docker-Content-Digest"] == manifest_digest
assert request("DELETE", repo + "/manifests/" + manifest_digest)[0] == 202
try:
    request("GET", repo + "/manifests/canary", headers={"Accept": media})
    raise AssertionError("Deleted manifest remains readable")
except urllib.error.HTTPError as error:
    assert error.code == 404
print("Registry startup, blob upload/download, manifest push/pull, digest and deletion checks passed")
