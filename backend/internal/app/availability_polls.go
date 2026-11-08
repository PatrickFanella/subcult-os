package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

const pollLinkHeader = "X-Poll-Link"
const pollEditHeader = "X-Poll-Edit"

type availabilityOption struct {
	ID       string  `json:"id"`
	StartsAt string  `json:"startsAt"`
	EndsAt   *string `json:"endsAt"`
	Timezone string  `json:"timezone"`
}
type availabilityResponse struct {
	DisplayName string            `json:"displayName"`
	PrivateNote string            `json:"privateNote,omitempty"`
	Answers     map[string]string `json:"answers"`
}
type availabilityPoll struct {
	ID             string                    `json:"id"`
	Prompt         string                    `json:"prompt"`
	Status         string                    `json:"status"`
	ShowNames      bool                      `json:"showNames"`
	ClosesAt       *string                   `json:"closesAt"`
	ChosenOptionID *string                   `json:"chosenOptionId"`
	Options        []availabilityOption      `json:"options"`
	Responses      []availabilityResponse    `json:"responses,omitempty"`
	Totals         map[string]map[string]int `json:"totals"`
	OwnResponse    *availabilityResponse     `json:"ownResponse,omitempty"`
}

type pollCreateRequest struct {
	Prompt    string     `json:"prompt"`
	ShowNames bool       `json:"showNames"`
	ClosesAt  *time.Time `json:"closesAt"`
	Options   []struct {
		StartsAt time.Time  `json:"startsAt"`
		EndsAt   *time.Time `json:"endsAt"`
		Timezone string     `json:"timezone"`
	} `json:"options"`
}

func parseEventEnd(input *string, previous sql.NullTime, start time.Time) (sql.NullTime, error) {
	end := previous
	if input != nil {
		end = sql.NullTime{}
		if strings.TrimSpace(*input) != "" {
			value, err := parseRFC3339Time(*input)
			if err != nil {
				return end, errors.New("invalid endsAt")
			}
			end = sql.NullTime{Time: value, Valid: true}
		}
	}
	if end.Valid && !end.Time.After(start) {
		return end, errors.New("endsAt must be after startsAt")
	}
	return end, nil
}

func (a *App) pollCoordinator(w http.ResponseWriter, r *http.Request) (eventRow, string, bool) {
	w.Header().Set("Cache-Control", "private, no-store")
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return eventRow{}, "", false
	}
	event, err := a.loadEventDetails(r.Context(), r.PathValue("eventID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "event not found")
		return eventRow{}, "", false
	}
	actor, _, ok := a.requireCoordinator(r, event.WorkspaceID)
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return eventRow{}, "", false
	}
	return event, actor, true
}

func (a *App) handleCreateAvailabilityPoll(w http.ResponseWriter, r *http.Request) {
	event, actor, ok := a.pollCoordinator(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var req pollCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Prompt = strings.TrimSpace(req.Prompt)
	if utf8.RuneCountInString(req.Prompt) < 1 || utf8.RuneCountInString(req.Prompt) > 500 || len(req.Options) < 2 || len(req.Options) > 12 || req.ClosesAt != nil && !req.ClosesAt.After(time.Now()) {
		writeError(w, http.StatusBadRequest, "a prompt, two to twelve options and an optional future closing time are required")
		return
	}
	seen := map[string]bool{}
	for _, option := range req.Options {
		zone, err := time.LoadLocation(option.Timezone)
		if err != nil || option.Timezone == "" || zone == time.Local || option.StartsAt.IsZero() || option.EndsAt != nil && !option.EndsAt.After(option.StartsAt) {
			writeError(w, http.StatusBadRequest, "options require valid instants, intervals and IANA timezones")
			return
		}
		key := option.StartsAt.UTC().Format(time.RFC3339Nano)
		if seen[key] {
			writeError(w, http.StatusBadRequest, "options must have distinct start times")
			return
		}
		seen[key] = true
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create poll")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	event, err = a.loadEventDetailsForUpdate(r.Context(), tx, event.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load event")
		return
	}
	if event.Status != eventStatusDraft {
		writeError(w, http.StatusConflict, "availability polls can only be created on draft events")
		return
	}
	token, hash, err := newToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create poll")
		return
	}
	var id string
	if err := tx.QueryRow(r.Context(), `insert into event_availability_polls(event_id,prompt,show_names,closes_at,token_hash,created_by_person_id) values($1,$2,$3,$4,$5,$6) returning id`, event.ID, req.Prompt, req.ShowNames, req.ClosesAt, hash, actor).Scan(&id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create poll")
		return
	}
	for i, option := range req.Options {
		if _, err := tx.Exec(r.Context(), `insert into event_availability_options(poll_id,sort_order,starts_at,ends_at,timezone) values($1,$2,$3,$4,$5)`, id, i, option.StartsAt, option.EndsAt, option.Timezone); err != nil {
			writeError(w, http.StatusInternalServerError, "could not create options")
			return
		}
	}
	if _, _, ok := a.requireCoordinator(r, event.WorkspaceID); !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err := a.audit(context.WithValue(r.Context(), txContextKey{}, tx), actor, "availability_poll.created", "availability_poll", id, map[string]any{"eventId": event.ID}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save poll")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id, "url": strings.TrimRight(a.config.PublicWebURL, "/") + "/poll/" + id + "#" + token})
}

