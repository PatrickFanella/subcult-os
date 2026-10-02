#!/usr/bin/env python3
"""Export the Subcult OS web brand assets from a subcult-studio checkout.

The Studio pack owns the masters. This script copies or derives the site
selection and records source and target hashes in docs/brand-provenance.json.
`--check` compares the committed targets with that record and needs neither
Studio nor the render tools. Exporting needs Pillow and rsvg-convert.
"""

import argparse
import hashlib
import json
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PROVENANCE = ROOT / "docs/brand-provenance.json"
PACK = "branding/2026-10-01-packs/brands/subcult-os"
REPOSITORY = "https://git.subcult.tv/subculture-collective/subcult-studio"

# The outlined mark sits in a 256 px canvas; its purple frame spans 50..206.
MARK_CROP = 'width="156" height="156" viewBox="50 50 156 156"'
OG_SIZE = (1200, 630)


def sha256(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def crop_mark(source, target):
    svg = source.read_text()
    cropped, count = re.subn(r'width="256"\s+height="256"\s+viewBox="0 0 256 256"', MARK_CROP, svg, count=1)
    if count != 1:
        raise SystemExit(f"Unexpected mark canvas in {source}")
    target.write_text(cropped)


def render_favicon(source, target):
    subprocess.run(["rsvg-convert", "-w", "32", "-h", "32", "-o", str(target), str(source)], check=True)


def resize_avatar(source, target):
    from PIL import Image

    Image.open(source).convert("RGB").resize((180, 180), Image.LANCZOS).save(target, optimize=True)


def letterbox_banner(source, target):
    from PIL import Image

    banner = Image.open(source).convert("RGB")
    height = round(banner.height * OG_SIZE[0] / banner.width)
    canvas = Image.new("RGB", OG_SIZE, "#000000")
    canvas.paste(banner.resize((OG_SIZE[0], height), Image.LANCZOS), (0, (OG_SIZE[1] - height) // 2))
    canvas.save(target, optimize=True)


# target, pack source, derivation, description recorded in the provenance file
SELECTION = [
    ("web/public/brand/mark.svg", "logos/mark-primary-outlined.svg", crop_mark,
     "viewBox cropped to the mark frame; paths unchanged"),
    ("web/public/favicon-32.png", "logos/mark-primary-outlined.svg", None,
     "rsvg-convert render of web/public/brand/mark.svg at 32x32"),
    ("web/public/apple-touch-icon.png", "social/avatar.png", resize_avatar,
     "resized to 180x180"),
    ("web/public/og-image.png", "social/banner-x-bluesky.png", letterbox_banner,
     "scaled to 1200 px wide and centered on a 1200x630 #000000 canvas"),
]


def export(studio):
    studio = studio.resolve()
    pack = studio / PACK
    manifest = json.loads((pack / "asset-manifest.json").read_text())
    hashes = {entry["path"]: entry["sha256"] for entry in manifest["files"]}
    files = []
    for target, source, derive, derivation in SELECTION:
        source_path = pack / source
        digest = sha256(source_path)
        if hashes.get(source) != digest:
            raise SystemExit(f"Source manifest mismatch: {source}")
        target_path = ROOT / target
        target_path.parent.mkdir(parents=True, exist_ok=True)
        if derive:
            derive(source_path, target_path)
        else:
            render_favicon(ROOT / "web/public/brand/mark.svg", target_path)
        files.append({
            "source": f"{PACK}/{source}",
            "sourceSha256": digest,
            "target": target,
            "derivation": derivation,
            "sha256": sha256(target_path),
        })
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
    expected = {target for target, *_ in SELECTION}
    failures = [f"selection differs from provenance: {sorted(expected ^ set(recorded))}"] if expected != set(recorded) else []
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
    group.add_argument("--studio-root", type=Path, help="subcult-studio checkout to export from")
    group.add_argument("--check", action="store_true", help="verify committed assets against the provenance record")
    args = parser.parse_args()
    check() if args.check else export(args.studio_root)
