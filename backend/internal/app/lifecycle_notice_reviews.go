package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type lifecycleNoticeReviewDTO struct {
	ID         string                       `json:"id"`
	Note       string                       `json:"note"`
	RecordedAt string                       `json:"recordedAt"`
	Recipients []lifecycleNoticeDeliveryDTO `json:"recipients"`
}

// Owners record observations; this endpoint neither retries nor resolves mail.
func (a *App) handleReviewLifecycleNotice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	if a.db == nil {
		writeError(w, 500, "database unavailable")
		return
	}
	var req struct {
		RequestKey string `json:"requestKey"`
		Note       string `json:"note"`
	}
	if decodeJSON(r, &req) != nil || !utf8.ValidString(req.Note) || strings.TrimSpace(req.Note) == "" || utf8.RuneCountInString(req.Note) > 2000 {
		writeError(w, 400, "a review note of 1 to 2000 characters is required")
		return
	}
	if _, err := uuid.Parse(req.RequestKey); err != nil {
		writeError(w, 400, "requestKey must be a UUID")
		return
	}
	if _, err := uuid.Parse(r.PathValue("changeID")); err != nil {
		writeError(w, 400, "invalid lifecycle intent ID")
		return
	}
	event, err := a.loadEventDetails(r.Context(), r.PathValue("eventID"))
	if err != nil {
		lifecycleIntentEventError(w, err)
		return
	}
	if _, _, ok := a.requireWorkspaceRole(r, event.WorkspaceID, roleOwner); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeNoticeError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	ctx := context.WithValue(r.Context(), txContextKey{}, tx)
	var lockedID string
	if err = tx.QueryRow(ctx, `select id from events where id=$1 and workspace_id=$2 for update`, event.ID, event.WorkspaceID).Scan(&lockedID); err != nil {
		writeNoticeError(w, err)
		return
	}
	actor, err := activeOwnerTx(ctx, tx, event.WorkspaceID, r.Context().Value(operatorPersonKey{}))
	if err != nil {
		writeError(w, 403, "forbidden")
		return
	}
	var noticeID string
	err = tx.QueryRow(ctx, `select id from lifecycle_notices where change_id=$1 and event_id=$2 and workspace_id=$3`, r.PathValue("changeID"), event.ID, event.WorkspaceID).Scan(&noticeID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "no notice has been queued")
		return
	}
	if err != nil {
		writeNoticeError(w, err)
		return
	}
	var same bool
	err = tx.QueryRow(ctx, `select notice_id=$2 and recorded_by_person_id=$3 and note=$4 from lifecycle_notice_reviews where request_key=$1`, req.RequestKey, noticeID, actor, req.Note).Scan(&same)
	if err == nil {
		if !same {
			writeError(w, 409, "requestKey already used for a different review")
			return
		}
		if err = tx.Commit(ctx); err != nil {
			writeNoticeError(w, err)
			return
		}
		a.respondLifecycleNotice(w, r, event.WorkspaceID, noticeID, 200)
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		writeNoticeError(w, err)
		return
	}
	// One SQL statement snapshots the server's recipient outcomes. A concurrent
	// worker may progress afterwards; the review remains a historical observation.
	var reviewID string
	err = tx.QueryRow(ctx, `insert into lifecycle_notice_reviews(notice_id,recorded_by_person_id,request_key,note,recipients)
	 select $1,$2,$3,$4,coalesce(jsonb_agg(jsonb_build_object('email',nr.recipient_email,'status',coalesce(e.delivery_status,'suppressed'),'attempts',coalesce(e.attempts,0),'feedback',case coalesce(e.feedback_rank,0) when 1 then 'delivered' when 2 then 'failed' when 3 then 'suppressed' when 4 then 'bounced' when 5 then 'complained' else 'unknown' end) order by nr.recipient_email),'[]'::jsonb)
	 from lifecycle_notice_recipients nr left join email_outbox e on e.id=nr.outbox_id where nr.notice_id=$1
	 on conflict(request_key) do nothing returning id`, noticeID, actor, req.RequestKey, req.Note).Scan(&reviewID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 409, "requestKey already used for a different review")
		return
	}
	if err != nil {
		writeNoticeError(w, err)
		return
	}
	if err = a.audit(ctx, actor, "lifecycle_notice.reviewed", "lifecycle_notice", noticeID, map[string]any{"reviewId": reviewID}); err != nil {
		writeNoticeError(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeNoticeError(w, err)
		return
	}
	a.respondLifecycleNotice(w, r, event.WorkspaceID, noticeID, 201)
}

func (a *App) loadLifecycleNoticeReviews(ctx context.Context, noticeID string) ([]lifecycleNoticeReviewDTO, error) {
	reviews := []lifecycleNoticeReviewDTO{}
	rows, err := a.db.Query(ctx, `select id,note,recorded_at,recipients from lifecycle_notice_reviews where notice_id=$1 order by recorded_at,id`, noticeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var review lifecycleNoticeReviewDTO
		var recorded time.Time
		var recipients []byte
		if err = rows.Scan(&review.ID, &review.Note, &recorded, &recipients); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(recipients, &review.Recipients); err != nil {
			return nil, err
		}
		review.RecordedAt = recorded.UTC().Format(time.RFC3339Nano)
		reviews = append(reviews, review)
	}
	return reviews, rows.Err()
}
