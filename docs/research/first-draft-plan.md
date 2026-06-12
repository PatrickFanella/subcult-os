#ivate reliability notes
* internal organizer comments

---

# Data Model Sketch

A clean backend model might look like:

```text
Organization
Event
Person
ContactMethod
Role
EventParticipant
Task
Commitment
Shift
Venue
VendorProfile
Payment
Expense
Invoice
DoorRecord
Incident
AccessibilityProfile
Asset
ArchiveEntry
Note
```

The most important conceptual object is probably:

```text
Commitment
```

Because it bridges both original ideas.

A commitment could be labor, money, gear, access, promotion, performance, or follow-up.

---

# The Unique Angle

The app should not just ask:

> “How many tickets did we sell?”

It should ask:

> “Did everyone know what they were doing, did people get paid, did the space work, did we keep our promises, and what should we remember next time?”

That is the difference.

---

# Possible Tagline Directions

## Simple

* **Run the show. Remember the scene.**
* **Event ops for temporary spaces.**
* **A ledger for shows, shifts, payouts, and promises.**
* **Plan the night. Settle the door. Keep the archive.**

## More political / collective

* **Infrastructure for events without extraction.**
* **Tools for scenes that run on trust.**
* **Coordination software for collective culture.**
* **Keep the door, the ledger, and the memory.**

## More product-y

* **The backstage system for pop-up events.**
* **Manage the event before, during, and after the show.**
* **A shared ops board for DIY events, venues, and collectives.**

My favorite:

> **Run the show. Settle the door. Remember what happened.**

That captures the whole product.

---

# Recommended Final Shape

I would define it like this:

## **Load-In**

**Load-In is a pop-up event operations ledger for DIY venues, collectives, artists, vendors, and temporary cultural spaces. It helps organizers plan events, coordinate workers and volunteers, track promises, manage door money and payouts, document incidents and accessibility needs, and preserve a useful post-show archive.**

The app is built around a simple lifecycle:

```text
Plan → Coordinate → Run Door → Settle → Archive
```

And a simple principle:

```text
Every event creates obligations.
Every obligation should be visible, owned, and resolved.
```

That gives the product a strong spine.
can have:

* rates
* requirements
* past events
* notes
* preferred communication
* payment info
* availability
* reliability history
* incident history, permissioned carefully
* social links

This should feel more like a **collective address book** than a CRM.

---

## 7. Incident Notes & Safety Log

This is important, but it has to be handled carefully.

Incident notes should be:

* private by default
* permission-controlled
* exportable
* time-stamped
* not casually visible to all event collaborators
* able to be marked sensitive

Types:

```text
medical
harassment
accessibility issue
payment dispute
venue issue
police/security issue
equipment damage
conflict
lost item
weather/logistics
```

Do not build this like a punitive surveillance tool.

Build it as a **care, continuity, and accountability record**.

---

## 8. Accessibility Info

Accessibility should be first-class, not a notes field.

For venues/events:

* wheelchair access
* step-free entrance
* bathroom access
* seating availability
* lighting/strobe notes
* sound level
* masks/air filtration
* transit/parking
* interpreter availability
* sensory notes
* service animal policy
* contact person for access needs

This could become one of the most meaningful differentiators.

A lot of DIY events want to be accessible but do not have a consistent system for checking and communicating it.

---

## 9. Flyers, Socials, and Archive

Every event should produce an archive.

Archive fields:

* flyer image
* lineup
* final attendance
* vendors
* performers
* payout summary
* photos/videos
* social posts
* press links
* lessons learned
* incident summary, private
* accessibility notes for next time
* reusable template

This turns the app into a scene memory system.

That is much more interesting than just “event management.”

---

# Suggested MVP

The MVP should not try to replace Eventbrite, Square, Notion, and Instagram all at once.

The first useful version should be:

## MVP: **Plan, staff, settle, archive**

### Must-have features

1. **Create an event**
2. **Add contacts/roles**
3. **Create tasks/promises**
4. **Create shifts**
5. **Track door income manually**
6. **Track expenses/payouts manually**
7. **Generate settlement summary**
8. **Store flyer/social/archive links**
9. **Basic accessibility notes**
10. **Export event report as PDF/Markdown/CSV**

That is enough to be useful.

The app can start as an organizer tool, not a public ticketing platform.

---

# Version 2 Features

Once the core ledger works:

