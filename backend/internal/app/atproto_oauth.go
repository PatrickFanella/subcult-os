package app

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	atprotocol "git.subcult.tv/PatrickFanella/subcult-os/internal/atproto"
)

type atprotoLinkFlow interface {
	StartLink(context.Context, string, string) (string, error)
	CompleteLink(context.Context, url.Values) (atprotocol.OAuthLinkResult, error)
}

type atprotoStartRequest struct {
	Identifier string `json:"identifier"`
}

type atprotoStartResponse struct {
	AuthorizationURL string `json:"authorizationUrl"`
}

func (a *App) handleATProtoClientMetadata(w http.ResponseWriter, r *http.Request) {
	if !a.atprotoOAuthAvailable(w, r) {
		return
	}
	writeOAuthDocument(w, a.atprotoOAuth.MetadataJSON())
}

func (a *App) handleATProtoJWKS(w http.ResponseWriter, r *http.Request) {
	if !a.atprotoOAuthAvailable(w, r) {
		return
	}
	writeOAuthDocument(w, a.atprotoOAuth.JWKSJSON())
}

func (a *App) handleATProtoStart(w http.ResponseWriter, r *http.Request) {
	if !a.atprotoFlowAvailable(w, r) {
		return
	}
	personID, ok := a.requirePersonID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var request atprotoStartRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	identifier, err := atprotocol.ParseAccountIdentifier(strings.TrimSpace(request.Identifier))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid AT Protocol identifier")
		return
	}
	authorizationURL, err := a.atprotoFlow.StartLink(r.Context(), personID, identifier.Value)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not start AT Protocol authorization")
		return
	}
	writeJSON(w, http.StatusOK, atprotoStartResponse{AuthorizationURL: authorizationURL})
}

func (a *App) handleATProtoCallback(w http.ResponseWriter, r *http.Request) {
	if !a.atprotoFlowAvailable(w, r) {
		return
	}
	status := "error"
	if _, err := a.atprotoFlow.CompleteLink(r.Context(), r.URL.Query()); err == nil {
		status = "linked"
	} else if errors.Is(err, atprotocol.ErrOAuthDenied) {
		status = "cancelled"
	}
	landing := strings.TrimRight(a.config.PublicWebURL, "/") + "/workspace?atproto=" + url.QueryEscape(status)
	http.Redirect(w, r, landing, http.StatusSeeOther)
}

func (a *App) atprotoOAuthAvailable(w http.ResponseWriter, r *http.Request) bool {
	if !a.config.ATProtoOAuthEnabled {
		http.NotFound(w, r)
		return false
	}
	if a.atprotoErr != nil || a.atprotoOAuth == nil {
		writeError(w, http.StatusServiceUnavailable, "AT OAuth unavailable")
		return false
	}
	return true
}

func (a *App) atprotoFlowAvailable(w http.ResponseWriter, r *http.Request) bool {
	if !a.config.ATProtoOAuthEnabled {
		http.NotFound(w, r)
		return false
	}
	if a.atprotoErr != nil || a.atprotoFlowErr != nil || a.atprotoFlow == nil {
		writeError(w, http.StatusServiceUnavailable, "AT OAuth unavailable")
		return false
	}
	return true
}

func writeOAuthDocument(w http.ResponseWriter, document []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(document)
}