func (a *App) handleListAvailabilityPolls(w http.ResponseWriter, r *http.Request) {
	event, _, ok := a.pollCoordinator(w, r)
	if !ok {
		return
	}
	rows, err := a.db.Query(r.Context(), `select id from event_availability_polls where event_id=$1 order by created_at`, event.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load polls")
		return
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			writeError(w, http.StatusInternalServerError, "could not load polls")
			return
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load polls")
		return
	}
	polls := []availabilityPoll{}
	for _, id := range ids {
		poll, err := a.readAvailabilityPoll(r.Context(), id, true, "")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load poll")
			return
		}
		polls = append(polls, poll)
	}
	if _, _, ok := a.requireCoordinator(r, event.WorkspaceID); !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	writeJSON(w, http.StatusOK, polls)
}

func (a *App) handleManageAvailabilityPoll(w http.ResponseWriter, r *http.Request) {
	event, actor, ok := a.pollCoordinator(w, r)
	if !ok {
		return
	}
	if !validUUID(r.PathValue("pollID")) {
		writeError(w, http.StatusNotFound, "poll not found")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not manage poll")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	event, err = a.loadEventDetailsForUpdate(r.Context(), tx, event.ID)
	if err != nil {
		writeError(w, http.StatusNotFound, "event not found")
		return
	}
	if event.Status != eventStatusDraft {
		writeError(w, http.StatusConflict, "use the lifecycle/reschedule flow for a published event")
		return
	}
	var id string
	if err := tx.QueryRow(r.Context(), `select id from event_availability_polls where id=$1 and event_id=$2 for update`, r.PathValue("pollID"), event.ID).Scan(&id); err != nil {
		writeError(w, http.StatusNotFound, "poll not found")
		return
	}
	action := "availability_poll.closed"
	if r.Method == http.MethodDelete {
		if _, err := tx.Exec(r.Context(), `delete from event_availability_polls where id=$1`, id); err != nil {
			writeError(w, http.StatusInternalServerError, "could not delete poll")
			return
		}
		action = "availability_poll.deleted"
	} else {
		if _, err := tx.Exec(r.Context(), `update event_availability_polls set status='closed' where id=$1`, id); err != nil {
			writeError(w, http.StatusInternalServerError, "could not close poll")
			return
		}
	}
	if _, _, ok := a.requireCoordinator(r, event.WorkspaceID); !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err := a.audit(context.WithValue(r.Context(), txContextKey{}, tx), actor, action, "availability_poll", id, map[string]any{"eventId": event.ID}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save poll")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) publicPollID(w http.ResponseWriter, r *http.Request) (string, bool) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.Header().Set("Referrer-Policy", "no-referrer")
	id := r.PathValue("pollID")
	token := r.Header.Get(pollLinkHeader)
	if a.db == nil || !validUUID(id) || len(token) < 32 || len(token) > 128 {
		writeError(w, http.StatusNotFound, "poll unavailable")
		return "", false
	}
	var found string
	// Keep access to closed polls for results and own answers, but never event data.
	if err := a.db.QueryRow(r.Context(), `select id from event_availability_polls where id=$1 and token_hash=$2`, id, tokenHash(token)).Scan(&found); err != nil {
		writeError(w, http.StatusNotFound, "poll unavailable")
		return "", false
	}
	return found, true
}

func (a *App) handlePublicAvailabilityPoll(w http.ResponseWriter, r *http.Request) {
	id, ok := a.publicPollID(w, r)
	if !ok {
		return
	}
	poll, err := a.readAvailabilityPoll(r.Context(), id, false, boundedPollEditToken(r.Header.Get(pollEditHeader)))
	if err != nil {
		writeError(w, http.StatusNotFound, "poll unavailable")
		return
	}
	writeJSON(w, http.StatusOK, poll)
}

func (a *App) handleAnswerAvailabilityPoll(w http.ResponseWriter, r *http.Request) {
	id, ok := a.publicPollID(w, r)
	if !ok {
		return
	}
	editToken := r.Header.Get(pollEditHeader)
	if len(editToken) < 32 || len(editToken) > 128 {
		writeError(w, http.StatusBadRequest, "edit capability is required")
		return
	}
	// Consume a persisted request budget before the content transaction so bad
	// answers cannot roll back the rate limit. Shared across all API processes.
	var count int
	err := a.db.QueryRow(r.Context(), `update event_availability_polls set
 rate_count=case when rate_window<=clock_timestamp()-interval '1 minute' then 1 else least(rate_count+1,31) end,
 rate_window=case when rate_window<=clock_timestamp()-interval '1 minute' then clock_timestamp() else rate_window end
 where id=$1 returning rate_count`, id).Scan(&count)
	if err != nil {
		writeError(w, http.StatusNotFound, "poll unavailable")
		return
	}
	if count > 30 {
		w.Header().Set("Retry-After", "60")
		writeError(w, http.StatusTooManyRequests, "too many responses; try again in a minute")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var req availabilityResponse
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.PrivateNote = strings.TrimSpace(req.PrivateNote)
	if utf8.RuneCountInString(req.DisplayName) < 1 || utf8.RuneCountInString(req.DisplayName) > 80 || utf8.RuneCountInString(req.PrivateNote) > 1000 || len(req.Answers) > 12 {
		writeError(w, http.StatusBadRequest, "use a name up to 80 characters and a note up to 1000 characters")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save response")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	var open bool
	if err := tx.QueryRow(r.Context(), `select status='open' and (closes_at is null or closes_at>clock_timestamp()) from event_availability_polls where id=$1 for update`, id).Scan(&open); err != nil {
		writeError(w, http.StatusNotFound, "poll unavailable")
		return
	}
	if !open {
		writeError(w, http.StatusConflict, "poll is closed")
		return
	}
	rows, err := tx.Query(r.Context(), `select id from event_availability_options where poll_id=$1`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load options")
		return
	}
	options := []string{}
	for rows.Next() {
		var option string
		if err := rows.Scan(&option); err != nil {
			rows.Close()
			writeError(w, http.StatusInternalServerError, "could not load options")
			return
		}
		options = append(options, option)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load options")
		return
	}
	if len(req.Answers) != len(options) {
		writeError(w, http.StatusBadRequest, "answer every poll option")
		return
	}
	for _, option := range options {
		answer := req.Answers[option]
		if answer != "yes" && answer != "if_need_be" && answer != "no" {
			writeError(w, http.StatusBadRequest, "each option needs yes, if need be, or no")
			return
		}
	}
	var responseID string
	err = tx.QueryRow(r.Context(), `select id from event_availability_responses where poll_id=$1 and edit_token_hash=$2`, id, tokenHash(editToken)).Scan(&responseID)
	if errors.Is(err, pgx.ErrNoRows) {
		var total int
		if err := tx.QueryRow(r.Context(), `select count(*) from event_availability_responses where poll_id=$1`, id).Scan(&total); err != nil {
			writeError(w, http.StatusInternalServerError, "could not save response")
			return
		}
		if total >= 200 {
			writeError(w, http.StatusConflict, "poll response limit reached")
			return
		}
		if err := tx.QueryRow(r.Context(), `insert into event_availability_responses(poll_id,edit_token_hash,display_name,private_note) values($1,$2,$3,$4) returning id`, id, tokenHash(editToken), req.DisplayName, req.PrivateNote).Scan(&responseID); err != nil {
			writeError(w, http.StatusConflict, "response capability conflicts with another poll")
			return
		}
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load response")
		return
	} else {
		if _, err := tx.Exec(r.Context(), `update event_availability_responses set display_name=$2,private_note=$3,updated_at=clock_timestamp() where id=$1`, responseID, req.DisplayName, req.PrivateNote); err != nil {
			writeError(w, http.StatusInternalServerError, "could not update response")
			return
		}
	}
	for _, option := range options {
		if _, err := tx.Exec(r.Context(), `insert into event_availability_answers(response_id,poll_id,option_id,answer) values($1,$2,$3,$4) on conflict(response_id,option_id) do update set answer=excluded.answer`, responseID, id, option, req.Answers[option]); err != nil {
			writeError(w, http.StatusInternalServerError, "could not save answers")
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save response")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"saved": true})
}

func (a *App) readAvailabilityPoll(ctx context.Context, id string, private bool, editToken string) (availabilityPoll, error) {
	poll := availabilityPoll{Options: []availabilityOption{}, Totals: map[string]map[string]int{}}
	tx, err := a.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return poll, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var closes sql.NullTime
	var chosen sql.NullString
	err = tx.QueryRow(ctx, `select id,prompt,case when status='closed' or closes_at<=clock_timestamp() then 'closed' else 'open' end,show_names,closes_at,chosen_option_id from event_availability_polls where id=$1`, id).Scan(&poll.ID, &poll.Prompt, &poll.Status, &poll.ShowNames, &closes, &chosen)
	if err != nil {
		return poll, err
	}
	poll.ClosesAt = nullableTimeString(closes)
	poll.ChosenOptionID = nullableString(chosen)
	rows, err := tx.Query(ctx, `select id,starts_at,ends_at,timezone from event_availability_options where poll_id=$1 order by sort_order`, id)
	if err != nil {
		return poll, err
	}
	for rows.Next() {
		var option availabilityOption
		var start time.Time
		var end sql.NullTime
		if err := rows.Scan(&option.ID, &start, &end, &option.Timezone); err != nil {
			rows.Close()
			return poll, err
		}
		option.StartsAt = start.UTC().Format(time.RFC3339Nano)
		option.EndsAt = nullableTimeString(end)
		poll.Options = append(poll.Options, option)
		poll.Totals[option.ID] = map[string]int{"yes": 0, "if_need_be": 0, "no": 0}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return poll, err
	}
	rows, err = tx.Query(ctx, `select r.id,r.display_name,r.private_note,r.edit_token_hash,a.option_id,a.answer from event_availability_responses r join event_availability_answers a on a.response_id=r.id where r.poll_id=$1 order by r.updated_at,r.id`, id)
	if err != nil {
		return poll, err
	}
	responses := map[string]*availabilityResponse{}
	order := []string{}
	for rows.Next() {
		var rid, name, note, hash, option, answer string
		if err := rows.Scan(&rid, &name, &note, &hash, &option, &answer); err != nil {
			rows.Close()
			return poll, err
		}
		poll.Totals[option][answer]++
		own := editToken != "" && tokenHash(editToken) == hash
		if private || poll.ShowNames || own {
			response := responses[rid]
			if response == nil {
				response = &availabilityResponse{DisplayName: name, Answers: map[string]string{}}
				if private || own {
					response.PrivateNote = note
				}
				responses[rid] = response
				order = append(order, rid)
			}
			response.Answers[option] = answer
			if own {
				poll.OwnResponse = response
			}
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return poll, err
	}
	if private || poll.ShowNames {
		for _, rid := range order {
			response := *responses[rid]
			if !private {
				response.PrivateNote = ""
			}
			poll.Responses = append(poll.Responses, response)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return poll, fmt.Errorf("read poll: %w", err)
	}
	return poll, nil
}

func boundedPollEditToken(value string) string {
	if len(value) < 32 || len(value) > 128 {
		return ""
	}
	return value
}
