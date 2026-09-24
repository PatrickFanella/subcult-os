#!/usr/bin/env python3
"""Rehearse synthetic DB/key recovery using the installed T3 test environment.

No database URL or key inputs are accepted. Both databases are private tmpfs
services. Keys exist only in process/container environments during this run.
"""

import base64
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import secrets
import socket
import subprocess
import sys
from datetime import datetime, timezone


def main():
    if len(sys.argv) != 1:
        raise SystemExit("Usage: python3 scripts/qa-backup-recovery.py (no arguments)")
    launcher = Path.home() / ".local/share/t3-dev-environments/dev.py"
    if not launcher.is_file():
        raise SystemExit("Install the T3 development environment before running this rehearsal")
    spec = importlib.util.spec_from_file_location("t3dev", launcher)
    dev = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(dev)
    root = dev.source_root()
    if root != Path(__file__).resolve().parent.parent:
        raise SystemExit("Run from this script's source checkout")
    env = dev.Environment(root, "subcult-os", test=True)
    # Extend only this invocation; preserve installed recipes and other projects.
    env.recipe = {**env.recipe, "database": "backup_rehearsal", "start": "",
                  "services": {}, "test_env": {}}
    env.recipe["services"]["restored"] = {
        "image": env.recipe["db"],
        "environment": {"POSTGRES_USER": "dev", "POSTGRES_PASSWORD": "disposable-dev-only",
                        "POSTGRES_DB": "backup_rehearsal"},
        "tmpfs": ["/var/lib/postgresql/data"], "mem_limit": "768m", "cpus": 1, "pids_limit": 128,
        "healthcheck": {"test": ["CMD-SHELL", "pg_isready -h 127.0.0.1 -U dev -d backup_rehearsal"],
                        "interval": "2s", "timeout": "3s", "retries": 45},
    }
    os.umask(0o077)
    evidence = Path.home() / ".local/state/subcult-os/backup-rehearsal" / datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%S%fZ")
    evidence.mkdir(parents=True, mode=0o700)
    receipt = {"host": socket.gethostname(), "source": str(root), "project": env.project,
               "launcher_sha256": hashlib.sha256(launcher.read_bytes()).hexdigest(),
               "command": "python3 scripts/qa-backup-recovery.py", "phases": [],
               "scope": "synthetic source-code recovery; no deployed secret-store qualification",
               "result": "failed", "cleanup": "not-started"}
    compose = ["docker", "compose", "--env-file", "/dev/null", "-p", env.project, "-f", str(env.file)]

    def command(args, **kwargs):
        return subprocess.run(args, check=True, **kwargs)

    def output(args):
        return command(args, stdout=subprocess.PIPE, text=True).stdout.strip()

    def phase(name, identity, signing):
        child_env = {**os.environ, "RECOVERY_PHASE": name,
                     "RECOVERY_IDENTITY_KEY": identity, "RECOVERY_SIGNING_KEY": signing,
                     "RECOVERY_DB_URL": "postgres://dev:disposable-dev-only@" +
                     ("db" if name == "seed" else "restored") + ":5432/backup_rehearsal?sslmode=disable"}
        args = compose + ["run", "--rm", "--no-deps"]
        for name_env in ("RECOVERY_PHASE", "RECOVERY_DB_URL", "RECOVERY_IDENTITY_KEY", "RECOVERY_SIGNING_KEY"):
            args += ["-e", name_env]
        command(args + ["checks", "/data/recovery.test", "-test.run=^TestBackupRecoveryFixture$", "-test.v"], env=child_env)
        receipt["phases"].append(name)

    print(f"Recovery evidence: {evidence}\nWaiting for the worktree and host test locks.", flush=True)
    console_out, console_err = os.dup(1), os.dup(2)
    with (evidence / "run.log").open("w") as log:
        os.dup2(log.fileno(), 1)
        os.dup2(log.fileno(), 2)
        try:
            with dev.lock(env.directory / "lock"), dev.lock(dev.STATE / "host-test.lock"):
                existing = output(["docker", "ps", "-aq", "--filter", f"label=com.docker.compose.project={env.project}"])
                if existing:
                    raise RuntimeError("existing test containers belong to this project; refusing to replace them")
                receipt["revision"] = output(["git", "rev-parse", "HEAD"])
                receipt["dirty_state"] = output(["git", "status", "--short"])
                try:
                    env.setup()
                    names = json.loads((env.directory / "source-files.json").read_text())
                    manifest = {name: hashlib.sha256((env.source / name).read_bytes()).hexdigest() for name in names}
                    (evidence / "source-sha256.json").write_text(json.dumps(manifest, indent=2) + "\n")
                    receipt["source_manifest_sha256"] = hashlib.sha256((evidence / "source-sha256.json").read_bytes()).hexdigest()
                    # A TCP SQL healthcheck avoids the image's temporary bootstrap server.
                    config = json.loads(env.file.read_text())
                    config["services"]["db"]["healthcheck"] = env.recipe["services"]["restored"]["healthcheck"]
                    config["services"]["db"]["pids_limit"] = 128
                    env.file.write_text(json.dumps(config, indent=2))
                    command(compose + ["up", "-d", "--wait", "--wait-timeout", "180", "db", "restored"])
                    source_id = output(compose + ["ps", "-q", "db"])
                    receipt["database_image"] = output(["docker", "inspect", "--format", "{{.Image}}", source_id])
                    restored_id = output(compose + ["ps", "-q", "restored"])
                    receipt["restored_image"] = output(["docker", "inspect", "--format", "{{.Image}}", restored_id])
                    if receipt["database_image"] != receipt["restored_image"]:
                        raise RuntimeError("source and restored database images differ")
                    receipt["tools_image"] = output(["docker", "image", "inspect", "--format", "{{.Id}}", dev.IMAGE])
                    command(compose + ["run", "--rm", "--no-deps", "checks", "bash", "-euc",
                                       "cd backend; go test -tags recovery_rehearsal -c -o /data/recovery.test ./internal/app; go build -o /data/recovery-keygen ./cmd/atproto-keygen"])
                    identity = base64.b64encode(secrets.token_bytes(32)).decode()
                    signing = output(compose + ["run", "--rm", "--no-deps", "checks", "/data/recovery-keygen"])
                    wrong_signing = output(compose + ["run", "--rm", "--no-deps", "checks", "/data/recovery-keygen"])
                    phase("seed", identity, signing)
                    dump = evidence / "synthetic.dump"
                    with dump.open("wb") as target:
                        command(compose + ["exec", "-T", "db", "pg_dump", "-U", "dev", "-d", "backup_rehearsal", "-Fc", "--no-owner", "--no-acl"], stdout=target)
                    receipt["backup_bytes"] = dump.stat().st_size
                    receipt["backup_sha256"] = hashlib.sha256(dump.read_bytes()).hexdigest()
                    command(compose + ["stop", "db"])
                    command(compose + ["rm", "-f", "db"])
                    if output(compose + ["ps", "-aq", "db"]):
                        raise RuntimeError("source database still exists")
                    receipt["source_removed_before_restore"] = True
                    with dump.open("rb") as source:
                        command(compose + ["exec", "-T", "restored", "pg_restore", "-U", "dev", "-d", "backup_rehearsal", "--no-owner", "--no-acl", "--exit-on-error"], stdin=source)
                    receipt["phases"].append("restore")
                    phase("wrong-identity", base64.b64encode(secrets.token_bytes(32)).decode(), signing)
                    phase("wrong-signing", identity, wrong_signing)
                    phase("verify", identity, signing)
                    receipt["result"] = "passed"
                finally:
                    # The preflight check and both locks establish ownership.
                    if env.file.exists():
                        cleaned = subprocess.run(compose + ["--profile", "tools", "down", "--volumes"], check=False)
                        remaining = output(["docker", "ps", "-aq", "--filter", f"label=com.docker.compose.project={env.project}"])
                        receipt["cleanup"] = "passed" if cleaned.returncode == 0 and not remaining else "failed"
                        if receipt["cleanup"] != "passed":
                            receipt["result"] = "failed"
                    for binary in ("recovery.test", "recovery-keygen"):
                        (env.directory / "data" / binary).unlink(missing_ok=True)
        except Exception as error:
            receipt["result"] = "failed"
            # Do not serialize exception arguments or environments containing keys.
            receipt["error_type"] = type(error).__name__
        finally:
            (evidence / "receipt.json").write_text(json.dumps(receipt, indent=2) + "\n")
            sys.stdout.flush()
            sys.stderr.flush()
            os.dup2(console_out, 1)
            os.dup2(console_err, 2)
            os.close(console_out)
            os.close(console_err)
    print(f"Recovery {receipt['result']}; cleanup {receipt['cleanup']}. Receipt: {evidence / 'receipt.json'}")
    return 0 if receipt["result"] == "passed" else 1


if __name__ == "__main__":
    sys.exit(main())
