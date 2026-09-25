#!/usr/bin/env node
// Synthetic OFFLINE-01 harness. It creates no production ticket cache, route,
// database row, network request, account, or device evidence.
import { mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import { createHash, randomUUID } from "node:crypto";

const [mode, inputPath, outputPath] = process.argv.slice(2);

if (mode === "--client") {
  const input = JSON.parse(readFileSync(inputPath, "utf8"));
  mkdirSync(input.storageDir, { recursive: true });
  const operations = input.scans.map((ticketCode, index) => ({
    id: `${input.clientId}:${randomUUID()}`,
    clientId: input.clientId,
    snapshotId: input.snapshot.id,
    snapshotRevision: input.snapshot.revision,
    eventId: input.snapshot.eventId,
    ticketCode,
    recordedAt: input.recordedAt,
  }));
  writeFileSync(join(input.storageDir, "synthetic-client-state.json"), JSON.stringify({ clientId: input.clientId, snapshotId: input.snapshot.id }));
  writeFileSync(outputPath, JSON.stringify({ clientId: input.clientId, storageDir: input.storageDir, operations }));
  process.exit(0);
}

if (mode !== undefined) {
  throw new Error("usage: node scripts/offline-door-experiment.mjs [--client input output]");
}

const root = mkdtempSync(join(tmpdir(), "subcult-offline-door-lab-"));
try {
  const snapshot = {
    id: "synthetic-snapshot-1",
    eventId: "synthetic-event-1",
    revision: "synthetic-roster-r1",
    expiresAt: "2026-09-25T18:15:00Z",
    tickets: { "SYNTH-ONE": { admissionEligible: true, checkedIn: false } },
  };
  const clients = ["synthetic-client-a", "synthetic-client-b"].map((clientId) => {
    const storageDir = join(root, clientId);
    const input = join(root, `${clientId}.json`);
    const output = join(root, `${clientId}-journal.json`);
    writeFileSync(input, JSON.stringify({ clientId, storageDir, snapshot, scans: ["SYNTH-ONE"], recordedAt: "2026-09-25T18:01:00Z" }));
    const child = spawnSync(process.execPath, [process.argv[1], "--client", input, output], { encoding: "utf8" });
    if (child.status !== 0) throw new Error(`synthetic client ${clientId} failed: ${child.stderr}`);
    return { pid: child.pid, ...JSON.parse(readFileSync(output, "utf8")) };
  });

  const results = new Map();
  const operations = new Map();
  const checkedIn = new Set();
  const merge = (operation, now, revoked = false) => {
    if (results.has(operation.id)) {
      const original = operations.get(operation.id);
      if (JSON.stringify(original) !== JSON.stringify(operation)) return { operationId: operation.id, status: "operation_conflict" };
      return { operationId: operation.id, status: "duplicate_operation", duplicateOf: results.get(operation.id).status };
    }
    let status = "accepted";
    if (revoked) status = "revoked_client_review";
    else if (operation.eventId !== snapshot.eventId || operation.snapshotId !== snapshot.id || operation.snapshotRevision !== snapshot.revision) status = "stale_snapshot";
    else if (now >= snapshot.expiresAt) status = "expired_snapshot";
    else if (!snapshot.tickets[operation.ticketCode]?.admissionEligible) status = "ineligible";
    else if (snapshot.tickets[operation.ticketCode]?.checkedIn || checkedIn.has(operation.ticketCode)) status = "duplicate_check_in";
    else checkedIn.add(operation.ticketCode);
    const result = { operationId: operation.id, status };
    results.set(operation.id, result);
    operations.set(operation.id, operation);
    return result;
  };

  // Deliberately reconnect B before A: the merge, not either partitioned
  // client, decides the winner. The second upload models a lost response.
  const b = clients[1].operations[0];
  const a = clients[0].operations[0];
  const report = {
    fixtureOnly: true,
    fixtureHash: createHash("sha256").update(JSON.stringify(snapshot)).digest("hex"),
    clientProcesses: clients.map(({ clientId, pid, storageDir }) => ({ clientId, pid, storageDir })),
    reconnect: [merge(b, "2026-09-25T18:02:00Z"), merge(a, "2026-09-25T18:02:00Z"), merge(b, "2026-09-25T18:02:00Z")],
    expiry: merge({ ...a, id: "synthetic-client-a:expired" }, "2026-09-25T18:16:00Z"),
    revocation: merge({ ...a, id: "synthetic-client-a:revoked" }, "2026-09-25T18:02:00Z", true),
    manualFallback: {
      automaticMerge: false,
      requiredFields: ["responsibleOperator", "numberedRecord", "recordedAt", "serverReconciliation"],
    },
    limits: "No physical-device, scanner, OS-cache, network-radio, provider, or real-door evidence.",
  };
  const statuses = report.reconnect.map((result) => result.status);
  if (statuses.join(",") !== "accepted,duplicate_check_in,duplicate_operation" || report.expiry.status !== "expired_snapshot" || report.revocation.status !== "revoked_client_review" || clients[0].storageDir === clients[1].storageDir) {
    throw new Error("synthetic fixture invariant failed");
  }
  process.stdout.write(`${JSON.stringify(report, null, 2)}\n`);
} finally {
  rmSync(root, { recursive: true, force: true });
}
