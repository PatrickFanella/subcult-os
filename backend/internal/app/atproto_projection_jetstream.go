package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/websocket"
)

// JetstreamSource is the live StreamSource adapter: a Jetstream-shaped JSON
// commit-event websocket consumer. See docs/development/projection.md for
// why Jetstream's JSON format, not the raw firehose CAR/CBOR stream, was
// chosen as the smallest adapter the pinned Indigo supports for a consumer
// that only needs three collections.
//
// This adapter is deliberately minimal: it does not implement ping/pong
// keepalive, batching, or reconnect itself (ProjectionProcessor.Run owns
// reconnect/backoff by constructing a fresh JetstreamSource at the stored
// cursor). It is exercised by build-only tests; ProcessEvent/Run's own
// tests use MemoryStreamSource so no test opens a real network connection.
type JetstreamSource struct {
	conn *websocket.Conn
}

// jetstreamCommitEvent mirrors the public Jetstream wire schema
// (https://github.com/bluesky-social/jetstream), restricted to the fields
// this adapter consumes.
type jetstreamCommitEvent struct {
	DID    string `json:"did"`
	TimeUS int64  `json:"time_us"`
	Kind   string `json:"kind"`
	Commit *struct {
		Rev        string          `json:"rev"`
		Operation  string          `json:"operation"`
		Collection string          `json:"collection"`
		RKey       string          `json:"rkey"`
		Record     json.RawMessage `json:"record"`
		CID        string          `json:"cid"`
	} `json:"commit,omitempty"`
	Account *struct {
		Active bool   `json:"active"`
		DID    string `json:"did"`
		Status string `json:"status,omitempty"`
	} `json:"account,omitempty"`
}

// NewJetstreamSource dials sourceURL (a Jetstream `wss://` subscribe
// endpoint), requesting only wantedCollections and resuming from cursor
// (Jetstream's "cursor" query parameter is a microsecond Unix timestamp; an
// empty cursor subscribes from the live tail).
func NewJetstreamSource(sourceURL, cursor string, wantedCollections []string) (*JetstreamSource, error) {
	parsed, err := url.Parse(sourceURL)
	if err != nil {
		return nil, fmt.Errorf("parse projection source URL: %w", err)
	}
	query := parsed.Query()
	query.Del("wantedCollections")
	for _, collection := range wantedCollections {
		query.Add("wantedCollections", collection)
	}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	parsed.RawQuery = query.Encode()

	origin := "https://" + parsed.Host
	conn, err := websocket.Dial(parsed.String(), "", origin)
	if err != nil {
		return nil, fmt.Errorf("dial jetstream source: %w", err)
	}
	return &JetstreamSource{conn: conn}, nil
}

// Next blocks until the next commit/account event, or ctx/the connection
// ends. Identity-only events (handle/signing-key changes) and info frames
// carry no admitted-collection data and are skipped internally.
func (j *JetstreamSource) Next(ctx context.Context) (StreamEvent, error) {
	for {
		if err := ctx.Err(); err != nil {
			return StreamEvent{}, err
		}
		var raw jetstreamCommitEvent
		if err := websocket.JSON.Receive(j.conn, &raw); err != nil {
			return StreamEvent{}, fmt.Errorf("receive jetstream event: %w", err)
		}
		cursor := fmt.Sprintf("%d", raw.TimeUS)
		switch strings.ToLower(raw.Kind) {
		case "commit":
			if raw.Commit == nil {
				continue
			}
			return StreamEvent{
				Cursor:     cursor,
				DID:        raw.DID,
				Kind:       "commit",
				Operation:  raw.Commit.Operation,
				Collection: raw.Commit.Collection,
				RKey:       raw.Commit.RKey,
				CID:        raw.Commit.CID,
				Rev:        raw.Commit.Rev,
				Record:     raw.Commit.Record,
			}, nil
		case "account":
			if raw.Account == nil {
				continue
			}
			status := raw.Account.Status
			if status == "" {
				if raw.Account.Active {
					status = "active"
				} else {
					status = "deactivated"
				}
			} else if raw.Account.Active {
				status = "active"
			}
			return StreamEvent{
				Cursor:        cursor,
				DID:           raw.Account.DID,
				Kind:          "account",
				AccountStatus: status,
			}, nil
		default:
			// "identity" and unrecognized kinds carry nothing an admitted
			// collection needs; skip to the next frame.
			continue
		}
	}
}

// Close closes the underlying websocket connection.
func (j *JetstreamSource) Close() error {
	if j.conn == nil {
		return nil
	}
	return j.conn.Close()
}
