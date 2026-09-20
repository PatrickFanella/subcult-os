# Subcults and OS Evidence
Earlier research in this task reviewed Subcults at 3cf88ec on fix/main-regression-recovery and Subcult OS base abf3f50. This is a dated source snapshot, not current production proof. The previous Subcult Research Dossier remains available separately.

Subcults source includes public profile/act/place/venue/scene/event/tour/appearance/assertion record families under tv.subcult.*; touring SQL, consent/suppression and delivery boundary implementations; OAuth/PDS record and projection patterns. Key anchors: internal/touring/sql_repository.go, internal/audience/service.go, internal/signal/delivery.go, lexicons/README.md and accepted ADR 0007 distinguishing public Scene from private Workspace.

OS source includes workspaces, commitments, staffing, role applications, reservation/ticket, check-in and settlement/archive paths in backend/internal/app/app.go and schema.sql. These do not independently prove a complete refunds, offline, accounting or payments operation.

Earlier release/readiness material retains provider/device, expiry, restore/capacity and stream-parity gates. Test file existence is not a pass; historic local passes are not deployment qualification. No new application tests, paid provider calls or live user journeys were performed for these funding claims.

Before sharing: recheck exact HEADs, dirty state, license and contributor ownership. Supply a disposable, reproducible demo rather than exposing local data or operational credentials.
