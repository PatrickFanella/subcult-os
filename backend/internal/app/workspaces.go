package app

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

type workspaceDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type memberDTO struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName *string `json:"displayName"`
	Role        string  `json:"role"`
}

type invitationDTO struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName *string `json:"displayName,omitempty"`
	Role        string  `json:"role"`
	Token       string  `json:"token,omitempty"`
	AcceptedAt  *string `json:"acceptedAt"`
}

type currentWorkspaceDTO struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Role        string          `json:"role"`
	Members     []memberDTO     `json:"members"`
	Invitations []invitationDTO `json:"invitations"`
}

type workspaceArchiveSummaryDTO struct {
	ID              string  `json:"id"`
	EventID         string  `json:"eventId"`
	Title           string  `json:"title"`
	StartsAt        string  `json:"startsAt"`
	LocationDisplay string  `json:"locationDisplay"`
	NoteCount       int     `json:"noteCount"`
	ReportID        string  `json:"reportId"`
	SettlementID    string  `json:"settlementId"`
	SeededEventID   *string `json:"seededEventId,omitempty"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

type createWorkspaceRequest struct {
	Name string `json:"name"`
}

type createInvitationRequest struct {
	Email string `json:"email"`
}

func (a *App) requireWorkspaceRole(r *http.Request, workspaceID string, allowed ...string) (personID string, role string, ok bool) {
	personID, ok = a.requirePersonID(r)
	if !ok || a.db == nil {
		return "", "", false
	}
	if workspaceID == "" {
		return "", "", false
	}

	var membershipRole string
	err := a.db.QueryRow(r.Context(), `
		select role
		from workspace_members
		where workspace_id = $1
		  and person_id = $2
		  and removed_at is null
	`, workspaceID, personID).Scan(&membershipRole)
	if err != nil {
		return "", "", false
	}

	if len(allowed) == 0 {
		return personID, membershipRole, true
	}
	for _, want := range allowed {
		if membershipRole == want {
			return personID, membershipRole, true
		}
	}
	return "", "", false
}

func (a *App) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	personID, ok := a.requirePersonID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createWorkspaceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var workspaceID string
	if err := tx.QueryRow(r.Context(), `
		insert into workspaces (name)
		values ($1)
		returning id
	`, name).Scan(&workspaceID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create workspace")
		return
	}
	if _, err := tx.Exec(r.Context(), `
		insert into workspace_members (workspace_id, person_id, role)
		values ($1, $2, 'owner')
	`, workspaceID, personID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not add owner")
		return
	}
	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, personID, "workspace.created", "workspace", workspaceID, map[string]any{
		"name": name,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save workspace")
		return
	}

	writeJSON(w, http.StatusOK, workspaceDTO{ID: workspaceID, Name: name, Role: "owner"})
}

func (a *App) handleCurrentWorkspace(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	personID, ok := a.requirePersonID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	workspace, err := a.loadCurrentWorkspace(r.Context(), personID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "workspace not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load workspace")
		return
	}

	writeJSON(w, http.StatusOK, workspace)
}

func (a *App) handleGetWorkspace(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	personID, ok := a.requirePersonID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	workspaceID := r.PathValue("workspaceID")
	workspace, err := a.loadWorkspace(r.Context(), personID, workspaceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load workspace")
		return
	}

	writeJSON(w, http.StatusOK, workspace)
}

func (a *App) handleListWorkspaceArchives(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	workspaceID := r.PathValue("workspaceID")
	_, _, ok := a.requireWorkspaceRole(r, workspaceID, "owner", "member")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	query := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("q")))
	if utf8.RuneCountInString(query) > 120 {
		writeError(w, http.StatusBadRequest, "search query is too long")
		return
	}

	archives, err := a.listWorkspaceArchives(r.Context(), workspaceID, query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load archives")
		return
	}

	writeJSON(w, http.StatusOK, archives)
}

func (a *App) loadCurrentWorkspace(ctx context.Context, personID string) (currentWorkspaceDTO, error) {
	var workspace currentWorkspaceDTO
	err := a.db.QueryRow(ctx, `
		select w.id, w.name, wm.role
		from workspace_members wm
		join workspaces w on w.id = wm.workspace_id
		where wm.person_id = $1
		  and wm.removed_at is null
		order by w.created_at desc, w.name
		limit 1
	`, personID).Scan(&workspace.ID, &workspace.Name, &workspace.Role)
	if err != nil {
		return currentWorkspaceDTO{}, err
	}
	return a.hydrateWorkspace(ctx, workspace)
}

func (a *App) loadWorkspace(ctx context.Context, personID string, workspaceID string) (currentWorkspaceDTO, error) {
	var workspace currentWorkspaceDTO
	err := a.db.QueryRow(ctx, `
		select w.id, w.name, wm.role
		from workspace_members wm
		join workspaces w on w.id = wm.workspace_id
		where wm.person_id = $1
		  and wm.workspace_id = $2
		  and wm.removed_at is null
	`, personID, workspaceID).Scan(&workspace.ID, &workspace.Name, &workspace.Role)
	if err != nil {
		return currentWorkspaceDTO{}, err
	}
	return a.hydrateWorkspace(ctx, workspace)
}

func (a *App) hydrateWorkspace(ctx context.Context, workspace currentWorkspaceDTO) (currentWorkspaceDTO, error) {
	members, err := a.listWorkspaceMembers(ctx, workspace.ID)
	if err != nil {
		return currentWorkspaceDTO{}, err
	}
	workspace.Members = members

	invites, err := a.listWorkspaceInvitations(ctx, workspace.ID)
	if err != nil {
		return currentWorkspaceDTO{}, err
	}
	workspace.Invitations = invites
	return workspace, nil
}

func (a *App) listWorkspaceArchives(ctx context.Context, workspaceID string, query string) ([]workspaceArchiveSummaryDTO, error) {
	baseQuery := `
		select ea.id, ea.event_id, e.title, e.starts_at, e.location_display, ea.note_count,
		       ea.report_id, ea.settlement_id, ea.seeded_event_id, ea.created_at, ea.updated_at
		from event_archives ea
		join events e on e.id = ea.event_id
		where e.workspace_id = $1
		order by e.starts_at desc, ea.created_at desc
		limit 100
	`
	searchQuery := `
		select ea.id, ea.event_id, e.title, e.starts_at, e.location_display, ea.note_count,
		       ea.report_id, ea.settlement_id, ea.seeded_event_id, ea.created_at, ea.updated_at
		from event_archives ea
		join events e on e.id = ea.event_id
		where e.workspace_id = $1
		  and (
		    position($2 in lower(e.title)) > 0
		    or position($2 in lower(coalesce(e.public_description, ''))) > 0
		    or position($2 in lower(coalesce(e.location_display, ''))) > 0
		    or exists (
		      select 1
		      from event_archive_notes ean
		      where ean.archive_id = ea.id
		        and position($2 in lower(ean.body)) > 0
		    )
		  )
		order by e.starts_at desc, ea.created_at desc
		limit 100
	`

	var (
		rows pgx.Rows
		err  error
	)
	if query == "" {
		rows, err = a.db.Query(ctx, baseQuery, workspaceID)
	} else {
		rows, err = a.db.Query(ctx, searchQuery, workspaceID, query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	archives := make([]workspaceArchiveSummaryDTO, 0)
	for rows.Next() {
		var archive workspaceArchiveSummaryDTO
		var startsAt time.Time
		var seededEventID sql.NullString
		var createdAt time.Time
		var updatedAt time.Time
		if err := rows.Scan(&archive.ID, &archive.EventID, &archive.Title, &startsAt, &archive.LocationDisplay, &archive.NoteCount, &archive.ReportID, &archive.SettlementID, &seededEventID, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		archive.StartsAt = startsAt.UTC().Format(time.RFC3339Nano)
		archive.SeededEventID = nullableString(seededEventID)
		archive.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		archive.UpdatedAt = updatedAt.UTC().Format(time.RFC3339Nano)
		archives = append(archives, archive)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return archives, nil
}

func (a *App) handleCreateInvitation(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	workspaceID := r.PathValue("workspaceID")
	personID, _, ok := a.requireWorkspaceRole(r, workspaceID, "owner")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req createInvitationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	email := normalizeEmail(req.Email)
	if !strings.Contains(email, "@") {
		writeError(w, http.StatusBadRequest, "invalid email")
		return
	}

	token, tokenHash, err := newToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create invitation token")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var invitationID string
	if err := tx.QueryRow(r.Context(), `
		insert into workspace_invitations (workspace_id, email, role, token_hash, invited_by_person_id)
		values ($1, $2, 'member', $3, $4)
		returning id
	`, workspaceID, email, tokenHash, personID).Scan(&invitationID); err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "invitation already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not create invitation")
		return
	}

	body := "You're invited to Signal Collective on subcult-os\n\nAccept your invitation here: /invite/" + token
	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, personID, "workspace.invitation.created", "workspace_invitation", invitationID, map[string]any{
		"workspaceId": workspaceID,
		"email":       email,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := a.enqueueEmail(txCtx, email, "You're invited to Signal Collective on subcult-os", body, "workspace_invitation", invitationID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not enqueue invitation email")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save invitation")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":          invitationID,
		"workspaceId": workspaceID,
		"email":       email,
		"role":        "member",
		"token":       token,
	})
}

func (a *App) handleAcceptInvitation(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	personID, ok := a.requirePersonID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	token := r.PathValue("token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var invitation struct {
		ID          string
		WorkspaceID string
		Email       string
		Role        string
		AcceptedAt  sql.NullTime
	}
	err = tx.QueryRow(r.Context(), `
		select id, workspace_id, email, role, accepted_at
		from workspace_invitations
		where token_hash = $1
	`, tokenHash(token)).Scan(&invitation.ID, &invitation.WorkspaceID, &invitation.Email, &invitation.Role, &invitation.AcceptedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "invitation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load invitation")
		return
	}
	if invitation.AcceptedAt.Valid {
		writeError(w, http.StatusConflict, "invitation already accepted")
		return
	}

	var personEmail string
	if err := tx.QueryRow(r.Context(), `
		select email
		from people
		where id = $1
	`, personID).Scan(&personEmail); err != nil {
		writeError(w, http.StatusInternalServerError, "could not load user")
		return
	}
	if normalizeEmail(personEmail) != normalizeEmail(invitation.Email) {
		writeError(w, http.StatusForbidden, "invitation email mismatch")
		return
	}

	var existingMembership struct {
		ID        string
		Role      string
		RemovedAt sql.NullTime
	}
	err = tx.QueryRow(r.Context(), `
		select id, role, removed_at
		from workspace_members
		where workspace_id = $1
		  and person_id = $2
		for update
	`, invitation.WorkspaceID, personID).Scan(&existingMembership.ID, &existingMembership.Role, &existingMembership.RemovedAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "could not load membership")
		return
	}
	if err == nil && !existingMembership.RemovedAt.Valid {
		writeError(w, http.StatusConflict, "user is already a workspace member")
		return
	}

	if result, err := tx.Exec(r.Context(), `
		update workspace_invitations
		set accepted_at = now()
		where id = $1
		  and accepted_at is null
	`, invitation.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not accept invitation")
		return
	} else if result.RowsAffected() == 0 {
		writeError(w, http.StatusConflict, "invitation already accepted")
		return
	}
	if existingMembership.ID != "" {
		if _, err := tx.Exec(r.Context(), `
			update workspace_members
			set removed_at = null
			where id = $1
		`, existingMembership.ID); err != nil {
			writeError(w, http.StatusInternalServerError, "could not reactivate workspace member")
			return
		}
	} else if _, err := tx.Exec(r.Context(), `
		insert into workspace_members (workspace_id, person_id, role, removed_at)
		values ($1, $2, $3, null)
		on conflict (workspace_id, person_id) do update
		set removed_at = null
	`, invitation.WorkspaceID, personID, invitation.Role); err != nil {
		writeError(w, http.StatusInternalServerError, "could not add workspace member")
		return
	}
	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, personID, "workspace.invitation.accepted", "workspace_invitation", invitation.ID, map[string]any{
		"workspaceId": invitation.WorkspaceID,
		"email":       invitation.Email,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save invitation acceptance")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	workspaceID := r.PathValue("workspaceID")
	actorID, _, ok := a.requireWorkspaceRole(r, workspaceID, "owner")
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	memberID := r.PathValue("memberID")
	if memberID == "" {
		writeError(w, http.StatusBadRequest, "member id is required")
		return
	}
	if memberID == actorID {
		// Still allow self-removal only if there is another owner.
		// The ownership check below enforces the last-owner guard.
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	if err := tx.QueryRow(r.Context(), `
		select id
		from workspaces
		where id = $1
		for update
	`, workspaceID).Scan(new(string)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "workspace not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not lock workspace")
		return
	}

	var targetRole string
	var targetRemoved sql.NullTime
	err = tx.QueryRow(r.Context(), `
		select role, removed_at
		from workspace_members
		where workspace_id = $1
		  and id = $2
	`, workspaceID, memberID).Scan(&targetRole, &targetRemoved)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "member not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load member")
		return
	}
	if targetRemoved.Valid {
		writeError(w, http.StatusNotFound, "member not found")
		return
	}

	if targetRole == "owner" {
		var owners int
		if err := tx.QueryRow(r.Context(), `
			select count(*)
			from workspace_members
			where workspace_id = $1
			  and role = 'owner'
			  and removed_at is null
		`, workspaceID).Scan(&owners); err != nil {
			writeError(w, http.StatusInternalServerError, "could not count owners")
			return
		}
		if owners <= 1 {
			writeError(w, http.StatusConflict, "cannot remove last owner")
			return
		}
	}

	if _, err := tx.Exec(r.Context(), `
		update workspace_members
		set removed_at = now()
		where id = $1
		  and removed_at is null
	`, memberID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not remove member")
		return
	}
	txCtx := context.WithValue(r.Context(), txContextKey{}, tx)
	if err := a.audit(txCtx, actorID, "workspace.member.removed", "workspace_member", memberID, map[string]any{
		"workspaceId": workspaceID,
		"role":        targetRole,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record audit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save member removal")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) listWorkspaceMembers(ctx context.Context, workspaceID string) ([]memberDTO, error) {
	rows, err := a.db.Query(ctx, `
		select wm.id, p.email, p.display_name, wm.role
		from workspace_members wm
		join people p on p.id = wm.person_id
		where wm.workspace_id = $1
		  and wm.removed_at is null
		order by wm.created_at, p.email
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]memberDTO, 0)
	for rows.Next() {
		var id, email, role string
		var displayName sql.NullString
		if err := rows.Scan(&id, &email, &displayName, &role); err != nil {
			return nil, err
		}
		item := memberDTO{ID: id, Email: email, Role: role}
		if displayName.Valid {
			item.DisplayName = &displayName.String
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (a *App) listWorkspaceInvitations(ctx context.Context, workspaceID string) ([]invitationDTO, error) {
	rows, err := a.db.Query(ctx, `
		select id, email, role, accepted_at
		from workspace_invitations
		where workspace_id = $1
		order by created_at
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]invitationDTO, 0)
	for rows.Next() {
		var id, email, role string
		var acceptedAt sql.NullTime
		if err := rows.Scan(&id, &email, &role, &acceptedAt); err != nil {
			return nil, err
		}
		item := invitationDTO{ID: id, Email: email, Role: role}
		if acceptedAt.Valid {
			v := acceptedAt.Time.UTC().Format(time.RFC3339Nano)
			item.AcceptedAt = &v
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
