# subcult-os

subcult-os describes the domain language for coordinating temporary cultural, social, and collective occasions without treating them as purely commercial products.

## Language

### Event lifecycle and product framing

**Event**:
A temporary cultural, social, or collective occasion that requires coordination of people, space, money, commitments, and memory. An Event may be a show, market, workshop, party, pop-up, benefit, skillshare, or small festival.
_Avoid_: Show as the generic term, gig, activation, campaign

**Event Status**:
The private operational lifecycle state of an Event, such as Draft, Planning, Active, End of Night, Closing, Closed, Reopened, Cancelled, or Rescheduled. Event Status is separate from Public Event Phase and Publish Status; cancelled and rescheduled Events remain records for refunds, communication, settlement, follow-up, and archive purposes.
_Avoid_: Using public page state as the operational lifecycle

**First Event Status Cut**:
The first v1 lifecycle supports only Draft, Published, and End of Night for the first tracer bullet. Cancellation, rescheduling, reopening, refunds, validity changes, and related notifications are later lifecycle capabilities.
_Avoid_: Adding cancellation and reschedule edge cases before the basic lifecycle is proven

**First Deletion Cut**:
The first v1 slice does not include deleting Events. Draft Events may remain in the Workspace, but deletion, hiding, retention cleanup, and removal rules are later capabilities because Tickets, Audit Trail, and Event Reports make Event deletion consequential.
_Avoid_: Adding destructive Event actions before lifecycle records have clear retention rules

**Rescheduled Event**:
An Event whose originally planned date, time, or occurrence has been moved rather than cancelled outright. A Rescheduled Event should preserve the original Event record and make the new timing clear for Tickets, Orders, Notifications, Public Event Page state, and Event Reports.
_Avoid_: Deleting and recreating the Event when continuity matters

**Reschedule Option**:
The attendee-facing or Organizer-recorded outcome for an Order when an Event is rescheduled, such as keeping Tickets, requesting a refund, receiving credit, transferring Tickets, or remaining unresolved. Tickets carry over by default unless a different Reschedule Option is chosen or recorded.
_Avoid_: Assuming rescheduled Tickets are either all valid or all refunded

**Cancellation Option**:
The attendee-facing or Organizer-recorded outcome for an Order when an Event is cancelled, such as refund, credit, donation, transfer, no refund under policy, or manual resolution.
_Avoid_: Assuming cancellation always means automatic refund

**Refund**:
Money returned or expected to be returned for a Payment or Order, either fully or partially. A Refund may result from cancellation, rescheduling, policy exception, duplicate purchase, dispute, or manual correction.
_Avoid_: Treating Refund as only an Event cancellation outcome

**Processing Fee**:
A fee charged by a payment processor for handling a Payment or Refund.
_Avoid_: Treating processor fees as Event revenue

**Platform Fee**:
A fee charged by subcult-os for use of the platform or transaction flow.
_Avoid_: Hiding platform revenue inside ticket price

**Net Proceeds**:
The amount remaining after relevant Processing Fees, Platform Fees, Refunds, and other deductions.
_Avoid_: Calling gross ticket sales the money available for Settlement

**Fee Policy**:
The rule for whether Processing Fees, Platform Fees, or other Ticket-related fees are passed to the Attendee, absorbed by the Host Workspace, or split. Fee Policy may vary by Event or Ticket Type.
_Avoid_: Assuming every Event handles fees the same way

### Event operations record

**Event Operations Record**:
The shared record of an Event's commitments, roles, shifts, money, incidents, access needs, and archive. It exists to make obligations visible, owned, and resolved before, during, and after the Event.
_Avoid_: Event management platform, CRM, ticketing platform, project management tool

**Operational Instruction**:
A shareable action or constraint needed to run an Event, separated from any sensitive details that explain why. Operational Instructions may come from Access Needs, Safety Notes, Entry Policy, Venue requirements, Assignments, or Settlement; they can become or attach to Assignables when someone must act on them.
_Avoid_: Exposing private context when only the action is needed

**Operational Note**:
A flexible note attached to an Event, Contact, Assignment, Settlement, Door Record, or Archive when structured fields are not enough. Operational Notes should not be used for Safety Notes, Access Needs, or money adjustments that need first-class handling.
_Avoid_: Using notes as a substitute for important structured records

**Event Orchestrator**:
The product framing for subcult-os as a tool that helps people coordinate Event operations. In the domain language, prefer Event Operations Record when referring to the shared source of truth rather than the product itself.
_Avoid_: Event Operations Manager

**Public Event Page**:
A public-facing page for an Event where attendees can view essential information and, when enabled, reserve or buy Tickets. The same Public Event Page may display ticketing before the Event and public Archive material after End of Night; it is not the same as the private Event Operations Record.
_Avoid_: Treating the public page as the operational workspace

**First Publication Cut**:
The first v1 Public Event Page requires only title, date and time, public description, location display text, Ticket Allocation, and Host Workspace Display Name. Flyer image, Venue object, full address release rules, Lineup, Age Policy, Tags, Series, and custom URL are later publication capabilities.
_Avoid_: Blocking first publication on rich event marketing or venue modeling

**First Published Edit Cut**:
The first v1 slice allows limited edits after publication: public description, location display text, date and time only when no Tickets have been reserved, and Ticket Allocation only upward or otherwise not below the reserved Ticket count. Cancellation, rescheduling, attendee notifications for major changes, and full revision history are later capabilities.
_Avoid_: Treating publication as either completely immutable or fully editable without attendee impact rules

**Event Discovery**:
Public browsing or search of published Public Event Pages by date, area, Tag, Series, Public Profile, Venue, or ticket and RSVP availability. Event Discovery excludes hidden, draft, private, and unlisted Events unless explicitly shared.
_Avoid_: Algorithmic feed or recommendation engine as v1 scope

**First Discovery Cut**:
The first v1 published Event is reachable by direct link only rather than listed in a public discovery index. Public browsing, search, moderation, geography filters, and discovery UX are later capabilities.
_Avoid_: Treating first publication as requiring a public event directory

**Public Event Phase**:
The public-facing phase of an Event page, such as Upcoming, Live, Ended, or Archived. Public Event Phase controls how the Public Event Page presents ticketing, live/event-day information, and public Archive material across the Event lifecycle.
_Avoid_: Creating unrelated public pages for the same Event lifecycle

**First Tracer Bullet**:
The first end-to-end v1 slice used to prove the Event lifecycle with the least payment complexity: Create Event, publish Public Event Page, reserve a free Ticket, check in at Door, run End of Night, and generate an Event Report.
_Avoid_: Starting v1 with paid Stripe checkout before the lifecycle path is proven

**First Acceptance Test Cut**:
The first v1 implementation plan should include tracer-bullet acceptance tests for the full lifecycle path: signup or login, create Workspace, invite Member, create and publish Event, guest reserves a free Ticket, Member checks in Ticket, Owner runs End of Night, and Event Report shows reserved, checked-in, and no-show counts.
_Avoid_: Verifying only isolated screens or handlers without proving the Event lifecycle works end to end

**First Event List Cut**:
The first v1 Host Workspace interface includes a minimal Event list showing Draft, Published, and End of Night Events with create and open actions. Calendar views, filters, search, archive browsing, and advanced event organization are later capabilities.
_Avoid_: Forcing Members to navigate by direct Event URL only inside the Workspace

### Identity, profiles, tags, and reputation

**Public Profile**:
A public-facing identity page for a Person, Collective, Workspace, or Event participant role that can show selected public information, Events, links, media, and contact paths. A Public Profile does not imply access to private Workspace records.
_Avoid_: Public account when referring to public presence rather than login/access

**First Public Profile Cut**:
The first v1 tracer bullet does not include public Host Workspace profiles. The Public Event Page displays the Host Workspace Display Name only. Public profile pages, bios, links, avatars, follows, and organizer archive pages are later capabilities.
_Avoid_: Blocking first Event publication on public identity/profile infrastructure

