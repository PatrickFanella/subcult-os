package app

import (
	"fmt"
	"net/http"
	"testing"
)

// discoveryTestPlaceRecord builds a valid tv.subcult.place record body.
func discoveryTestPlaceRecord(name, locality, region, country string) []byte {
	return []byte(fmt.Sprintf(
		`{"$type":"tv.subcult.place","name":%q,"locality":%q,"region":%q,"country":%q,"coordinates":{"latitude":"41.8781","longitude":"-87.6298"},"createdAt":"2026-09-23T00:00:00Z"}`,
		name, locality, region, country,
	))
}

// discoveryTestOccurrenceRecord builds a valid tv.subcult.event.occurrence
// record body, using the same fixed placeholder profile strong reference
// cultural_public_projection.go uses (a real, validator-passing at-uri/cid
// pair; not treated as a live reference).
func discoveryTestOccurrenceRecord(name, startsAt string, placeURI string) []byte {
	place := ""
	if placeURI != "" {
		place = fmt.Sprintf(`,"place":{"uri":%q,"cid":%q}`, placeURI, publicPreviewPlaceholderCID)
	}
	return []byte(fmt.Sprintf(
		`{"$type":"tv.subcult.event.occurrence","name":%q,"profile":{"uri":%q,"cid":%q}%s,"startsAt":%q,"createdAt":"2026-09-23T00:00:00Z"}`,
		name, publicPreviewPlaceholderURI, publicPreviewPlaceholderCID, place, startsAt,
	))
}

func newDiscoveryProjectionProcessor(t *testing.T, fx lifecycleFixture) *ProjectionProcessor {
	t.Helper()
	return NewProjectionProcessor(fx.app.db, fx.app.lexiconCatalog)
}