* guest list / RSVP
* mobile door check-in
* payment links
* invoice generation
* recurring event templates
* vendor applications
* volunteer signup forms
* public event pages
* performer/vendor portals
* post-event archive pages
* shared collective workspace
* granular permissions
* accounting exports
* calendar sync
* QR code door flow
* payout reminders
* reputation/history across events

---

# Version 3 Features

Later, it can become real infrastructure:

* multi-collective networks
* venue availability calendars
* shared contact pools
* mutual aid requests
* equipment lending ledger
* scene-wide archive
* cooperative revenue sharing
* decentralized/federated public event listings
* public accessibility database
* community moderation and dispute workflows
* grant/reporting tools for arts orgs

---

# Role System

You will want roles early.

Suggested roles:

| Role      | Permissions                           |
| --------- | ------------------------------------- |
| Owner     | Full control                          |
| Organizer | Manage event ops, people, money       |
| Treasurer | Manage settlement, payments, invoices |
| Door      | Guest list, check-in, door totals     |
| Volunteer | View assigned shifts/tasks            |
| Performer | View schedule, payout, requirements   |
| Vendor    | View setup info, fees, schedule       |
| Safety    | View/write incident notes             |
| Viewer    | Read-only access                      |

Important: not everyone should see everything.

Especially:

* incident notes
* money
* contact details
* pr **Load-In** may be the best product name. It is casual, scene-native, and not too precious.

---

# Main Modules

## 1. Events

The event is the main object.

Each event has:

* title
* date/time
* location
* public/private visibility
* organizer team
* flyer
* description
* age policy
* capacity
* ticket/door policy
* accessibility notes
* safety notes
* lineup/program
* vendor list
* shifts
* payments
* archive

An event can be a one-night show, a recurring night, a multi-day pop-up, or a small festival.

---

## 2. People & Contacts

This is where **Guild Ledger** folds in.

Instead of just having “attendees” or “users,” the app should have a **scene contact graph**.

Each person/contact can have:

* name
* role
* contact info
* preferred payment method
* skills
* availability
* relationship notes
* past events worked
* reliability notes, carefully permissioned
* accessibility needs
* rates
* links/socials
* tags

Example tags:

```text
sound
door
DJ
projection
security
flyer design
photography
vendor
bartender
stage manager
accessible transport
knows generators
has PA system
```

This becomes very valuable over time.

The app is not just managing this event. It is helping a scene remember who can do what.

---

## 3. Promises / Commitments

This is the heart of the merged idea.

A commitment is something someone has promised to do, provide, deliver, pay, or confirm.

Examples:

* “Maya will bring two folding tables.”
* “Luis will run door from 8–10.”
* “Venue will provide PA and two mics.”
* “Organizer will pay DJ $150 after close.”
* “Vendor owes $25 table fee.”
* “Sam will make flyer by Tuesday.”
* “Collective owes photographer credit in archive post.”
* “A ramp needs to be confirmed before publishing accessibility info.”

Each commitment should have:

```text
who
what
for which event
due date
status
notes
proof/attachment
related payment
visibility
```

Statuses:

```text
proposed
accepted
in progress
delivered
blocked
cancelled
disputed
settled
```

This turns vague group chat promises into a lightweight accountability ledger.

---

## 4. Scheduling & Shifts

This handles the operational side.

Useful shift types:

* door
* merch
* load-in
* load-out
* sound
* stage
* vendor setup
* cleanup
* safety/support
* accessibility support
* runner
* photographer/videographer
* social posting

A nice UX pattern would be:

```text
Event Timeline
5:00 PM — Load-in
6:00 PM — Vendor setup
7:00 PM — Doors
8:00 PM — First act
9:00 PM — Headliner
10:30 PM — Breakdown
11:00 PM — Settlement
```

Each block can have:

* assigned people
* notes
* required gear
* payout
* task checklist
* private organizer notes

---

## 5. Door, Payments, and Settlement

This is where the app becomes immediately useful.

For small events, settlement is always annoying.

The app should track:

* cash collected
* card/mobile payments
* comped entries
* guest list
* vendor fees
* performer guarantees
* percentage splits
* venue cut
* expenses
* donations
* remaining balance
* who has been paid

Example settlement view:

| Item                      | Amount |
| ------------------------- | -----: |
| Door cash                 |   $420 |
| Mobile payments           |   $260 |
| Vendor table fees         |   $100 |
| Total in                  |   $780 |
| Venue split               |  -$150 |
| Sound person              |  -$100 |
| DJ 1                      |  -$150 |
| DJ 2                      |  -$150 |
| Flyer printing            |   -$40 |
| Remaining collective fund |   $190 |

