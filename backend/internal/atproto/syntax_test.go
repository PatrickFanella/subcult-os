package atproto

import "testing"

// Cases are independently reduced from current public AT Protocol syntax
// specifications, not copied from the unlicensed Subcults checkout.
func TestParseAccountIdentifier(t *testing.T) {
	tests := []struct {
		raw      string
		want     string
		wantKind IdentifierKind
		valid    bool
	}{
		{raw: "did:plc:vwzwgnygau7ed7b7wt5ux7y2", want: "did:plc:vwzwgnygau7ed7b7wt5ux7y2", wantKind: IdentifierDID, valid: true},
		{raw: "User.Example.COM", want: "user.example.com", wantKind: IdentifierHandle, valid: true},
		{raw: "@user.example.com"},
		{raw: "example.com:3000"},
		{raw: ""},
	}
	for _, test := range tests {
		t.Run(test.raw, func(t *testing.T) {
			got, err := ParseAccountIdentifier(test.raw)
			if !test.valid {
				if err == nil {
					t.Fatalf("ParseAccountIdentifier(%q) unexpectedly succeeded: %#v", test.raw, got)
				}
				return
			}
			if err != nil || got.Value != test.want || got.Kind != test.wantKind {
				t.Fatalf("ParseAccountIdentifier(%q) = %#v, %v", test.raw, got, err)
			}
		})
	}
}

func TestParseRecordRefRequiresExactRecord(t *testing.T) {
	valid := "at://did:plc:vwzwgnygau7ed7b7wt5ux7y2/APP.Bsky.feed.post/3k5nobkf2w72g"
	got, err := ParseRecordRef(valid)
	if err != nil {
		t.Fatal(err)
	}
	if got.URI != "at://did:plc:vwzwgnygau7ed7b7wt5ux7y2/app.bsky.feed.post/3k5nobkf2w72g" || got.Collection != "app.bsky.feed.post" || got.RecordKey != "3k5nobkf2w72g" {
		t.Fatalf("unexpected normalized record reference: %#v", got)
	}

	for _, invalid := range []string{
		"at://did:plc:vwzwgnygau7ed7b7wt5ux7y2",
		"at://did:plc:vwzwgnygau7ed7b7wt5ux7y2/app.bsky.feed.post",
		"at://foo.com/",
		"at://user:pass@foo.com/app.bsky.feed.post/key",
		"https://example.com/app.bsky.feed.post/key",
	} {
		if got, err := ParseRecordRef(invalid); err == nil {
			t.Fatalf("ParseRecordRef(%q) unexpectedly succeeded: %#v", invalid, got)
		}
	}
}

func TestParseStrongRefRequiresDIDAndCID(t *testing.T) {
	uri := "at://did:plc:vwzwgnygau7ed7b7wt5ux7y2/app.bsky.feed.post/3k5nobkf2w72g"
	cid := "bafyreievgu2y7e6bz6y7d6jv4yq6xldc5xkd4wqg5u6k2n3vhp4xqm4cfi"
	got, err := ParseStrongRef(uri, cid)
	if err != nil || got.CID != cid {
		t.Fatalf("ParseStrongRef() = %#v, %v", got, err)
	}
	if _, err := ParseStrongRef("at://retr0.id/app.bsky.feed.post/3k5nobkf2w72g", cid); err == nil {
		t.Fatal("handle-authority strong reference unexpectedly succeeded")
	}
	if _, err := ParseStrongRef(uri, "not-a-cid"); err == nil {
		t.Fatal("invalid CID unexpectedly succeeded")
	}
}

func TestParseCollection(t *testing.T) {
	got, err := ParseCollection("COM.Example.fooBar")
	if err != nil || got != "com.example.fooBar" {
		t.Fatalf("ParseCollection() = %q, %v", got, err)
	}
	for _, invalid := range []string{"com.example", "com.example.3", "com.exa💩ple.thing"} {
		if _, err := ParseCollection(invalid); err == nil {
			t.Fatalf("ParseCollection(%q) unexpectedly succeeded", invalid)
		}
	}
}