**Profile Claim**:
A request or action that links a manual or imported Contact reference to a real Person, Workspace, or Public Profile. Profile Claim can improve identity accuracy but requires approval from the Workspace that owns the Contact link, and does not transfer Workspace-owned private Contact notes or history to the claimed profile.
_Avoid_: Treating Contact data as owned by the claimed person or Workspace

**Public Identity**:
The public-facing identity represented by a Public Profile, either for a Person or a Workspace. Event Roles may display a selected Public Identity when published.
_Avoid_: Assuming every Public Profile is tied to a login account

**Display Name**:
The name shown publicly or contextually for a Person, Workspace, Event Role, Credit, Ticket Holder, or Public Profile. Display Name may differ from legal, payment, account, or private Contact names.
_Avoid_: Assuming one name is correct in every context

**Legal Name**:
A sensitive name used only where required for payment, tax, Agreement, identity, refund, or compliance workflows. Legal Name should follow Need-to-Know and should not be used as the default public or contextual name.
_Avoid_: Using Legal Name as the default Display Name

**Pronouns**:
Optional self-described pronouns for a Person, shown only in contexts and visibility settings the Person chooses. Pronouns may appear on Public Profiles, Event Roles, Credits, or internal coordination views when allowed.
_Avoid_: Requiring pronouns or exposing them without consent

**Bio**:
Public or profile-facing descriptive text for a Person, Workspace, Public Profile, Event Role, or Public Identity. Bio should be controlled by the represented person or Workspace where possible and should not be used for private organizer notes.
_Avoid_: Mixing public profile copy with private Contact notes

**Social Link**:
A profile or contact link such as website, Instagram, Bandcamp, SoundCloud, Linktree, TikTok, newsletter, shop, or other public presence. Social Links may appear on Public Profiles, Event Roles, Credits, Applications, or Contacts depending on visibility.
_Avoid_: Burying public links in generic notes

**Tag**:
A lightweight label used to organize, filter, or discover records such as Contacts, Events, Public Profiles, Assets, Event Templates, or Event Series. Tags may be Workspace-owned or public-facing depending on context.
_Avoid_: Treating Tags as formal roles, permissions, or verified claims

**Tag Visibility**:
Whether a Tag is private to a Workspace, visible to selected Members, or public-facing on Public Profiles, Public Event Pages, Series Pages, or Archives.
_Avoid_: Assuming all Tags are safe to publish

**Sensitive Tag**:
A private evaluative Tag that could affect someone's reputation, access, booking, payment, or safety context, such as late payer, no-showed, do not book, or safety-adjacent labels. Sensitive Tags should follow Need-to-Know and be supported by context such as Safety Notes, Disputes, Lessons Learned, or Operational Notes where appropriate.
_Avoid_: Casual punitive labeling without context or visibility controls

**Reliability Context**:
Factual, permission-controlled history that helps a Workspace understand whether a Person, Workspace, or Contact has followed through before. Reliability Context should be Workspace-private by default, not shared through Connections, and derived from records such as Commitments, Assignments, Disputes, Lessons Learned, and Sensitive Tags rather than reduced to a universal score.
_Avoid_: Global reputation scores or contextless reliability ratings

**Recommendation**:
A deliberate, manually shared endorsement or reference about a Person, Workspace, Asset, Venue, or Contact. A Recommendation is not a global score and should not expose private Reliability Context unless explicitly included by the author.
_Avoid_: Automatic reputation sharing through Connections

**Recommendation Visibility**:
Whether a Recommendation is public, visible only through Connections, shared directly with selected recipients, or kept internal to a Workspace. Recommendation Visibility controls who can see the endorsement and whether it appears on public-facing profiles; public Recommendations require approval from the recommended Person or Workspace before appearing on their Public Profile.
_Avoid_: Assuming every Recommendation is public or automatically shared with all Connections

**Publish Status**:
Whether a Public Event Page, Public Profile, Archive item, or other public-facing record is hidden, draft, published, or unlisted. Publish Status is separate from Workspace access and Permissions.
_Avoid_: Assuming records are public because they exist

### Workspaces, contacts, and connections

**Organizer**:
A person responsible for making an Event happen by coordinating people, space, money, commitments, and follow-up. An Organizer may act alone or on behalf of a Collective.
_Avoid_: Manager, admin, promoter as the generic term

**Crew**:
People assigned to operational work for an Event, including paid workers, volunteers, and helpers. Crew is the broad labor relationship; specific jobs such as Door, Sound, Safety, Runner, or Load-In are specialties or Shift subtypes rather than primary Event Roles.
_Avoid_: Staff as the generic term, employees/volunteers as the repeated umbrella

**Workspace**:
The private account space where Members coordinate Events, contacts, records, profiles, and archives. A Workspace may represent one person, a Collective, a Venue, a Merchant business, a Vendor business, a Performer group, or another operating group.
_Avoid_: Organization as the user-facing term, account, tenant

**Host Workspace**:
The single Workspace that owns and coordinates an Event Operations Record. Other Workspaces may participate in the Event as Performers, Vendors, Merchants, Exhibitors, Venues, or partners without owning the Event.
_Avoid_: Multiple Host Workspaces for one Event, assuming every Workspace that appears in an Event can manage the Event

**Co-Host**:
A Event Role for a Workspace credited as helping produce or present an Event without being the Host Workspace that owns the Event Operations Record. Co-Hosts may receive Permissions or Payouts, but ownership remains with the Host Workspace.
_Avoid_: Multiple Event owners in v1

**Collective**:
A real-world group of people who organize or support Events together. A Collective may own or participate in a Workspace, but not every Workspace is a Collective.
_Avoid_: Organization when the group is informal or scene-based

**Person**:
A human represented in subcult-os, whether or not they can log in.
_Avoid_: User as the generic human term

**Contact**:
A Workspace-owned bookmark or address-book record for a Person, Workspace, or manually entered entity known for possible Event participation, communication, payment, resource use, or archive reference. If no mutual Connection exists, the Contact only exposes manually saved details and the other party's Public Profile.
_Avoid_: Lead, customer, CRM record, treating Contact as the identity itself, assuming a Contact is mutual

**First Contact Cut**:
The first v1 tracer bullet does not include Contacts. Members are invited by email, attendees reserve Tickets by email, and Door uses Ticket lookup. Contact management, recurring participant records, performer/vendor address books, and relationship history are later capabilities.
_Avoid_: Blocking the first Event lifecycle on CRM-style relationship modeling

**Connection**:
A mutual relationship between two Workspaces that allows selected profile, contact, messaging, and collaboration information to be shared more reliably than a one-sided Contact record. A Connection is closer to mutual friends than a bookmark.
_Avoid_: Requiring mutual acceptance before a Workspace can record a Contact

**Shared Field**:
A specific profile, contact, payment, requirement, or collaboration detail that a Person or Workspace chooses to share through a Connection. Shared Fields are more trusted than public profile data but should remain explicitly controlled.
_Avoid_: Treating Connection as full contact-data access

### Communication and notifications

**Contact Action**:
An app-assisted action that helps a Member communicate outside subcult-os, such as calling, texting, emailing, copying contact details, or starting a group text for selected Event people. Contact Actions may be listed on a Workspace or Event for reuse, but they do not create in-app message threads or store external message history.
_Avoid_: Messaging or chat for v1

**Communication Group**:
A saved set of recipients and purpose used to repeat Contact Actions from a Workspace or Event, such as reopening an SMS composer with the same contacts, launching an email flow, copying contact details, or starting a call. A Communication Group stores who to contact, not the contents or history of external messages.
_Avoid_: Treating Communication Groups as in-app chats or guaranteed links to the same external SMS/email thread

**Communication Activity**:
A lightweight record that a Contact Action was launched, including when, by whom, which Communication Group or recipients, and what channel was used. Communication Activity does not store message contents or external replies, and should follow Need-to-Know visibility because recipient sets and purpose can reveal sensitive context.
_Avoid_: Message history, chat log

