---
type: metric-design
status: proposed
created: 2026-09-19
research_as_of: 2026-10-05
tags:
  - subcult-research
---

# Subcult metrics and measurement

This is a proposed measurement contract for METRICS-01, issue #37, revised on 2026-10-05. No production baseline or actual business performance is reported. It defines a manual pilot worksheet, not permission to add analytics, export operational records or collect participant data.

RESEARCH-01, issue #34, remains the prerequisite for choosing the buyer, territory, recurring job and supported ticket/provider workflow. These definitions can be reviewed now. Do not run a real experiment until that research decision, named owners and the separate pilot authorization are recorded. Definitions do not satisfy #34, owner headline decision #201 or physical-device qualification #6.

## Organizer activation

Activation is a distinct eligible workspace completing its first real organizer event cycle within the observation window. Before enrollment, the organizer and pilot operator agree which jobs apply to its selected workflow and what proves completion:

- Prepare a correct event and confirm the intended shareable listing. A draft or publication attempt is insufficient. AT publication is required only when the approved workflow actually uses it and its separate gates pass.
- Assign and resolve the agreed event commitments and staffing jobs. An assignment alone is insufficient. The organizer confirms done, or records a cancellation and its reason category.
- Run the agreed event-day workflow and accept private closeout, including the status of any applicable finance obligations. A saved finance row or archive entry alone is insufficient. Outstanding obligations must remain visible.

A cancellation does not count as a completed event cycle. A workflow may mark a job not applicable only before outcomes are known, with a reason. Missing evidence is unknown, never assumed complete. Count each workspace once, after all applicable jobs pass organizer confirmation and operator review.

Activation rate = activated workspaces / eligible enrolled workspaces whose first-cycle observation deadline has elapsed. Report the numerator and denominator counts, assisted and unassisted completions separately, and counts still awaiting their deadline. Founder demos, synthetic tests, staff-only rehearsals, account creation and forced signup do not count. Assistance can count when the organizer performs the job, but founder substitution cannot.

## Three primary measures

| Metric | Definition and grain | Decision use |
| --- | --- | --- |
| Repeat workflow completion | Distinct pilot workspaces completing the agreed organizer jobs and private closeout for two eligible event cycles / workspaces with two scheduled eligible cycles elapsed | Does the combined workflow retain utility beyond a demo? |
| Paid continuation | Distinct eligible pilot workspaces accepting and paying for the defined recurring offer / workspaces offered it after the pilot | Is there a real buyer, at this scope and price? |
| Contribution per active paid workspace | Subscription revenue less defined variable provider/infrastructure and support labor costs, aggregated monthly | Can the service operate without unsustainable hand-holding? |

Keep numerator/denominator lists auditable. Report raw counts alongside rates because a five-to-ten-team cohort is small. Teams without a second eligible event yet are immature, not retained or churned by assumption. Show cancellations separately rather than silently excluding difficult cases.

## Drivers

For repeat completion, inspect time to first correct shareable event and time from event end to accepted closeout. For paid continuation, keep budget-holder verification and the stated replacement/value rationale in restricted research evidence. The measurement report uses only a verified-buyer flag and bounded rationale category. For contribution, track support minutes per event and variable delivery costs.

Use authorized workflow observations to calculate durations with unambiguous timestamps and the event's timezone. Keep exact timestamps in restricted evidence; reports use durations and cohort periods. For duration comparisons, distinguish active work time from elapsed calendar time. A faster closeout could reflect simpler events rather than better software.

## Guardrails

Privacy/permission failures include confirmed unauthorized messages or private-data disclosures, with severity and affected scope. Target zero; investigate every occurrence rather than hiding it in a percentage.

Critical operational failures include incorrect admission/payment state, inaccessible essential assignment information or unreconciled destructive changes. A single severe failure can block expansion even if average metrics improve.

## Initial thresholds

Proposed learning gate: five teams complete two cycles and three confirm paid continuation, with no unresolved severe guardrail incident. These are judgment-based pilot criteria, not industry benchmarks or statistical proof.

