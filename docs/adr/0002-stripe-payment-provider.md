# ADR 0002: Assume Stripe for Payments

`subcult-os` will assume Stripe for ticket checkout and future Stripe Connect payout integration because built-in ticketing is part of the product direction and Stripe gives a credible path from checkout to later connected-account payouts. v1 should track Payouts manually while preserving a model that can support Connect later; domain language remains provider-neutral: Payments, Payouts, Refunds, Processing Fees, Fee Policy, Net Proceeds, and Settlement should describe the money model without making every concept Stripe-specific.