**Contact Preference**:
A Person or Contact's preferred and restricted communication channels, such as phone, SMS, email, social link, do-not-text, or operational-only. Contact Preferences guide Contact Actions but do not replace legal consent requirements for platform-sent marketing.
_Avoid_: Treating saved contact info as permission to contact for any purpose

**Operational Communication**:
Communication needed to coordinate, attend, work, settle, or follow up on a specific Event or Workspace relationship.
_Avoid_: Treating operational messages as promotional outreach

**Marketing Communication**:
Promotional or fundraising communication not required for a specific Event obligation, Ticket, Assignment, Payment, or Payout.
_Avoid_: Sending marketing through operational Contact Actions without consent

**Notification Preference**:
A Person's opt-in or opt-out settings for notifications sent by subcult-os, including channels, topics, and frequency. Notification Preferences are separate from Contact Preferences used for human-initiated Contact Actions; v1 notifications should be operational rather than marketing-oriented.
_Avoid_: Treating app notifications and organizer contact preferences as the same thing

**Notification**:
A system-generated update from subcult-os about an Event, Assignment, Ticket, Order, Payment, Payout, Safety Note, Access Need, or other operational change. Notifications are governed by Notification Preferences.
_Avoid_: Using Notification for human-initiated Contact Actions

**Transactional Notification**:
A required or expected Notification tied to a specific Order, Ticket, Payment, Refund, or Guest Checkout flow, such as receipt, ticket delivery, cancellation, rescheduling, or refund status. Essential Transactional Notifications cannot be disabled, while nonessential reminders and updates may follow Notification Preferences; Transactional Notifications do not imply Follow, Event Watch, or Marketing Communication consent.
_Avoid_: Treating guest ticket updates as marketing opt-in

### Members, assignments, commitments, and live plans

**Member**:
A Person with access to a Workspace. Members may have permissions to view or change Event records.
_Avoid_: User when describing Workspace belonging

**First Workspace Cut**:
The first v1 slice includes multi-member Workspaces rather than a single-user-only Organizer flow. The initial Event lifecycle should assume a Host Workspace can have more than one Member, even if detailed permission bundles stay minimal.
_Avoid_: Building the first Event flow around an implicit solo Workspace that must be replaced later

**First Workspace Membership Cut**:
The first v1 account model allows one Person to belong to multiple Workspaces. The product may default to a current or first Workspace in the UI, but the domain model should not assume exactly one Workspace per account.
_Avoid_: Tying a Person's account permanently to a single collective or operating group

**First Authentication Cut**:
The first v1 Organizer and Member flow uses real authentication rather than mocked or dev-only access. Authentication may be minimal, such as email/password or magic link, but must be real enough to support multi-member Workspaces, ownership, invitations, and account-based operational access.
_Avoid_: Proving multi-member Workspace behavior against fake users that must be replaced later

**First Login Method Cut**:
The first v1 authentication method is email and password. Email verification may be optional in local development but should be required before production public ticketing. Magic links, social login, passkeys, and SSO are later authentication capabilities.
_Avoid_: Starting with login methods that make invitation and local debugging harder than the first lifecycle slice requires

**Assignment**:
The link between a piece of Event work and the Person, Contact, Vendor, Collective, or Workspace responsible for it. An Assignment records accountability for work that may otherwise begin as an open need.
_Avoid_: Treating every open Task or Shift as a Commitment before someone is accountable for it

**Assignable**:
Anything in an Event Operations Record that can be assigned to an accountable party. Assignable types include Tasks, Shifts, Contributions, and Open Slots; each type may have more specific subtypes such as Door Shift, Sound Shift, Gear Contribution, Cash Contribution, or Flyer Task.
_Avoid_: Putting work categories directly on Assignment

**Commitment**:
A visible promise that a Person, Contact, Vendor, Collective, or Workspace is expected to fulfill for an Event. Commitments may arise from accepted Assignments, agreed payments, promised gear, confirmed access needs, or follow-up obligations.
_Avoid_: Promise as the canonical term, obligation, todo

**Task**:
A piece of Event work that needs to be completed. A Task may be unassigned, assigned, or resolved; it becomes a Commitment only when an accountable party is attached and expected to follow through.
_Avoid_: Using Task for scheduled labor blocks, money owed, gear promised, or relationship obligations

**Shift**:
A scheduled block of Event labor that needs coverage. A Shift may be open until assigned to Crew or another responsible party.
_Avoid_: Using Shift for unscheduled checklist work

**Timeline**:
The ordered schedule of an Event, including public program times, private operations times, load-in/load-out, doors, performances, setup, breakdown, and Settlement moments. Timeline items may create Shifts, Tasks, Operational Instructions, or Public Event Page schedule entries.
_Avoid_: Treating every schedule item as a Shift

**Actual Timeline**:
The recorded sequence of what actually happened during an Event, including timing changes, delays, skipped items, added items, and real closeout moments. Actual Timeline belongs in the Archive and can suggest Lessons Learned, but Organizers decide what becomes a Lesson Learned.
_Avoid_: Treating the planned Timeline as the historical record

**Run of Show**:
The event-day execution plan that tells Organizers and Crew what happens when, who is responsible, what cues or transitions matter, and what operational details must be followed. Run of Show is derived from the Timeline but is focused on live execution.
_Avoid_: Treating Run of Show as the public Program or the entire planning record

**Run of Show Version**:
A dated version of the Run of Show used to distinguish drafts, published Crew plans, event-day updates, and the final actual sequence remembered in the Archive.
_Avoid_: Overwriting event-day changes without knowing which plan people followed

**Load-In**:
The planned arrival, unloading, setup, and placement process before or during an Event for Crew, Performers, Merchants, Vendors, Exhibitors, Assets, Supplies, and Spaces.
_Avoid_: Treating setup logistics as only generic Tasks

**Load-Out**:
The planned breakdown, packing, cleanup, return, and departure process after or during an Event for Crew, Performers, Merchants, Vendors, Exhibitors, Assets, Supplies, and Spaces.
_Avoid_: Treating breakdown logistics as only generic Tasks

**Timeline Visibility**:
The rule for who can see a Timeline item, such as public, Members only, specific Event Roles, specific Assignments, Ticket Holders, or Need-to-Know recipients.
_Avoid_: Assuming the whole Event schedule is public

**Lineup**:
The public or internal list of Performers, speakers, artists, screenings, Merchants, Exhibitors, or other programmed or featured participants in an Event. Lineup may connect to Timeline items and should group participants by Event Role when helpful, but it is not the same as the full operational Timeline.
_Avoid_: Treating the whole Timeline as the public lineup

**Program**:
The public-facing structure of an Event's featured content, such as performances, talks, screenings, workshops, markets, exhibitions, or activities. Program may use Lineup and public Timeline items but does not include the full private operational Timeline.
_Avoid_: Using Program for private operations planning

**Program Track**:
A named lane within an Event Program, such as a stage, room, workshop track, market area, screening room, or activity stream. Program Tracks may map to Spaces and public Timeline items.
_Avoid_: Using stage for every kind of parallel program area, treating Stage as a universal top-level term

### Contributions, assets, supplies, and agreements

**Contribution**:
Something a Person, Contact, Vendor, Collective, or Workspace provides to support an Event, such as gear, money, space, promotion, food, transport, or access support. A Contribution may become a Commitment when an accountable party is attached.
_Avoid_: Treating all Contributions as payments or donations

**Asset**:
A reusable or trackable item used for Events, such as gear, equipment, supplies, furniture, signage, cash box, projector, PA, tables, or accessibility equipment. Assets may be owned by a Workspace, provided as a Contribution, rented from a Vendor, assigned for transport, or referenced in Incident, Safety, or Archive records.
_Avoid_: Treating trackable gear only as a note or generic task

**Asset Condition**:
The recorded state of an Asset before, during, or after an Event, such as available, checked out, returned, damaged, missing, needs repair, or retired. Asset Condition may create Assignables, Expenses, Safety Notes, or Archive notes.
_Avoid_: Discovering gear problems only through memory or scattered notes

