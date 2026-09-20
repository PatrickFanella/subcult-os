---
type: decision-register
status: draft
created: 2026-09-19
research_as_of: 2026-09-19
tags:
  - subcult-research
---

# Subcult Decisions and Open Questions

All decisions here are **proposed**, not accepted by the user or project maintainers.

| Decision | Recommendation | Evidence needed before commitment |
| --- | --- | --- |
| Brand | One Subcult brand, participant experience plus Studio | Naming review and interviews testing whether “Studio” is understood |
| First buyer | Recurring independent collectives and small venues | Budget-holder interviews, not attendee enthusiasm alone |
| Entry workflow | Public event plus commitments, scoped announcement and reusable closeout | Two complete event cycles per pilot team |
| Ticketing | Coexist with an existing provider first | Authorized export/API access and tested cancellation handling |
| Technical integration | Preserve service boundaries before repository consolidation | Identity, event authority and failure-recovery contract |
| Public schemas | Interoperate with community calendars without silent dual publishing | Mapping ADR and an independent reader |
| Private data | Application database initially; synthetic Spaces research only | Security review, recovery and revocation evidence |
| Revenue | Workspace subscription plus transparent variable delivery costs | Paid commitments and observed contribution margin |
| Protocol effort | Small fixed budget, first spent on calendar/sync fixtures | External maintainer interest and a second consumer |

## Questions that change the business

Who feels the pain strongly enough to buy: a venue manager, volunteer collective, artist manager or promoter? Those roles may cooperate without sharing budgets. A participant community is not itself a software customer.

Which manual artifact is recreated every event? Request examples: run sheets, artist confirmations, contact lists, settlement spreadsheets. If the pain is only ticket discovery, this proposal may be too operational. If the pain is only booking contracts, existing professional tools may be better.

What would a team stop paying for? A new subscription added to an unchanged tool stack has a weaker case than demonstrable time savings or replacement of a recurring expense.

## Questions that change the architecture

Can one public occurrence be authoritatively linked to multiple provider listings without duplicating attendance? Who can correct the public date, and who owns the consequences for orders? Does an artist's public co-host credit reflect real approval? What happens after an organizer leaves the team?

Which identity is operated by a group, how is control recovered, and who can revoke a former worker? Resolve these before promising decentralized organization ownership.

## Decision procedure

Use [[Subcult Decision Template]] for irreversible or cross-cutting choices. Record alternatives, contrary evidence, scope, reversibility and a revisit trigger. A green pilot metric should not override an unresolved privacy or financial correctness defect. Review alongside [[Subcult Risk Register]].

