# Upstream contribution plan
## Scope
Use the integrated application as a test bed for small reusable contributions, not as evidence that a new protocol standard has been accepted.

## Sequence
1. Build a minimal failing lifecycle or compatibility fixture in the application.
2. Determine whether the issue belongs in application policy, an SDK, sync tooling or the core protocol.
3. Search existing upstream issues/designs and contribution rules.
4. Submit a minimal reproducer or documentation clarification only with authorization to publish.
5. Offer a tested patch when appropriate.
6. Keep application-specific behavior out of a core proposal unless multiple independent consumers need it.

## Candidate packages
- Record lifecycle fixture harness: create/update/delete, stale CID, lost response, replay and recovery.
- Public payload boundary fixtures: prove internal fields cannot enter public representations.
- Namespace migration examples: explicit old/new reader behavior and no silent dual-publishing.
- Organizational authorship experiments: scope and revocation without claiming official real-world authority.
- Maintainer recovery guide: reproducible rebuild and incident handoff.

Public schemas belong under controlled namespaces and need interoperability review. AT supports application-specific Lexicons and AppViews; a new application record type is not automatically a core protocol change. [Protocol overview](https://atproto.com/specs/atp).
Review current [Lexicon compatibility guidance](https://atproto.com/guides/lexicon) before publication.

## Acceptance for extraction
An independent consumer can run the package without private Subcult APIs, credentials or datasets. License and contributor rights are reviewed. Fixtures describe supported and unsupported behavior. A named maintainer owns issues and compatibility releases.
Success can be a small upstream fix or an external reproduction finding. It need not be a new foundation, universal namespace or dedicated infrastructure service.

## Not yet promised
No new private AT storage primitive, cryptographic consent portability, generalized reputation graph, universal organization registry or guaranteed upstream merge. Coordinate with existing ecosystem work instead of duplicating it to fit a grant narrative.
