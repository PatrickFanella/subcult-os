#!/usr/bin/env python3
"""Export the Subcult OS brand assets from a subcult-tv checkout.

Studio sources moved into subcult-tv under studio/ on October 2, 2026.

The Studio pack owns the masters and exports each file at its final size, so
this script only copies bytes and records their hashes in
docs/brand-provenance.json. `--check` compares the committed targets with
that record and does not need Studio.
"""

import argparse
import hashlib
import json
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PROVENANCE = ROOT / "docs/brand-provenance.json"
PACK = "studio/branding/library/brands/subcult-os"
REPOSITORY = "https://git.subcult.tv/subculture-collective/subcult-tv"

# target in this repository -> source in the pack
SELECTION = {
    "web/public/brand/mark.svg": "product/mark-tight-outlined.svg",
    "web/public/favicon-32.png": "product/favicon-32.png",
    "web/public/apple-touch-icon.png": "product/apple-touch-icon.png",
    "web/public/og-image.png": "product/og.png",
    "mobile/assets/icon.png": "product/app-icon.png",
    "mobile/assets/android-icon-foreground.png": "product/android-adaptive-foreground.png",
    "mobile/assets/android-icon-monochrome.png": "product/android-adaptive-monochrome.png",
    "mobile/assets/splash-icon.png": "product/splash-icon.png",
    "mobile/assets/favicon.png": "product/favicon-48.png",
}


def sha256(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def export(studio):
    studio = studio.resolve()
    pack = studio / PACK
    manifest = json.loads((pack / "asset-manifest.json").read_text())
    hashes = {entry["path"]: entry["sha256"] for entry in manifest["files"]}
    files = []
    for target, source in SELECTION.items():
        digest = sha256(pack / source)
        if hashes.get(source) != digest:
            raise SystemExit(f"Source manifest mismatch: {source}")
        target_path = ROOT / target
        target_path.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(pack / source, target_path)
        files.append({"source": f"{PACK}/{source}", "target": target, "sha256": digest})
    revision = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=studio, text=True).strip()
    provenance = {
        "repository": REPOSITORY,
        "revision": revision,
        "brand": manifest["brand"],
        "direction": manifest["direction"],
        "status": manifest["status"],
        "files": files,
    }
    PROVENANCE.write_text(json.dumps(provenance, indent=2) + "\n")
    print(f"Exported {len(files)} Subcult OS brand assets; provenance written.")


def check():
    provenance = json.loads(PROVENANCE.read_text())
    recorded = {entry["target"]: entry["sha256"] for entry in provenance["files"]}
    failures = [f"selection differs from provenance: {sorted(set(SELECTION) ^ set(recorded))}"] if set(SELECTION) != set(recorded) else []
    for target, digest in recorded.items():
        path = ROOT / target
        if not path.is_file():
            failures.append(f"missing {target}")
        elif sha256(path) != digest:
            failures.append(f"{target} differs from docs/brand-provenance.json")
    if failures:
        print("\n".join(failures), file=sys.stderr)
        raise SystemExit("Brand assets drifted; re-export with scripts/brand-assets.py --studio-root")
    print(f"Brand assets match provenance ({len(recorded)} files).")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--studio-root", type=Path, help="subcult-tv checkout to export from (its studio/ directory holds the pack)")
    group.add_argument("--check", action="store_true", help="verify committed assets against the provenance record")
    args = parser.parse_args()
    check() if args.check else export(args.studio_root)