func TestDiscoveryListExcludesDeletedAndUnavailable(t *testing.T) {
	fx, _ := newPublicLinkFixture(t)
	p := newDiscoveryProjectionProcessor(t, fx)
	ctx := t.Context()

	did := "did:plc:discoverylistauthority01"
	activeURI := "at://" + did + "/tv.subcult.event.occurrence/act0001"
	deletedURI := "at://" + did + "/tv.subcult.event.occurrence/del0001"

	if _, err := p.ProcessEvent(ctx, StreamEvent{
		Cursor: "1", Kind: "commit", Operation: "create", DID: did,
		Collection: "tv.subcult.event.occurrence", RKey: "act0001", CID: "bafyact1", Rev: "3l1",
		Record: discoveryTestOccurrenceRecord("Active Night", "2026-10-01T20:00:00Z", ""),
	}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ProcessEvent(ctx, StreamEvent{
		Cursor: "2", Kind: "commit", Operation: "create", DID: did,
		Collection: "tv.subcult.event.occurrence", RKey: "del0001", CID: "bafydel1", Rev: "3l1",
		Record: discoveryTestOccurrenceRecord("Deleted Night", "2026-10-02T20:00:00Z", ""),
	}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ProcessEvent(ctx, StreamEvent{
		Cursor: "3", Kind: "commit", Operation: "delete", DID: did,
		Collection: "tv.subcult.event.occurrence", RKey: "del0001", Rev: "3l2",
	}, nil); err != nil {
		t.Fatal(err)
	}

	resp := getJSON(t, fx.app, nil, "/api/public/discovery/occurrences", http.StatusOK)
	items, ok := resp.JSON.([]any)
	if !ok {
		t.Fatalf("expected array response, got %T", resp.JSON)
	}
	foundActive, foundDeleted := false, false
	for _, item := range items {
		obj := mustObject(t, item)
		if obj["uri"] == activeURI {
			foundActive = true
		}
		if obj["uri"] == deletedURI {
			foundDeleted = true
		}
	}
	if !foundActive {
		t.Fatalf("expected active occurrence in list, body=%s", resp.Body)
	}
	if foundDeleted {
		t.Fatalf("deleted occurrence must not appear in list, body=%s", resp.Body)
	}

	// The detail route still returns the deleted occurrence, flagged.
	encodedTail := deletedURI[len("at://"):]
	detail := getJSON(t, fx.app, nil, "/api/public/discovery/occurrences/"+encodedTail, http.StatusOK)
	detailObj := mustObject(t, detail.JSON)
	if detailObj["projectionStatus"] != "deleted" {
		t.Fatalf("projectionStatus = %v, want deleted", detailObj["projectionStatus"])
	}
}

func TestDiscoveryDetailUnknownOccurrenceIsNotFound(t *testing.T) {
	fx, _ := newPublicLinkFixture(t)
	getJSON(t, fx.app, nil, "/api/public/discovery/occurrences/did:plc:nope/tv.subcult.event.occurrence/nope", http.StatusNotFound)
}

func TestDiscoveryDetailIncludesSafePublicLocation(t *testing.T) {
	fx, _ := newPublicLinkFixture(t)
	p := newDiscoveryProjectionProcessor(t, fx)
	ctx := t.Context()

	did := "did:plc:discoverylocationauth01"
	placeURI := "at://" + did + "/tv.subcult.place/place0001"
	occURI := "at://" + did + "/tv.subcult.event.occurrence/occ0001"

	if _, err := p.ProcessEvent(ctx, StreamEvent{
		Cursor: "1", Kind: "commit", Operation: "create", DID: did,
		Collection: "tv.subcult.place", RKey: "place0001", CID: "bafyplace1", Rev: "3l1",
		Record: discoveryTestPlaceRecord("The Venue", "Chicago", "IL", "US"),
	}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ProcessEvent(ctx, StreamEvent{
		Cursor: "2", Kind: "commit", Operation: "create", DID: did,
		Collection: "tv.subcult.event.occurrence", RKey: "occ0001", CID: "bafyocc1", Rev: "3l1",
		Record: discoveryTestOccurrenceRecord("Venue Night", "2026-10-01T20:00:00Z", placeURI),
	}, nil); err != nil {
		t.Fatal(err)
	}

	detail := getJSON(t, fx.app, nil, "/api/public/discovery/occurrences/"+occURI[len("at://"):], http.StatusOK)
	obj := mustObject(t, detail.JSON)
	location := mustObject(t, obj["location"])
	if location["name"] != "The Venue" || location["locality"] != "Chicago" {
		t.Fatalf("location = %v", location)
	}
	if location["latitude"] != "41.8781" {
		t.Fatalf("latitude = %v, want coarse public coordinate", location["latitude"])
	}
	source := mustObject(t, obj["source"])
	if source["did"] != did {
		t.Fatalf("source.did = %v, want %v", source["did"], did)
	}
}

// TestDiscoveryListFiltersByPlaceLocalityNotOccurrenceName proves the
// ?locality= filter matches on the occurrence's *place* locality, not just
// a substring of the occurrence's own display name: an occurrence whose
// name does not mention the locality at all must still be included when its
// place is actually there, and an occurrence whose place is in a different
// locality must be excluded even though it lives in the same collection.
func TestDiscoveryListFiltersByPlaceLocalityNotOccurrenceName(t *testing.T) {
	fx, _ := newPublicLinkFixture(t)
	p := newDiscoveryProjectionProcessor(t, fx)
	ctx := t.Context()

	did := "did:plc:discoverylocalityfilter01"
	chicagoPlaceURI := "at://" + did + "/tv.subcult.place/place0002"
	denverPlaceURI := "at://" + did + "/tv.subcult.place/place0003"
	// Deliberately named so the name substring match would NOT match
	// "Chicago", proving the place-locality join is what admits it.
	chicagoOccURI := "at://" + did + "/tv.subcult.event.occurrence/occ0004"
	denverOccURI := "at://" + did + "/tv.subcult.event.occurrence/occ0005"

	if _, err := p.ProcessEvent(ctx, StreamEvent{
		Cursor: "1", Kind: "commit", Operation: "create", DID: did,
		Collection: "tv.subcult.place", RKey: "place0002", CID: "bafyplace2", Rev: "3l1",
		Record: discoveryTestPlaceRecord("Signal Night Venue", "Chicago", "IL", "US"),
	}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ProcessEvent(ctx, StreamEvent{
		Cursor: "2", Kind: "commit", Operation: "create", DID: did,
		Collection: "tv.subcult.place", RKey: "place0003", CID: "bafyplace3", Rev: "3l1",
		Record: discoveryTestPlaceRecord("Denver Hall", "Denver", "CO", "US"),
	}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ProcessEvent(ctx, StreamEvent{
		Cursor: "3", Kind: "commit", Operation: "create", DID: did,
		Collection: "tv.subcult.event.occurrence", RKey: "occ0004", CID: "bafyocc4", Rev: "3l1",
		Record: discoveryTestOccurrenceRecord("Signal Night", "2026-10-03T20:00:00Z", chicagoPlaceURI),
	}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ProcessEvent(ctx, StreamEvent{
		Cursor: "4", Kind: "commit", Operation: "create", DID: did,
		Collection: "tv.subcult.event.occurrence", RKey: "occ0005", CID: "bafyocc5", Rev: "3l1",
		Record: discoveryTestOccurrenceRecord("Denver Night", "2026-10-04T20:00:00Z", denverPlaceURI),
	}, nil); err != nil {
		t.Fatal(err)
	}

	resp := getJSON(t, fx.app, nil, "/api/public/discovery/occurrences?locality=Chicago", http.StatusOK)
	items, ok := resp.JSON.([]any)
	if !ok {
		t.Fatalf("expected array response, got %T", resp.JSON)
	}
	foundChicago, foundDenver := false, false
	for _, item := range items {
		obj := mustObject(t, item)
		if obj["uri"] == chicagoOccURI {
			foundChicago = true
		}
		if obj["uri"] == denverOccURI {
			foundDenver = true
		}
	}
	if !foundChicago {
		t.Fatalf("expected occurrence at a Chicago place to be included even though its name doesn't mention Chicago, body=%s", resp.Body)
	}
	if foundDenver {
		t.Fatalf("occurrence at a Denver place must not match locality=Chicago, body=%s", resp.Body)
	}
}

func TestDiscoveryHandoffMissingMappingReturnsNone(t *testing.T) {
	fx, _ := newPublicLinkFixture(t)
	p := newDiscoveryProjectionProcessor(t, fx)
	ctx := t.Context()

	did := "did:plc:discoverynomapping0001"
	occURI := "at://" + did + "/tv.subcult.event.occurrence/occ0002"
	if _, err := p.ProcessEvent(ctx, StreamEvent{
		Cursor: "1", Kind: "commit", Operation: "create", DID: did,
		Collection: "tv.subcult.event.occurrence", RKey: "occ0002", CID: "bafyocc2", Rev: "3l1",
		Record: discoveryTestOccurrenceRecord("Unmapped Night", "2026-10-01T20:00:00Z", ""),
	}, nil); err != nil {
		t.Fatal(err)
	}

	detail := getJSON(t, fx.app, nil, "/api/public/discovery/occurrences/"+occURI[len("at://"):], http.StatusOK)
	obj := mustObject(t, detail.JSON)
	handoff := mustObject(t, obj["handoff"])
	if handoff["kind"] != "none" {
		t.Fatalf("handoff.kind = %v, want none", handoff["kind"])
	}
	if handoff["reason"] != "no_mapping" {
		t.Fatalf("handoff.reason = %v, want no_mapping", handoff["reason"])
	}
}

func TestDiscoveryHandoffStaleMappingReturnsNone(t *testing.T) {
	fx, fetcher := newPublicLinkFixture(t)
	p := newDiscoveryProjectionProcessor(t, fx)
	ctx := t.Context()

	event := createEvent(t, fx, "Stale Mapping Event", 100)
	eventID := mustString(t, event, "id")
	publishEvent(t, fx, eventID)

	did := "did:plc:discoverystalemapping01"
	occURI := "at://" + did + "/tv.subcult.event.occurrence/occ0003"
	if _, err := p.ProcessEvent(ctx, StreamEvent{
		Cursor: "1", Kind: "commit", Operation: "create", DID: did,
		Collection: "tv.subcult.event.occurrence", RKey: "occ0003", CID: "bafyocc3", Rev: "3l1",
		Record: discoveryTestOccurrenceRecord("Stale Night", "2026-10-01T20:00:00Z", ""),
	}, nil); err != nil {
		t.Fatal(err)
	}

	fetcher.setRecord(occURI, did, "stale.example.test", "bafyreicidstale")
	attach := postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/public-links", map[string]any{"publicUri": occURI}, http.StatusOK)
	linkID := mustString(t, attach.JSON, "id")

	// Make the link "unavailable" via refresh.
	fetcher.setError(occURI, fmt.Errorf("identity resolution failed"))
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventID+"/public-links/"+linkID+"/refresh", map[string]any{}, http.StatusOK)

	detail := getJSON(t, fx.app, nil, "/api/public/discovery/occurrences/"+occURI[len("at://"):], http.StatusOK)
	obj := mustObject(t, detail.JSON)
	handoff := mustObject(t, obj["handoff"])
	if handoff["kind"] != "none" {
		t.Fatalf("handoff.kind = %v, want none for a stale mapping", handoff["kind"])
	}
}

func TestDiscoveryHandoffResolvesLocalReservationAndNeverCrossesEvents(t *testing.T) {
	fx, fetcher := newPublicLinkFixture(t)
	p := newDiscoveryProjectionProcessor(t, fx)
	ctx := t.Context()

	// Two occurrences that deliberately SHARE the same public display name,
	// each correctly mapped to a *different* local event. Resolving one
	// must never accidentally route to the other's reservation destination.
	sharedTitle := "Recurring Series Night"
	did := "did:plc:discoveryambiguoustitle1"
	occURIA := "at://" + did + "/tv.subcult.event.occurrence/occA001"
	occURIB := "at://" + did + "/tv.subcult.event.occurrence/occB001"

	for i, uri := range []string{occURIA, occURIB} {
		rkey := "occA001"
		if i == 1 {
			rkey = "occB001"
		}
		if _, err := p.ProcessEvent(ctx, StreamEvent{
			Cursor: fmt.Sprintf("occ-%d", i), Kind: "commit", Operation: "create", DID: did,
			Collection: "tv.subcult.event.occurrence", RKey: rkey, CID: fmt.Sprintf("bafyocc%d", i), Rev: "3l1",
			Record: discoveryTestOccurrenceRecord(sharedTitle, "2026-10-0"+fmt.Sprint(i+1)+"T20:00:00Z", ""),
		}, nil); err != nil {
			t.Fatal(err)
		}
		_ = uri
	}

	eventA := mustString(t, createEvent(t, fx, "Series Night — Week A", 100), "id")
	eventB := mustString(t, createEvent(t, fx, "Series Night — Week B", 100), "id")
	publishedA := publishEvent(t, fx, eventA)
	publishedB := publishEvent(t, fx, eventB)
	slugA := mustString(t, publishedA, "publicSlug")
	slugB := mustString(t, publishedB, "publicSlug")

	fetcher.setRecord(occURIA, did, "series.example.test", "bafyreicidA")
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventA+"/public-links", map[string]any{"publicUri": occURIA}, http.StatusOK)

	fetcher.setRecord(occURIB, did, "series.example.test", "bafyreicidB")
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+eventB+"/public-links", map[string]any{"publicUri": occURIB}, http.StatusOK)

	detailA := getJSON(t, fx.app, nil, "/api/public/discovery/occurrences/"+occURIA[len("at://"):], http.StatusOK)
	detailB := getJSON(t, fx.app, nil, "/api/public/discovery/occurrences/"+occURIB[len("at://"):], http.StatusOK)

	objA := mustObject(t, detailA.JSON)
	objB := mustObject(t, detailB.JSON)
	if objA["name"] != sharedTitle || objB["name"] != sharedTitle {
		t.Fatalf("both occurrences must share the same display title for this test to be meaningful")
	}

	handoffA := mustObject(t, objA["handoff"])
	handoffB := mustObject(t, objB["handoff"])
	if handoffA["kind"] != "local" || handoffB["kind"] != "local" {
		t.Fatalf("expected both handoffs to resolve locally: A=%v B=%v", handoffA, handoffB)
	}
	if handoffA["eventSlug"] != slugA {
		t.Fatalf("occurrence A resolved to slug %v, want %v (its own event)", handoffA["eventSlug"], slugA)
	}
	if handoffB["eventSlug"] != slugB {
		t.Fatalf("occurrence B resolved to slug %v, want %v (its own event)", handoffB["eventSlug"], slugB)
	}
	if handoffA["eventSlug"] == handoffB["eventSlug"] {
		t.Fatalf("two distinct occurrences sharing a title must never resolve to the same event slug")
	}
}

func TestDiscoveryRoutesNeverLeakPrivacySentinels(t *testing.T) {
	fx := newCulturalFixture(t)
	fetcher := newFixturePublicLinkFetcher()
	fx.app.recordFetcher = fetcher
	p := newDiscoveryProjectionProcessor(t, fx)
	ctx := t.Context()

	sentinels := plantPrivacySentinels(t, fx)

	did := "did:plc:discoveryprivacysentinel1"
	occURI := "at://" + did + "/tv.subcult.event.occurrence/occ0009"
	if _, err := p.ProcessEvent(ctx, StreamEvent{
		Cursor: "1", Kind: "commit", Operation: "create", DID: did,
		Collection: "tv.subcult.event.occurrence", RKey: "occ0009", CID: "bafyocc9", Rev: "3l1",
		Record: discoveryTestOccurrenceRecord("Sentinel Discovery Night", "2026-10-01T20:00:00Z", ""),
	}, nil); err != nil {
		t.Fatal(err)
	}

	fetcher.setRecord(occURI, did, "sentinel.example.test", "bafyreicidsentinel")
	postJSON(t, fx.app, fx.ownerCookie, "/api/events/"+sentinels.eventID+"/public-links", map[string]any{"publicUri": occURI}, http.StatusOK)

	list := getJSON(t, fx.app, nil, "/api/public/discovery/occurrences", http.StatusOK)
	detail := getJSON(t, fx.app, nil, "/api/public/discovery/occurrences/"+occURI[len("at://"):], http.StatusOK)

	assertNoSentinelLeak(t, "discovery list", list.Body, sentinels.sentinels())
	assertNoSentinelLeak(t, "discovery detail", detail.Body, sentinels.sentinels())
}