**Asset Inventory**:
The Workspace's collection of reusable or trackable Assets, including ownership, availability, location, condition, and Event use history. Asset Inventory can be used to plan Contributions, Assignments, rentals, repairs, and replacements.
_Avoid_: Treating Event gear as one-off notes when it is reusable or accountable

**Supply**:
A consumable or count-based item used for an Event, such as wristbands, drink tickets, earplugs, masks, tape, water, ice, printed materials, or other supplies. Supplies may be tracked in Asset Inventory when recurring stock matters, but they are not reusable Assets.
_Avoid_: Treating consumables and reusable gear as the same thing

**Merch**:
Goods sold or distributed at an Event by a Performer, Merchant, Host Workspace, Co-Host, or other Event Role, such as shirts, records, zines, posters, art, or other items. Merch may be represented as a simple line item for planning, Settlement, or Archive purposes; detailed sales or inventory tracking should be optional.
_Avoid_: Requiring every Performer or Merchant to track item-level merch sales or inventory

**Open Slot**:
An unfilled Assignable for an Event Role, Shift, Task, or Contribution need that has not yet been assigned to a Person, Contact, Collective, or Workspace. An Open Slot is not a Commitment until assigned and accepted or otherwise made accountable.
_Avoid_: Fake contacts, TBD person

**Confirmation**:
Evidence that an Assignment or Commitment is accepted, agreed, or otherwise treated as reliable. Confirmation may come from the assignee directly or be recorded by an Organizer based on outside communication.
_Avoid_: Requiring every Commitment to be accepted inside the app

**Agreement**:
A recorded understanding between the Host Workspace and another Person, Workspace, or Event Role about participation, payment, requirements, benefits, access, conduct, media, or other Event terms. An Agreement may create Commitments, Payouts, Sponsor Benefits, Venue Requirements, or Operational Instructions.
_Avoid_: Hiding important terms only in notes or messages

**Agreement Evidence**:
The proof or record supporting an Agreement, such as a signed document, checkbox acknowledgment, uploaded file, email, text, outside confirmation, or Organizer-recorded note. Agreement Evidence shows why the Agreement is treated as valid.
_Avoid_: Requiring every Agreement to be a formal contract

**Code of Conduct**:
A public or internal policy describing expected behavior, boundaries, safety expectations, and consequences for an Event, Event Series, or Workspace. Code of Conduct acknowledgment may be captured as Agreement Evidence where needed; it should be optional in v1 but inherited or strongly prompted before publishing public ticketing or application flows.
_Avoid_: Treating conduct expectations as informal vibes only

**Media Policy**:
A public or internal policy describing photo, video, recording, livestream, press, and Archive use expectations for an Event, Event Series, or Workspace. Media Policy may create Operational Instructions and Agreement Evidence where consent or acknowledgment is needed.
_Avoid_: Assuming all Event media can be captured or published freely

**Media Restriction**:
A specific limit on photo, video, recording, livestream, press, or Archive capture for an Event, Space, Person, or time period. Media Restrictions may include public instructions and private Need-to-Know details, and may create Operational Instructions for Crew, Door, photographers, or Public Archive review.
_Avoid_: Treating Media Policy as only static legal text

### Assignment and commitment states

**Assignable Status**:
The operational state of an Assignable, such as open, assigned, covered, completed, or cancelled. Assignable Status describes whether the Event need is handled, not whether a specific party fulfilled their promise.
_Avoid_: Using Assignable Status as accountability history

**Commitment Status**:
The accountability state of a Commitment, such as proposed, confirmed, fulfilled, blocked, cancelled, disputed, or settled. Commitment Status describes whether a responsible party followed through or still owes resolution.
_Avoid_: Using Commitment Status as the only state for open work

**Dispute**:
A contested Event record or obligation where parties disagree about money, fulfillment, responsibility, access, safety, or another consequential outcome. Disputes should be typed by context and follow Need-to-Know visibility.
_Avoid_: Using Dispute only for payment processor chargebacks

**Dispute Resolution**:
The recorded outcome of a Dispute, including what was decided, who approved it, what changed, and any follow-up Commitments, Refunds, Payouts, Safety Notes, or Settlement Adjustments. Dispute Resolution approval depends on dispute type and required Permission, follows Need-to-Know, and should leave an Audit Trail.
_Avoid_: Resolving Disputes only by changing status without context

### Event roles and participation applications

**Event Role**:
The part a Person, Workspace, Collective, or manually entered entity plays in a specific Event, such as Organizer, Crew, Performer, Vendor, Merchant, Exhibitor, Venue, or Co-Host.
_Avoid_: Treating Event Roles as permanent identity

**First Event Role Cut**:
The first v1 tracer bullet includes only Host Workspace ownership and attendee Ticket reservations, not explicit Event Roles for Performers, Vendors, Merchants, Crew, Venues, Co-Hosts, or Exhibitors. Event Roles, applications, lineups, assignments, and participant Settlement are later capabilities after the lifecycle spine works.
_Avoid_: Blocking the first Event lifecycle on participant and production modeling

**Vendor**:
An Event Role for a Contact or Workspace that the Event pays to provide goods or services, such as catering, sound, security, equipment rental, or hired production support.
_Avoid_: Using Vendor for people who sell to attendees

**Merchant**:
An Event Role for a Contact or Workspace that is a business or commercial participant selling goods or services to attendees at an Event, such as a food truck, artist table, zine seller, or trunk show booth. A Merchant may pay a fee, share revenue, or participate for free with a zero-dollar fee.
_Avoid_: Vendor, booth as the person or group

**Merchant Settlement**:
The Event-facing money relationship with a Merchant, such as booth fee, revenue share, waived fee, reported sales total, amount owed, amount paid, or dispute. Merchant Settlement does not require item-level Merchant sales tracking.
_Avoid_: Treating Merchant inventory or point-of-sale as part of the core Event record

**Participation Application**:
A request from a Person, Workspace, guest applicant, or manually entered entity to participate in an Event through an Event Role, such as Merchant, Exhibitor, Performer, Crew, or Sponsor. A Participation Application may capture role-specific needs, approval state, fees, setup details, and follow-up; guest applicants may create or link an account later. Rejected, waitlisted, and withdrawn Participation Applications remain part of the Event record by default with Need-to-Know visibility.
_Avoid_: Creating unrelated application systems for each Event Role, separate role-specific application terms as canonical language

**Application Form**:
A public, unlisted, invite-only, or internal form used to create Participation Applications for an Event Role. Application Forms collect role-specific information and should follow Publish Status and Need-to-Know rules.
_Avoid_: Assuming all applications are public or all are internal

**Application Field Visibility**:
The visibility rule for a question or answer on an Application Form, controlling who can view sensitive applicant information after submission. Application Field Visibility should follow Need-to-Know.
_Avoid_: Making every application answer visible to everyone managing the Event

**Application Template**:
A reusable Application Form pattern for a Workspace, Event Template, or Event Series, including role, questions, requirements, fees, and review defaults.
_Avoid_: Rebuilding the same application form from scratch for every Event

**Application Status**:
The state of a Participation Application, such as Draft, Submitted, Under Review, Accepted, Waitlisted, Rejected, Withdrawn, or Confirmed. Accepted applications may still require Confirmation, Payment, Assignment, or account linking before participation is complete; Confirmed applications are ready enough to appear in the Event plan.
_Avoid_: Treating acceptance as the same as fully confirmed participation, using Converted as user-facing language

**Exhibitor**:
An Event Role for a Contact or Workspace that presents, displays, demos, tables, or shares something at an Event without primarily being a commercial seller or being paid by the Event. An Exhibitor may accept donations or sell incidental items, but their role is civic, charitable, political, educational, artistic, or informational rather than business-first.
_Avoid_: Merchant when commercial selling is not the primary relationship, Vendor when the Event is not paying them

