---
type: metric-design
status: proposed
created: 2026-09-19
research_as_of: 2026-09-19
tags:
  - subcult-research
---

# Subcult Metrics and Measurement

This is a proposed measurement contract. No production baseline or actual business performance is reported.

## Three primary measures

| Metric | Definition and grain | Decision use |
| --- | --- | --- |
| Repeat workflow completion | Distinct pilot workspaces completing public event, assigned commitments and private closeout for two eligible event cycles / workspaces with two scheduled eligible cycles elapsed | Does the combined workflow retain utility beyond a demo? |
| Paid continuation | Distinct eligible pilot workspaces accepting and paying for the defined recurring offer / workspaces offered it after the pilot | Is there a real buyer, at this scope and price? |
| Contribution per active paid workspace | Subscription revenue less defined variable provider/infrastructure and support labor costs, aggregated monthly | Can the service operate without unsustainable hand-holding? |

Keep numerator/denominator lists auditable. Report raw counts alongside rates because a five-to-ten-team cohort is small. Teams without a second eligible event yet are immature, not retained or churned by assumption. Show cancellations separately rather than silently excluding difficult cases.

## Drivers

For repeat completion, inspect time to first correct shareable event and time from event end to accepted closeout. For paid continuation, record the budget holder and stated replacement/value rationale. For contribution, track support minutes per event and variable delivery costs.

Use actual event-relative timestamps in the event's timezone, stored unambiguously. For duration comparisons, distinguish active work time from elapsed calendar time. A faster closeout could reflect simpler events rather than better software.

## Guardrails

**Privacy/permission failures:** confirmed unauthorized messages or private-data disclosures, with severity and affected scope. Target zero; investigate every occurrence rather than hiding it in a percentage.

**Critical operational failures:** incorrect admission/payment state, inaccessible essential assignment information or unreconciled destructive changes. A single severe failure can block expansion even if average metrics improve.

## Initial thresholds

Proposed learning gate: five teams complete two cycles and three make paid commitments, with no unresolved severe guardrail incident. These are judgment-based pilot criteria, not industry benchmarks or statistical proof.

Do not set an artificial “20% time saved” promise without a comparable baseline. Collect paired workflow observations first and report variation and confounders.

## Instrumentation and ownership

Product lead owns the metric definitions; engineering owns event correctness; the pilot operator validates real completion; the commercial lead validates payment. Assign named people before launch.

Suggested private events: event_publication_confirmed, commitment_assigned, closeout_accepted, subscription_paid, delivery_suppressed and support_session_logged. These are proposals, not existing instrumentation.

Attribution should be labeled honestly. A tagged click followed by a purchase establishes the chosen attribution rule, not incremental causal revenue. Randomized tests may be inappropriate or underpowered in a small pilot; do not fabricate causal certainty.

Related: [[Subcult Experiment Backlog]], [[Subcult Release Qualification]], [[Subcult Pricing and Unit Economics]].

