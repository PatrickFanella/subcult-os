package app

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type lifecycleNoticeDeliveryDTO struct {
	Email    string `json:"email"`
	Status   string `json:"status"`
	Attempts int    `json:"attempts"`
	Feedback string `json:"feedback"`
}
type lifecycleNoticeDTO struct {
	ID         string                       `json:"id"`
	ChangeID   string                       `json:"changeId"`
	Subject    string                       `json:"subject"`
	Body       string                       `json:"body"`
	QueuedAt   string                       `json:"queuedAt"`
	Recipients []lifecycleNoticeDeliveryDTO `json:"recipients"`
}

func (a *App) handleApproveLifecycleNotice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	if a.db == nil {
		writeError(w, 500, "database unavailable")
		return
	}
	var req struct {
		RequestKey  string   `json:"requestKey"`
		PreviewHash string   `json:"previewHash"`
		Audiences   []string `json:"audiences"`
	}
	if decodeJSON(r, &req) != nil || !validLifecycleNoticeAudiences(req.Audiences) {
		writeError(w, 400, "invalid notice approval")
		return
	}
	if _, err := uuid.Parse(req.RequestKey); err != nil {
		writeError(w, 400, "requestKey must be a UUID")
		return
	}
	digest, err := hex.DecodeString(req.PreviewHash)
	if err != nil || len(digest) != 32 {
		writeError(w, 400, "invalid previewHash")
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
	var lockedID, originalOwner string
	if err = tx.QueryRow(ctx, `select id from events where id=$1 and workspace_id=$2 for update`, event.ID, event.WorkspaceID).Scan(&lockedID); err != nil {
		writeNoticeError(w, err)
		return
	}
	err = tx.QueryRow(ctx, `select approved_by_person_id from event_lifecycle_changes where id=$1 and event_id=$2 and workspace_id=$3 for update`, r.PathValue("changeID"), event.ID, event.WorkspaceID).Scan(&originalOwner)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "lifecycle intent not found")
		return
	}
	if err != nil {
		writeNoticeError(w, err)
		return
	}
	actor, err := activeOwnerTx(ctx, tx, event.WorkspaceID, r.Context().Value(operatorPersonKey{}))
	if err != nil {
		writeError(w, 403, "forbidden")
		return
	}
	sort.Strings(req.Audiences)
	audiences, _ := json.Marshal(req.Audiences)
	// Resolve a lost response before testing the occurrence's newer revision.
	var noticeID string
	var same bool
	err = tx.QueryRow(ctx, `select id,change_id=$3 and approved_by_person_id=$4 and preview_hash=$5 and audiences=$6::jsonb from lifecycle_notices where workspace_id=$1 and request_key=$2`, event.WorkspaceID, req.RequestKey, r.PathValue("changeID"), actor, req.PreviewHash, audiences).Scan(&noticeID, &same)
	if err == nil {
		if !same {
			writeError(w, 409, "requestKey is bound to a different notice approval")
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
	if _, err = activeOwnerTx(ctx, tx, event.WorkspaceID, originalOwner); err != nil {
		writeError(w, 409, "original decision owner no longer has authority")
		return
	}
	preview, bindings, err := buildLifecycleNoticePreview(ctx, tx, event.ID, event.WorkspaceID, r.PathValue("changeID"), req.Audiences, true)
	if err != nil {
		writeNoticeError(w, err)
		return
	}
	if preview.PreviewHash != req.PreviewHash {
		writeError(w, 409, "notice content or recipients changed; preview again")
		return
	}
	var exists bool
	if err = tx.QueryRow(ctx, `select exists(select 1 from lifecycle_notices where workspace_id=$1 and (change_id=$2 or (occurrence_id=$3 and occurrence_updated_at=$4::timestamptz)))`, event.WorkspaceID, preview.ChangeID, preview.OccurrenceID, preview.Revision).Scan(&exists); err != nil {
		writeNoticeError(w, err)
		return
	}
	if exists {
		writeError(w, 409, "this listing revision already has a notice; inspect its outcomes before any correction")
		return
	}
	if len(preview.Recipients) == 0 {
		writeError(w, 409, "no operational recipients to notify")
		return
	}
	noticeID = uuid.NewString()
	actionID := uuid.NewString()
	payload, _ := json.Marshal(map[string]string{"noticeId": noticeID})
	_, err = tx.Exec(ctx, `insert into event_lifecycle_actions(id,change_id,action_kind,destination,idempotency_key,payload,dispatch_approved,status,attempt_count,finished_at) values($1,$2,'operational_notice','notice_email_outbox',$3,$4,true,'succeeded',1,clock_timestamp())`, actionID, preview.ChangeID, "lifecycle-notice:"+noticeID, payload)
	if err != nil {
		writeNoticeError(w, err)
		return
	}
	_, err = tx.Exec(ctx, `insert into lifecycle_notices(id,change_id,action_id,workspace_id,event_id,occurrence_id,occurrence_updated_at,public_cid,subject,body,audiences,preview_hash,request_key,approved_by_person_id) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, noticeID, preview.ChangeID, actionID, event.WorkspaceID, event.ID, preview.OccurrenceID, preview.Revision, preview.PublicCID, preview.Subject, preview.Body, audiences, preview.PreviewHash, req.RequestKey, actor)
	if err != nil {
		writeNoticeError(w, err)
		return
	}
	for i, recipient := range preview.Recipients {
		recipientID := uuid.NewString()
		var outboxID *string
		var withheld *string
		if recipient.Suppressed {
			reason := "recipient_suppressed"
			withheld = &reason
		} else {
			status := "held"
			if a.config.MailDeliveryEnabled {
				status = "pending"
			}
			var id string
			err = tx.QueryRow(ctx, `insert into email_outbox(recipient_email,subject,body,related_type,related_id,delivery_status,sender_address,reply_to_address,purpose,workspace_id,expires_at) values($1,$2,$3,'lifecycle_notice',$4,$5,$6,$7,'transactional',$8,now()+interval '23 hours') returning id::text`, recipient.Email, preview.Subject, preview.Body, recipientID, status, a.config.MailFrom, a.config.MailReplyTo, event.WorkspaceID).Scan(&id)
			if err != nil {
				writeNoticeError(w, err)
				return
			}
			outboxID = &id
		}
		_, err = tx.Exec(ctx, `insert into lifecycle_notice_recipients(id,notice_id,recipient_email,source_type,source_id,outbox_id,withheld_reason) values($1,$2,$3,$4,$5,$6,$7)`, recipientID, noticeID, recipient.Email, recipient.SourceType, bindings[i], outboxID, withheld)
		if err != nil {
			writeNoticeError(w, err)
			return
		}
	}
	if err = a.audit(ctx, actor, "lifecycle_notice.queued", "lifecycle_notice", noticeID, map[string]any{"workspaceId": event.WorkspaceID, "eventId": event.ID, "changeId": preview.ChangeID, "recipientCount": len(preview.Recipients)}); err != nil {
		writeNoticeError(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeNoticeError(w, err)
		return
	}
	a.respondLifecycleNotice(w, r, event.WorkspaceID, noticeID, 201)
}

func (a *App) handleGetLifecycleNotice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	if a.db == nil {
		writeError(w, 500, "database unavailable")
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
	if _, err := uuid.Parse(r.PathValue("changeID")); err != nil {
		writeError(w, 400, "invalid lifecycle intent ID")
		return
	}
	var id string
	err = a.db.QueryRow(r.Context(), `select id from lifecycle_notices where change_id=$1 and event_id=$2 and workspace_id=$3`, r.PathValue("changeID"), event.ID, event.WorkspaceID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "no notice has been queued")
		return
	}
	if err != nil {
		writeNoticeError(w, err)
		return
	}
	a.respondLifecycleNotice(w, r, event.WorkspaceID, id, 200)
}

func (a *App) respondLifecycleNotice(w http.ResponseWriter, r *http.Request, workspaceID, id string, status int) {
	var notice lifecycleNoticeDTO
	var queued time.Time
	err := a.db.QueryRow(r.Context(), `select id,change_id,subject,body,queued_at from lifecycle_notices where id=$1 and workspace_id=$2`, id, workspaceID).Scan(&notice.ID, &notice.ChangeID, &notice.Subject, &notice.Body, &queued)
	if err != nil {
		writeNoticeError(w, err)
		return
	}
	notice.QueuedAt = queued.UTC().Format(time.RFC3339Nano)
	notice.Recipients = []lifecycleNoticeDeliveryDTO{}
	rows, err := a.db.Query(r.Context(), `select nr.recipient_email,coalesce(e.delivery_status,'suppressed'),coalesce(e.attempts,0),case coalesce(e.feedback_rank,0) when 1 then 'delivered' when 2 then 'failed' when 3 then 'suppressed' when 4 then 'bounced' when 5 then 'complained' else 'unknown' end from lifecycle_notice_recipients nr left join email_outbox e on e.id=nr.outbox_id where nr.notice_id=$1 order by nr.recipient_email`, id)
	if err != nil {
		writeNoticeError(w, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var recipient lifecycleNoticeDeliveryDTO
		if err = rows.Scan(&recipient.Email, &recipient.Status, &recipient.Attempts, &recipient.Feedback); err != nil {
			writeNoticeError(w, err)
			return
		}
		notice.Recipients = append(notice.Recipients, recipient)
	}
	if rows.Err() != nil {
		writeNoticeError(w, rows.Err())
		return
	}
	rows.Close()
	if _, _, ok := a.requireWorkspaceRole(r, workspaceID, roleOwner); !ok {
		writeError(w, 403, "forbidden")
		return
	}
	writeJSON(w, status, notice)
}