**Performer**:
An Event Role for a Contact or Workspace that appears in the Event program to perform, present, play, screen, speak, or otherwise provide the public-facing cultural content of the Event. A Performer may have any payment arrangement.
_Avoid_: Vendor for performers, talent as the generic term

### Venue, space, capacity, and placement

**Venue**:
The Event Role for the place, host, or space provider where an Event happens. A Venue may be a formal business, informal DIY space, private home, outdoor location, or temporary pop-up site.
_Avoid_: Location as the domain term, facility

**Location Visibility**:
The rule for what location information is shown to different audiences, such as public area, ticket-holder address, Member-only details, or private operational instructions. Location Visibility may protect private homes, DIY spaces, pop-ups, or sensitive Events.
_Avoid_: Assuming every Event publishes its exact address

**Location Release**:
The rule or timing for revealing protected location details, such as immediately, after Ticket purchase, after approved RSVP, at a scheduled time, or through Event Brief or Notification only. Location Release uses existing audience and access concepts such as Public, Ticket Holder, signed-in RSVP, approved RSVP, Member, Event Role, Connection, or Event Brief recipient rather than a separate trust score.
_Avoid_: Treating Location Visibility as static for the whole Event lifecycle

**Venue Requirement**:
A rule, constraint, obligation, or operational detail required by a Venue or Space for an Event, such as load-in rules, curfew, capacity, insurance, sound limits, security rules, bar rules, or prohibited equipment. Venue Requirements may create Operational Instructions, Assignables, Commitments, or Settlement terms.
_Avoid_: Burying Venue constraints in generic notes

**Default Requirement**:
A reusable requirement saved on a Venue, Space, Contact, or Workspace that can be copied into an Event as a Venue Requirement, Operational Instruction, Assignable, or Commitment.
_Avoid_: Assuming every Event starts from blank venue rules

**Space**:
A physical area used within an Event, such as a room, stage, booth area, entrance, green room, kitchen, or outdoor zone. A Venue may contain multiple Spaces.
_Avoid_: Venue when referring to a sub-area

**Event Map**:
A visual or structured representation of Spaces, entrances, exits, booths, stages, access routes, door positions, load-in paths, and other physical layout details for an Event. Event Map may include public information and private Need-to-Know operational details.
_Avoid_: Requiring a precise architectural floor plan for every Event

**Placement**:
The assigned location of an Event Role, Asset, Supply, Access Marker station, booth, table, or operational area within a Space or Event Map. Placement may create Operational Instructions or Assignments.
_Avoid_: Using booth placement for every physical setup decision

**Placement Visibility**:
The rule for who can see Placement details, such as public, Ticket Holders, Members, specific Event Roles, or Need-to-Know recipients. Placement Visibility protects sensitive layout details such as cash box locations, safety stations, private green rooms, access routes, and restricted areas.
_Avoid_: Assuming every physical layout detail is safe to publish

**Event Capacity**:
The total intended or allowed attendance for an Event.
_Avoid_: Using Event Capacity for every Ticket Type, comp pool, booth count, or room limit

**Space Capacity**:
The intended or allowed occupancy for a specific Space within an Event or Venue.
_Avoid_: Treating Space Capacity as the same as total Event Capacity

### Money, settlement, and payouts

**Settlement**:
The Event's money state and reconciliation: what came in, what went out, what is owed, what was paid, and what remains. Settlement may include door income, Merchant fees, Vendor costs, Performer Payouts, Venue splits, expenses, donations, and retained funds; it should support generating an Event Report.
_Avoid_: Payment as the whole money model, accounting

**Split**:
A rule or recorded outcome for dividing Net Proceeds, Payments, Merchant fees, Donations, or other money between payees such as the Host Workspace, Venue, Performers, Crew, Beneficiaries, or Co-Hosts.
_Avoid_: Hiding revenue-sharing rules in notes

**Settlement Adjustment**:
A manual change to a calculated Settlement amount, Split, Payment, Payout, Expense, Refund, or Net Proceeds value, recorded with a reason. Applying a Settlement Adjustment requires money-level Permission such as Treasurer or Owner; other Members may propose adjustments if allowed.
_Avoid_: Silent overrides that make Settlement hard to audit

**Settlement Status**:
The state of a Settlement, such as Draft, Closed, Adjusted, Disputed, or Finalized. Settlement Status preserves the difference between end-of-night closeout and later corrections.
_Avoid_: Treating Settlement as an overwrite-only report

**End of Night**:
The closeout moment when an Event stops taking in new money, captures its current Settlement state, generates an End of Night Event Report Version, and moves the Event toward its Archive. End of Night does not require that all Payouts have already gone out.
_Avoid_: Treating End of Night as making the Event permanently uneditable

**Event Close**:
The later closure point when an Event is considered operationally finished: no more Payments, Payouts, disputes, or required follow-up changes are expected. Event Close generates a final Event Report Version; it does not prevent Archive updates, but it marks the Event as no longer active work.
_Avoid_: End of Workspace

**Event Reopen**:
A deliberate action that moves a Closed Event back into active follow-up because a meaningful Payment, Payout, Refund, Dispute, Archive correction, or required change appeared after Event Close. Event Reopen should require appropriate Permission, leave an Audit Trail, and generate a new final Event Report Version when the Event closes again.
_Avoid_: Editing closed Events silently without marking that active work resumed

**Payment**:
Incoming money received or expected by an Event, Workspace, or Organizer, such as ticket sales, door income, donations, Merchant fees, or other collected funds.
_Avoid_: Using Payment for outgoing settlement money; use Payout

**Payment Status**:
The state of incoming money for an Event, Order, Door Sale, Donation, Merchant fee, or other Payment, such as Pending, Authorized, Paid, Failed, Refunded, Partially Refunded, Disputed, Cancelled, or Manually Recorded. Payment Status is independent from any processor-specific status.
_Avoid_: Treating Stripe status as the domain status

**Donation**:
Voluntary money given to support an Event, Collective, beneficiary, or cause, separate from required Ticket price or Merchant fees. A Donation creates a Payment but may or may not be tied to a Ticket, Order, or Entry.
_Avoid_: Treating every Donation as ticket revenue

**Beneficiary**:
A Person, Workspace, Collective, organization, or cause intended to receive Donations, Net Proceeds, or a designated Payout from an Event.
_Avoid_: Assuming all Event proceeds belong to the Host Workspace

**Sponsor**:
An Event Role for a Person, Workspace, business, organization, or manually entered entity that supports an Event with money, goods, services, promotion, or other value, usually in exchange for recognition or agreed benefits.
_Avoid_: Treating Sponsor as Vendor, Merchant, or Beneficiary

**Sponsor Benefit**:
Recognition, access, comps, placement, booth/table presence, or other value promised to a Sponsor in exchange for their support. Sponsor Benefits may create Commitments for the Host Workspace.
_Avoid_: Hiding sponsor obligations in notes

**Sponsor Visibility**:
Whether a Sponsor is publicly credited, public-anonymous, privately tracked, or internally restricted on public-facing and Workspace-facing Event materials. Sponsor Visibility is separate from the underlying Contribution, Payment, or Sponsor Benefit records and should follow Need-to-Know when Sponsor identity is sensitive.
_Avoid_: Assuming every Sponsor must be publicly listed

**Payment Method**:
The way a Payment is collected or recorded, such as online card, door card, cash, mobile transfer, external payment link, or manual adjustment.
_Avoid_: Assuming all Payments go through Stripe

**Payment Info**:
Sensitive information used to record or prepare Payments, Payouts, Reimbursements, or Merchant Settlements, such as preferred payout method, account connection status, tax/payment notes, or manual payment instructions. Payment Info should follow Need-to-Know and may be shared through explicit Shared Fields.
_Avoid_: Storing payment details in general Contact notes

**Tax Info**:
Sensitive information needed for tax, reporting, or payout compliance. Tax Info is not part of v1 manual Settlement by default and should only be collected when there is a clear legal or payout-provider need.
_Avoid_: Collecting tax details casually in Contact notes or Application Forms