The killer feature is not payment processing itself at first.

The killer feature is **settlement clarity**.

You can integrate payments later.

---

## 6. Vendor / Venue / Skilled Person Directory

This becomes a local infrastructure memory bank.

Contacts can be grouped by type:

* venues
* pop-up spaces
* artists
* performers
* vendors
* food vendors
* tech people
* door people
* designers
* photographers
* security/safety
* accessibility support
* mutual aid groups
* gear owners
* printers
* promoters

Each contact # Merged Concept: **Pop-Up Event Manager / Collective Ledger**

This should become **one app**, not two.

The clean version is:

> **A non-extractive operating system for pop-up events, DIY venues, mutual aid shows, collectives, and temporary cultural spaces.**
> It helps organizers coordinate people, money, promises, labor, safety, vendors, accessibility, and post-event archives without turning the scene into a corporate ticketing funnel.

The key insight is that **events are where collective coordination becomes concrete**.

A guild ledger tracks:

* who can help
* who needs help
* what was promised
* what was delivered
* what is owed
* what happened

A venue/event manager tracks:

* who is playing/working/vendor-ing
* who is scheduled
* who got paid
* who showed up
* who needs access/support
* what incidents happened
* what should be remembered next time

Those are the same system viewed from different angles.

---

# Working Product Shape

## Core Idea

The app manages a pop-up event from **planning → staffing → money → showtime → settlement → archive**.

Each event becomes a living record:

```text
Event
├── People
│   ├── organizers
│   ├── performers
│   ├── vendors
│   ├── volunteers
│   ├── door staff
│   ├── tech/sound
│   └── venue contacts
├── Promises
│   ├── who said they would do what
│   ├── when it is due
│   ├── whether it was delivered
│   └── what is still unresolved
├── Money
│   ├── door income
│   ├── vendor fees
│   ├── performer payouts
│   ├── venue split
│   ├── invoices
│   └── expenses
├── Operations
│   ├── schedule
│   ├── shifts
│   ├── load-in/load-out
│   ├── accessibility notes
│   ├── incident notes
│   └── task assignments
└── Archive
    ├── flyers
    ├── social links
    ├── photos/video links
    ├── attendance
    ├── payouts
    ├── notes
    └── what to improve next time
```

---

# Better Framing

I would not frame it as a “venue app” first.

I would frame it as:

## **An event ops ledger for temporary culture**

Because the strongest market is not just venues. It is:

* DIY shows
* art pop-ups
* basement shows
* warehouse parties
* zine fairs
* mutual aid events
* small festivals
* comedy nights
* workshops
* community markets
* house shows
* experimental theater
* skillshares
* local music scenes
* underground dance events
* collectives without formal admin infrastructure

That gives it more range.

---

# Product Thesis

Most event tools assume the event is a commercial product.

This app assumes the event is a **temporary collective**.

That difference matters.

Eventbrite, Square, Google Sheets, Venmo, Notion, Airtable, and Instagram all solve pieces of this. But DIY organizers usually end up with a messy stack:

```text
Instagram flyer
+ Google Form
+ Cash App/Venmo
+ spreadsheet
+ group chat
+ notes app
+ random DMs
+ someone’s memory
+ regret
```

Your app would replace that mess with one lightweight system.

Not corporate. Not extractive. Not bloated.

---

# Possible Names

A few directions:

## Practical / Clear

* **Doorlist**
* **Load-In**
* **Showbook**
* **Event Ledger**
* **Pop-Up Ledger**
* **Venue Kit**
* **Night Sheet**
* **SettleUp**
* **Backline**

## More Subcult / DIY

* **Forked Venue**
* **House Ledger**
* **Scene Sheet**
* **The Door**
* **Signal Booth**
* **Common Door**
* **Handbill**
* **Greenroom**
* **Switchyard**
* **Patchbay**

## Strongest candidates

My favorites for this concept:

1. **Load-In**
   Very event-native. Implies setup, labor, logistics, and scene work.

2. **Doorlist**
   Extremely clear. Starts narrow but could expand.

3. **Patchbay**
   Nice metaphor: routing people, tasks, venues, money, and signals.

4. **Common Door**
   Good if you want the non-extractive/collective angle front and center.

5. **Showbook**
   Simple, memorable, practical.

I think
