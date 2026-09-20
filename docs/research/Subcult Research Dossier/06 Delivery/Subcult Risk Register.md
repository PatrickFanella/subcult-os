---
type: risk-register
status: proposed
created: 2026-09-19
research_as_of: 2026-09-19
tags:
  - subcult-research
---

# Subcult Risk Register

Likelihood is not quantified because no operating baseline exists. Priorities below reflect consequence and architectural relevance, not a measured risk score.

| Risk | Early signal | Mitigation / stop condition |
| --- | --- | --- |
| Too broad to adopt | Teams use one screen and abandon the rest | Narrow to the repeated paid job |
| Weak willingness to pay | Enthusiasm without budget-holder commitment | Paid scoped pilot; no inflated TAM |
| Support overwhelms price | Founder manually resolves every event | Track labor, reduce scope or reprice |
| Private data leaks publicly | Unsafe fields reach a publish payload | Separate storage, validation, review; stop publication |
| Unwanted contact | Imported attendees become marketing recipients | Scoped evidence, suppression and pre-send checks |
| Financial/admission errors | Provider and local states disagree | Authoritative provider, reconciliation, fallback |
| Duplicate/stale occurrences | Multiple namespace copies drift | Canonical mapping and independent reader tests |
| Organization takeover/access drift | Public credits imply permissions | Explicit private roles and recovery procedure |
| Provider/API dependency | Required access unavailable or terms change | Link/manual fallback; no unauthorized scraping |
| Protocol experimentation blocks product | Core workflow waits for Spaces | Private baseline and bounded research budget |
| False release confidence | Historical test pass treated as current | Evidence by revision/environment and explicit gates |
| Network cold start | Empty maps without operational value | Single-team utility and regional concentration |
| Community distrust | Opaque scoring or monetized private relationships | Transparent controls, no universal reputation |
| Regulatory mismatch | Messaging/checkout rules copied across regions | Jurisdiction-specific review before launch |

## Escalation

Severe privacy, payment or admission failures pause the affected workflow immediately. Product usage or revenue targets do not override that stop condition.

Business risks require a different response: gather evidence, narrow scope or revise the offer. Lack of immediate network growth is not justification for scraping personal data or automatically subscribing attendees.

## Operational ownership

Before pilot, assign named owners for security/privacy, provider delivery, event-night support, billing and incident communication. “The team” is not a sufficient escalation destination at doors-open.

Keep a private incident record separate from this shareable research pack. Public postmortems require a deliberate review for personal and venue-sensitive information.

## Review rhythm

Review open risks before each pilot event and after each closeout. Revisit the business risks after two cycles rather than using initial enthusiasm as a long-term assumption. Update the decision register when a mitigation materially changes product scope.

Related: [[Subcult Decisions and Open Questions]], [[Subcult Release Qualification]], [[Subcult Pricing and Unit Economics]].