**Comp**:
A free or waived admission granted by the Host Workspace, Venue, Performer allotment, Guest List, policy exception, or door decision. A Comp may create a Ticket or Entry but does not create incoming Payment.
_Avoid_: Treating Comp as revenue or as a normal Payment Method

**Payout**:
Money owed or sent from an Event, Workspace, or Organizer to a Performer, Vendor, Venue, Crew member, Merchant, Collective, or other payee. A Payout may be tracked manually in Settlement or later processed through a payout provider.
_Avoid_: Using Payment for both incoming ticket money and outgoing settlement money

**Payout Status**:
The state of money owed or sent from an Event, such as Planned, Owed, Ready, Sent, Failed, Disputed, Waived, or Cancelled. Payout Status is independent from any future payout provider status.
_Avoid_: Treating manual payout tracking and provider payout status as the same thing

**Expense**:
A cost incurred for an Event, such as printing, supplies, rentals, permits, travel, food, or Vendor services. An Expense may be already paid, unpaid, reimbursable, or included in Settlement.
_Avoid_: Treating every outgoing amount as a Payout

**Reimbursement**:
A Payout owed to a Person or Workspace for an Expense they covered on behalf of an Event.
_Avoid_: Losing who fronted the money by recording only the original Expense

### Ticketing, door, RSVP, and attendance

**Door Record**:
The Event's operational record of attendance, entry types, guest list use, door income, comps, no-shows, and headcount. A Door Record may be detailed per person or summarized as counts.
_Avoid_: Ticketing as the generic term, attendance as the full door model

**Door Sale**:
An admission transaction recorded at the Event entrance, such as cash, card, mobile payment, comp, or manual entry. A Door Sale may create Tickets, Entries, Payments, and Door Record totals according to the Entry Policy.
_Avoid_: Treating all ticket activity as online checkout

**Guest List**:
A named list of people expected or allowed to enter an Event through comp, RSVP, house list, performer allotment, press, or other non-standard admission. Guest List entries may issue Tickets or authorize Entries depending on the door workflow.
_Avoid_: Treating Guest List as unrelated to Tickets and Door Record

**RSVP**:
A response indicating that someone intends to attend an Event. An RSVP may come from a signed-in Person, a newly created account, or an anonymous guest; it may be unticketed for a free Event, may reserve or issue a free Ticket for a free ticketed Event, or may be a soft interest signal for a paid ticketed Event. RSVP alone does not always guarantee Entry.
_Avoid_: Treating every RSVP as a Ticket or guaranteed admission

**RSVP Visibility**:
Whether an RSVP is publicly shown, visible only to the Host Workspace, or hidden from public attendee lists. RSVP Visibility may be influenced by a Person's account-level preference, their Event-specific choice, and the Host Workspace's Event-level display policy.
_Avoid_: Assuming every RSVP is publicly displayed or always tied to a Public Profile

**Anonymous RSVP**:
An RSVP not publicly linked to a Person or Public Profile. A guest may RSVP anonymously without contact follow-up; a signed-in Person may also choose an anonymous RSVP for public display while still receiving account-based Notifications where allowed. Signed-in anonymous RSVPs may remain visible to the Host Workspace under Need-to-Know rules.
_Avoid_: Collecting contact details for anonymous RSVP follow-up

**RSVP Mode**:
The rule for how RSVPs behave for an Event, such as unticketed free attendance, free ticket reservation, soft interest for paid ticket sales, signed-in-only RSVP, public-only RSVP, anonymous RSVP allowance, or RSVP disabled. RSVP Mode may allow more RSVPs than Event Capacity when no-show assumptions are intentional.
_Avoid_: Assuming RSVP count and allowed attendance are always the same

**Admission Buffer**:
An opt-in, configurable allowance for more Tickets, RSVPs, or Guest List spots than Event Capacity based on expected no-shows or door policy. Admission Buffer may be expressed as a percentage, fixed extra quantity, unlimited allowance, or another explicit rule, and should be visible to Organizers because it can create crowding or turn-away risk.
_Avoid_: Accidental overbooking caused by unclear capacity settings

**RSVP Count**:
The RSVP headcount signal for an Event, separated by source or confidence such as signed-in RSVPs and anonymous guest RSVPs. RSVP Count may contribute to Admission Buffer planning but should be shown separately from Tickets issued and Entries recorded.
_Avoid_: Treating anonymous RSVPs, signed-in RSVPs, Tickets, and Entries as the same count

**Entry**:
One admission to an Event, whether paid, comped, guest-listed, performer-allotted, RSVP-based, or manually counted.
_Avoid_: Ticket when no ticket was issued

**Check-In**:
The Door Record action that validates a Ticket, Guest List entry, RSVP, or manual admission and records that someone entered or was counted for the Event.
_Avoid_: Treating Check-In, Ticket, and Entry as the same thing

**First Door Cut**:
The first v1 Door flow supports manual Ticket lookup and check-in before QR scanning. Tickets may still have a code or QR representation from the start, but proving the Door Record does not depend on camera or scanner support.
_Avoid_: Blocking Door Record validation on QR scanning UI complexity

**First Door Device Cut**:
The first v1 Door Check-In interface must work well on mobile devices. Desktop support is acceptable, but the Door flow should be designed for phone use because check-in is likely to happen at the Event entrance.
_Avoid_: Treating Door Check-In as a desktop-only back-office workflow

**First Check-In Cut**:
The first v1 Door flow allows one Check-In per Ticket and no re-entry. If the same Ticket is presented again, the Door Record should show that it was already checked in. Re-entry, Check-Out, Access Markers, and Occupancy Count are later Door capabilities.
_Avoid_: Building re-entry and occupancy management before basic Ticket validation is proven

**Entry Policy**:
The rules governing how Tickets, Guest List entries, RSVPs, and manual admissions become Entries, including re-entry, age limits, capacity handling, door price, comp rules, check-in limits, and one-in-one-out behavior when a Venue or Space is at capacity.
_Avoid_: Hard-coding one door flow for all Events

**Age Policy**:
The rule for age-related admission or access at an Event or Space, such as all ages, 18+, 21+, guardian required, ID required, or age-restricted areas. Age Policy should inform Public Event Page, Entry Policy, Door Record, and Operational Instructions.
_Avoid_: Burying age restrictions in event description text only

**ID Check**:
A Door Record action or requirement where Crew verifies identity, age, ticket ownership, Guest List status, or access to age-restricted areas. ID Check should record only the needed outcome, not unnecessary identity document details.
_Avoid_: Storing ID document data unless legally required

**Access Marker**:
A physical or digital marker used during an Event to show allowed access, such as wristband, hand stamp, badge, pass, or credential. Access Markers may reflect Entry Policy, Age Policy, re-entry, Crew status, Performer access, or Space restrictions; they can be tied to Supplies and counted in the Door Record when issued.
_Avoid_: Treating wristbands or stamps as only Supplies when they also encode access

**Occupancy Count**:
The current number of people inside an Event or Space for capacity management. Occupancy Count changes with entries and exits and is separate from total Entries or attendance.
_Avoid_: Using total Entry count to decide whether the room is currently full

**Check-Out**:
The Door Record action that records someone leaving an Event or Space for Occupancy Count and re-entry purposes. Check-Out may be tied to a known Ticket or Entry, or recorded as an anonymous count adjustment.
_Avoid_: Assuming every exit can be matched to a named Attendee or Ticket

**Ticket**:
A claim to one Entry for an Event, whether paid, free, comped, RSVP-based, or manually issued.
_Avoid_: Using Ticket for anonymous door counts

**Ticket Status**:
The state of an individual Ticket, such as Reserved, Issued, Checked In, Voided, Refunded, Transferred, or Expired. Ticket Status is separate from Order Status and Payment Status.
_Avoid_: Treating all Tickets in an Order as having the same state

**Ticket Holder**:
The Person or guest currently associated with a Ticket when known. A Ticket may be issued without a named Ticket Holder, especially for Guest Checkout, group Orders, comps, or bearer-style QR entry.
_Avoid_: Requiring every Ticket to identify an Attendee before Entry