Do not set an artificial "20% time saved" promise without a comparable baseline. Collect paired workflow observations first and report variation and confounders.

## Instrumentation and ownership

Product lead owns the metric definitions; engineering owns event correctness; the pilot operator validates real completion; the commercial lead validates payment. Assign named people before launch.

Possible future signals include event_publication_confirmed, commitment_completed, closeout_accepted, subscription_paid, delivery_suppressed and support_session_logged. No such analytics instrumentation is introduced here. Existing private commitments, finance and archive APIs contain operational content and are not telemetry payloads. An event ticket payment is not an OS subscription payment; require separately verified commercial evidence.

Attribution should be labeled honestly. A tagged click followed by a purchase establishes the chosen attribution rule, not incremental causal revenue. Randomized tests may be inappropriate or underpowered in a small pilot; do not fabricate causal certainty.

## Counting and cost rules

Freeze the cohort and each workspace's eligible event cycles before the experiment. Use an experiment-local random workspace code and cycle code to deduplicate manual observations. Keep any identity mapping separately with restricted access. Retries, edits, duplicated observations and rescheduled versions of the same cycle do not create new completions.

- Repeat completion uses distinct workspaces with two completed eligible cycles as its numerator. Its denominator is every enrolled workspace whose two predeclared cycle deadlines have elapsed. Report immature cohorts separately. Keep cancellations and withdrawals after enrollment in the matured denominator, with their status counts. Do not choose only successful first events for the denominator. A missed deadline is incomplete; a missing observation is unknown and cannot pass the gate.
- Paid continuation uses distinct workspaces with accepted terms and confirmed, non-refunded payment for the defined post-pilot recurring offer as its numerator. The denominator is all eligible workspaces offered the same scope and price whose predeclared payment decision deadline elapsed. Report eligible-but-not-offered workspaces, pending deadlines, declines, refunds, discounts and service arrangements separately. A promise, invoice, free trial, founder payment or ticket sale is insufficient. Show price cohorts separately; never pool incompatible offers to imply willingness to pay.
- Support burden is all pilot support labor minutes / all elapsed eligible event cycles, including failed and cancelled cycles. Also report total minutes, number of supported cycles and minutes per active workspace-month. Include founder, volunteer, onboarding, event-night, recovery and reconciliation work, using separate categories. Separate one-time onboarding from recurring support without dropping it from total burden. Record active work time separately from waiting time, and do not treat unlogged labor as zero.
- Variable delivery cost is actual attributable provider charges and usage-dependent infrastructure cost for the cohort period, including accepted, suppressed-if-charged, failed and retried delivery attempts. Report total and cost per elapsed eligible cycle and active paid workspace-month. Separate provider acceptance from delivery. Record quantity, unit rate, currency, invoice period and allocation rule, with provider evidence restricted. Fixed platform overhead is separate. Unknown charges stay unknown; no production margin claim until reconciled.
- Monthly contribution = net OS subscription revenue attributable to that month, less refunds, discounts and applicable payment fees, less variable delivery/infrastructure cost, less support hours multiplied by the predeclared labor rate. Allocate annual receipts to their service period. Exclude ticket proceeds, pass-through money and taxes from OS subscription revenue. Count distinct active paid workspaces in that month, including zero-event workspaces, and define active paid service dates before the period. Contribution per active paid workspace = monthly contribution / that count. Use one currency or a documented conversion source/date. Avoid double-counting fees or support. This is contribution before fixed costs, not accounting gross margin.

For every rate, a zero denominator is not applicable, not zero percent. Unknown cost or missing labor prevents a positive-contribution conclusion. Keep exclusion counts and reason categories auditable without putting identity lists in the shareable report.

## Baseline and decision rules

Use [[Subcult Experiment Template]] before any E1 through E10 experiment. Freeze the following in its dated, versioned design before observing results:

- The #34 research decision, authorized population, jobs, cycle eligibility, numerator, denominator, deduplication rule and exclusions. State which measures apply and why others do not.
- A comparable recent-event baseline with its observation dates, source permission, workflow complexity, event cadence and support level. If no baseline exists, mark it unavailable and run a baseline-only study. Do not claim improvement or time saved.
- Enrollment cutoff, cycle completion and closeout deadlines, payment decision window, timezone, minimum mature cohort, missing-data treatment and review date. Record active-work and elapsed-time baselines separately.
- The success threshold, maximum support minutes, maximum variable cost and minimum acceptable contribution, with units and rationale. Existing five-team/two-cycle/three-paid thresholds remain proposals. The owner must choose limits before a run; unset limits block launch.
- Safety stops and who can pause the experiment. Pause affected real use immediately for suspected unauthorized messaging, private disclosure, wrong admission/payment state or loss of essential assignment access. Preserve restricted evidence and invoke the existing incident process. An unresolved severe incident blocks expansion regardless of averages.
- Cost and feasibility stops. Pause further enrollment or delivery when the predeclared support/cost ceiling is reached, required evidence or permissions fail, or a prerequisite no longer holds. At the fixed review date, stop or revise if recurrence, paid continuation or contribution misses its threshold. Insufficient maturity or incomplete evidence is inconclusive, never a pass.

Do not stop early merely because a small sample looks favorable. Report raw counts with rates, uncertainty and confounders. Preserve the original rule and failed attempts. Any amendment gets a new dated revision, its reason and a separate result interpretation; it cannot retroactively turn a failed run into a pass. A stop does not authorize refunds, retained-data deletion, provider changes or a release.

## Privacy-safe worksheet

Only after separate collection approval, use an access-controlled manual worksheet with these allowlisted columns. This document and the blank template collect nothing.

| Record | Allowed measurement fields |
| --- | --- |
| Experiment design | ID, revision, named internal owner, research decision reference, permission reference, period, job/checklist version, formulas, deadlines, limits and retention/access plan |
| Workspace cycle | Experiment-local workspace/cycle code, eligibility status, bounded exclusion/cancellation reason category, applicable job completion flags, assisted flag, duration totals, deadline maturity and evidence-verification status |
| Commercial period | Experiment-local workspace code, offer scope/price cohort, currency, offer/payment deadline maturity, verified paid/refunded status, attributable revenue and aggregate fees |
| Support/cost period | Cohort period, bounded labor category, minutes/hours, rate/currency, aggregate usage/charges, allocation rule and completeness status |
| Review | Raw counts, denominator status counts, aggregate costs, guardrail category/severity/count, decision, uncertainty and next review date |

Keep operational evidence references and identity mappings in a separate restricted evidence index. Never copy contact names, email addresses, phone numbers, message bodies, consent receipts, ticket codes, private notes, exact personal location, protected venue coordinates, device identifiers or credential-bearing URLs into measurement records. Do not hash contact content as an identifier. Do not infer individual attendance histories, cross-event participant profiles or opaque fan scores.

Reports use cohort counts and cost/duration summaries. Do not expose workspace/cycle rows or small subgroup breakdowns outside authorized reviewers. Before collection, the owner must set a minimum reportable group size, access roles, review/retention dates and an approved handling plan for withdrawal. If a breakdown is below that threshold, suppress it and avoid complementary totals that reveal it. Pseudonymous codes are still private. No public AT record, third-party analytics export or new SDK is authorized by this contract.

## Revision evidence and limits

On 2026-10-05, #37 and #34 issue bodies/comments, main at `180e9071f11511c657f56605455aad376762a7c4`, the experiment backlog/template, pricing scenarios and current delivery plans were reviewed. #34 supplied research acceptance criteria, not a completed buyer/pilot decision. This revision specifies definitions and a blank recording instrument. It reports no interviews, pilot completions, payments, adoption or measured costs. Software validation does not qualify a device, live provider, intended user or release.

Related: [[Subcult Experiment Backlog]], [[Subcult Release Qualification]], [[Subcult Pricing and Unit Economics]].