**Ticket Transfer**:
A change in who controls or is associated with a Ticket. In v1, transfer may be informal through sharing the Ticket; formal transfer, resale, and ownership history can come later.
_Avoid_: Building resale workflows before core ticketing and door check-in are stable

**Ticket Type**:
A category of Ticket with shared rules such as price, capacity, comp status, sales window, or door policy.
_Avoid_: Tier when the category is not price-ranked

**Ticket Allocation**:
The quantity of Tickets reserved for a Ticket Type, sales channel, guest list, comp pool, or other admission category. Ticket Allocation is distinct from Event Capacity or Space Capacity.
_Avoid_: Using capacity for every ticket bucket

**First Ticket Allocation Cut**:
The first v1 Ticketing flow has one free Ticket Type with one Event-level Ticket Allocation number. Multiple Ticket Types, tiers, comps, Guest List, waitlist, per-person limits, sales windows, and multiple allocations are later capabilities.
_Avoid_: Building ticket tiering before the basic reservation and Door lifecycle is proven

**First Full Allocation Cut**:
When the first v1 Ticket Allocation is full, public reservation closes and the Public Event Page shows a simple full or sold-out state. Waitlists, Admission Buffer, Organizer approval queues, overbooking, and request-to-attend flows are later capabilities.
_Avoid_: Adding overflow workflows before basic allocation enforcement is proven

**Pricing Mode**:
The way a Ticket Type determines price, such as free, fixed price, sliding scale, pay-what-you-can, donation, or at-door-only.
_Avoid_: Assuming every Ticket Type has one fixed price

**First Payment Cut**:
The first v1 tracer bullet does not include Stripe, paid checkout, webhooks, Refunds, Processing Fees, Platform Fees, or Payment Status. The Order and Ticket model should remain compatible with paid Tickets later, but the first lifecycle proof uses free Ticket reservations only.
_Avoid_: Letting payment integration complexity block the first Event lifecycle path

**Order**:
A record of one or more Tickets reserved or issued together. An Order may be free, manually recorded, or connected to payment processing.
_Avoid_: Payment as the ticketing container

**Order Status**:
The state of a Ticket Order as a container for one or more Tickets and related Payments, such as Draft, Pending Payment, Confirmed, Partially Fulfilled, Cancelled, Refunded, Partially Refunded, or Disputed. Order Status is separate from Payment Status and individual Ticket state.
_Avoid_: Using Payment Status as the whole Order lifecycle

**Guest Checkout**:
A ticket checkout or reservation flow that lets an Attendee place an Order without creating an account. Guest Checkout may collect contact information and later link to a Person or Public Profile.
_Avoid_: Requiring attendee accounts for basic Ticket purchase or reservation

**First Attendee Cut**:
The first v1 free Ticket reservation flow allows Guest Checkout. Attendees can reserve a free Ticket without creating an account, receive a Ticket link or QR by email, and optionally create or link an account later. Organizer and Workspace access remains account-based.
_Avoid_: Blocking the first public Ticket flow on attendee account creation

**First Reservation Data Cut**:
The first v1 free Ticket reservation flow requires only an email address and may optionally collect a Display Name. Email exists to deliver and resend the Ticket link or QR. Phone number, Legal Name, pronouns, age, address, account password, and marketing consent are out of the first reservation slice.
_Avoid_: Collecting attendee profile, compliance, or marketing data before it is needed

**First Ticket Delivery Cut**:
The first v1 Ticket reservation flow confirms the Ticket on screen and delivers the Ticket link or QR by email. Early implementation may log or store outbound email records before real sending, but the domain flow assumes email delivery and later resend support.
_Avoid_: Treating on-screen confirmation alone as durable Ticket delivery

**First Email Sending Cut**:
The first v1 slice may record intended outbound emails before integrating real email sending. Ticket delivery and invitation flows should still behave as email-based flows in the product language, with real provider delivery added later.
_Avoid_: Confusing development-time logged emails with a non-email user experience

**Attendee**:
A known Person who attends or is expected to attend an Event. Anonymous or counted admissions should be represented as Entries rather than Attendees.
_Avoid_: Using Attendee as the umbrella for all Event participants

### Archive, reports, series, and follows

**Archive**:
The durable memory of an Event, including its flyer, lineup, final attendance, participants, Settlement summary, links, media, notes, lessons learned, and private summaries where appropriate. An Archive begins at End of Night and may continue receiving updates after Event Close.
_Avoid_: Memory as the canonical term

**Lesson Learned**:
A reusable insight captured from an Event for future planning, such as what worked, what failed, what to change, who to call again, what to avoid, or what access, safety, or logistics detail mattered. Lessons Learned belong in the Archive and may inform future Event templates.
_Avoid_: Burying repeatable post-event learning in generic notes

**Event Template**:
A reusable starting point for creating future Events from prior Events, recurring formats, or saved planning patterns. An Event Template may include roles, Assignables, Ticket Types, Entry Policy, Settlement defaults, Access Information, and Lessons Learned.
_Avoid_: Copying old Events blindly without reviewing changed details

**Event Series**:
A named recurring or related set of Events that share identity, format, branding, audience, or planning patterns. Each occurrence is still its own Event with its own Event Operations Record, Tickets, Settlement, and Archive.
_Avoid_: Treating a recurring series as one giant Event

**Series Page**:
A public-facing page for an Event Series that shows shared identity, description, upcoming Events, past Public Archive Pages, links, and selected public information. A Series Page does not replace each Event's Public Event Page.
_Avoid_: Forcing recurring Events to rely only on individual Event pages

**Follow**:
An opt-in relationship where a signed-in Person receives allowed Notifications about a Public Profile, Event Series, Workspace, or public Event activity. Following requires an account, does not grant Workspace access, and should respect Notification Preferences.
_Avoid_: Treating follows as permission for unrestricted Marketing Communication

**Event Watch**:
An opt-in or automatically created operational relationship where a signed-in Person receives allowed Notifications about a specific Event, such as ticket availability, schedule changes, cancellation, rescheduling, or public Archive publication. Account-linked Ticket Orders and signed-in RSVPs may create Event Watch for operational updates; Event Watch ends or becomes inactive after Event Close unless the Person also Follows a related Public Profile, Workspace, or Event Series.
_Avoid_: Treating one-time Event interest as a durable Follow

**Public Archive Page**:
A published public view of selected Archive material for an Event, limited to information that is already public or deliberately approved for public release, such as flyer, lineup, public credits, media, links, and public notes. A Public Archive Page excludes private Settlement details, Safety Notes, Access Needs, and other Need-to-Know information by default.
_Avoid_: Publishing the private Archive by default

**Archive Review**:
The pre-publication review of Archive material before it appears on a Public Archive Page, checking Publish Status, Media Policy, Media Restrictions, public/private boundaries, credits, and sensitive information.
_Avoid_: Automatically publishing uploaded media to the public Archive

**Credit**:
A public or private acknowledgment of a Person, Workspace, or manually entered entity's contribution to an Event, Contribution, Event Role, Archive item, media item, Sponsor Benefit, or Public Event Page. Credits should be attachable to specific records and respect Publish Status, Sponsor Visibility, Media Policy, and Need-to-Know restrictions.
_Avoid_: Burying attribution only in notes or captions

**Event Report**:
A generated snapshot of an Event Operations Record, Archive, and Settlement at a point in time for sharing, review, grant reporting, accounting, or future planning. Event Reports may include summary planned-versus-actual timing from the Timeline and Actual Timeline when available.
_Avoid_: Treating reports as always-live pages without version context

**Event Report Version**:
A generated version of an Event Report tied to a point in time, such as End of Night report, adjusted report, final Settlement report, or public archive report.
_Avoid_: Overwriting earlier reports without knowing what changed

**First Event Report Cut**:
The first v1 Event Report is a minimal lifecycle report generated at End of Night. It includes Event title and date, Publish Status or public URL, Ticket Allocation, Tickets reserved, Tickets checked in, no-shows, generation time, and the Member who generated it. Money, archive media, Lessons Learned, Safety Notes, and Timeline variance are later report capabilities.
_Avoid_: Making the first report depend on Settlement, archive curation, or advanced operations data

**First Archive Cut**:
The first v1 Event lifecycle ends with a private Event Report rather than a Public Archive Page. Public Archive publishing, media review, public Credits, and post-event public page rules are later lifecycle capabilities.
_Avoid_: Blocking the first End of Night flow on public archive curation

### Safety, access, privacy, and permissions

**Safety Note**:
A permission-controlled record about harm, risk, conflict, access, medical needs, lost items, damage, or follow-up care connected to an Event. Safety Notes exist for care, continuity, and accountability, not casual visibility or punishment.
_Avoid_: Incident as the blanket term, surveillance log

**Safety Detail**:
The sensitive explanation, report, or context behind a Safety Note or Incident, visible only to Members with explicit Need-to-Know. Operational safety instructions should be shareable without exposing Safety Details.
_Avoid_: Sharing sensitive reports when only an operational action is needed

**Incident**:
A serious Safety Note involving harm, credible risk, harassment, medical emergency, police/security involvement, major conflict, or significant damage.
_Avoid_: Using Incident for every minor note or concern

**Access Information**:
First-class information about the accessibility conditions of a Venue, Space, or Event, such as step-free access, bathrooms, seating, lighting, sound, air quality, transit, parking, interpreters, sensory conditions, and contact points.
_Avoid_: Accessibility notes as an unstructured catch-all

**Access Need**:
A specific need or accommodation requested by or associated with a Person participating in an Event. Access Needs should be permission-controlled, shared only where operationally necessary, and not broadly visible to the Host Workspace by default.
_Avoid_: Accessibility Profile as the generic term

**Access Detail**:
The sensitive explanation or context behind an Access Need, visible only to Members with explicit Need-to-Know. The operational accommodation should be shareable without exposing Access Details.
_Avoid_: Sharing medical or personal explanations when only the accommodation is needed

**Need-to-Know**:
The privacy principle that sensitive Event information is only visible to people who need it for a specific operational reason. Safety Notes, Access Needs, money, contact details, and private notes should follow Need-to-Know access.
_Avoid_: Making all Event collaborators able to see everything

**Permission**:
A specific ability to view, create, update, export, or manage part of an Event Operations Record. Permissions may come from Workspace membership, Event Role, or explicit sharing.
_Avoid_: Treating role names alone as the complete privacy model

**Audit Trail**:
A record of sensitive or consequential changes, including who changed what, when, and why where applicable. Audit Trail should cover money records, Permissions, Safety Notes, Access Needs, Refunds, Settlement Adjustments, Event Status changes, publication changes, Participation Application status changes, and sensitive application review changes; Audit Trail visibility follows Need-to-Know by record type.
_Avoid_: Relying only on current field values for trust-sensitive records

**First Audit Trail Cut**:
The first v1 slice records a minimal Audit Trail for consequential lifecycle actions: publishing an Event, reserving a Ticket, checking in a Ticket, running End of Night or generating an Event Report, inviting a Member, and accepting a Workspace invitation. Full field-level diff history is a later capability.
_Avoid_: Building full version history before the first lifecycle path is proven

**Retention Policy**:
The rule for how long Event records, sensitive details, applications, Audit Trail entries, money records, and public or archive materials are kept, hidden, exported, or deleted. Retention Policy should balance privacy, accountability, legal obligations, and scene memory.
_Avoid_: Keeping sensitive records forever by accident or deleting accountability records without policy

### Data portability and cleanup

**Data Export**:
A downloadable copy of selected Workspace or Event records, such as Event Reports, Settlement data, Door Records, Archives, Contacts, or Asset Inventory. Data Export should respect Permissions, Need-to-Know, and Retention Policy; sensitive records such as Safety Details, Access Details, restricted sponsor identity, and full sensitive Audit Trail entries should not be included in default exports and require explicit scoped export when exportable at all.
_Avoid_: Locking organizers into subcult-os as the only holder of their scene records

**Data Import**:
A structured way to bring external records into a Workspace or Event, such as Contacts, Guest Lists, Asset Inventory, or other planning data. Data Import should validate fields, preserve source context, and avoid importing sensitive records without explicit review.
_Avoid_: Assuming every spreadsheet can become trusted data without cleanup

**Import Source**:
The origin of imported records, such as a CSV, spreadsheet, ticketing platform export, contact list, or manual batch entry. Import Source helps Members understand trust, cleanup needs, and provenance.
_Avoid_: Treating imported data as if it were created and verified inside subcult-os

**Suggested Match**:
A possible link between an imported or manual Contact and an existing Person, Workspace, or Public Profile. Suggested Matches help clean up records but do not create Connections or overwrite Contact data without approval.
_Avoid_: Automatically merging imported contacts into live identities

**Contact Merge**:
A deliberate action that combines duplicate or overlapping Contact records while preserving useful history, Import Sources, notes, Event references, and Suggested Matches. Contact Merge requires contact-management Permission, should leave an Audit Trail, and should not create a Connection unless both Workspaces accept one.
_Avoid_: Silent deduplication that loses provenance or relationship context

### Workspace permission bundles and fallback access

**Workspace Role**:
A default Permission bundle for a Member within a Workspace, such as Owner, Admin, Member, or Viewer. Workspace Roles are not the same as Event Roles.
_Avoid_: Using Workspace Roles to describe what someone does at an Event

**First Permission Cut**:
The first v1 multi-member Workspace permission model includes only Owner and Member. Owners manage the Workspace, invite or remove Members, create and publish Events, run End of Night, and view Event Reports. Members can help edit Event details and run Door Check-In. Admin, Viewer, Treasurer, and Safety Lead remain later permission bundles rather than first-slice roles.
_Avoid_: Starting with a detailed role matrix before the lifecycle path is proven

**First Lifecycle Permission Cut**:
In the first v1 slice, only Owners can publish an Event or run End of Night. Members can edit Event details and run Door Check-In, but lifecycle state changes remain Owner-controlled.
_Avoid_: Letting every Member perform consequential lifecycle transitions before a fuller permission model exists

**First Invitation Cut**:
The first v1 multi-member Workspace flow includes minimal email invitations. An Owner invites someone by email; the invitee accepts after signup or login and becomes a Member. Custom invite messages, expiration rules, role selection beyond Member, invite resend, and advanced invitation management are later capabilities.
_Avoid_: Claiming multi-member Workspaces while requiring manual database access to add Members

**First Member Removal Cut**:
The first v1 multi-member Workspace flow allows an Owner to remove a Member, causing that Person to lose Workspace and Event access going forward. Ownership transfer, account suspension, voluntary leave flows, historical anonymization, and detailed removal reason workflows are later capabilities.
_Avoid_: Creating multi-member Workspaces with no way for Owners to correct membership

**Treasurer**:
A Workspace Role or Permission bundle for a Member trusted to view and manage money records such as Settlement, Payments, expenses, and payouts. For a specific Event, settlement work should be represented as Assignments rather than a separate Event Role.
_Avoid_: Treasurer as a generic Event Role

**Safety Lead**:
A Crew specialty or Permission bundle for a Member trusted to handle Safety Notes, Incident follow-up, de-escalation, medical coordination, or other care/accountability work for an Event.
_Avoid_: Safety as a broad Event Role

**Event Brief**:
A limited, shareable view of Event information prepared for someone involved in an Event who is not using the Workspace directly. Event Briefs are a fallback for people who will not log in, not the primary collaboration model.
_Avoid_: Portal for the MVP, public page when the view is private or role-specific

**Access Status**:
The relationship between a Person and a Workspace or Event access flow, such as Member, Invited, or Brief Only. Access Status describes whether someone can use the app directly, not what role they play at an Event.
_Avoid_: Connected/not connected as the canonical wording
