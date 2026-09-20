package app

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

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

type atprotoLinkDTO struct {
	DID        string    `json:"did"`
	Handle     *string   `json:"handle"`
	VerifiedAt time.Time `json:"verifiedAt"`
}

type atprotoLinksResponse struct {
	Links []atprotoLinkDTO `json:"links"`
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

func (a *App) handleATProtoLinks(w http.ResponseWriter, r *http.Request) {
	if !a.atprotoLinkStoreAvailable(w, r) {
		return
	}
	personID, ok := a.requirePersonID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	links, err := a.atprotoStore.ListActiveLinks(r.Context(), personID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load AT Protocol links")
		return
	}
	response := atprotoLinksResponse{Links: make([]atprotoLinkDTO, 0, len(links))}
	for _, link := range links {
		var handle *string
		if link.Handle != "" {
			value := link.Handle
			handle = &value
		}
		response.Links = append(response.Links, atprotoLinkDTO{DID: link.DID, Handle: handle, VerifiedAt: link.VerifiedAt})
	}
	writeJSON(w, http.StatusOK, response)
}

func (a *App) handleATProtoUnlink(w http.ResponseWriter, r *http.Request) {
	if !a.atprotoLinkStoreAvailable(w, r) {
		return
	}
	personID, ok := a.requirePersonID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := a.atprotoStore.RevokeLocalLink(r.Context(), personID, r.PathValue("did")); err != nil {
		if errors.Is(err, atprotocol.ErrOAuthInvalidDID) {
			writeError(w, http.StatusBadRequest, "invalid AT Protocol DID")
			return
		}
		if errors.Is(err, atprotocol.ErrOAuthLinkNotFound) {
			writeError(w, http.StatusNotFound, "AT Protocol link not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not unlink AT Protocol identity")
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

func (a *App) atprotoLinkStoreAvailable(w http.ResponseWriter, r *http.Request) bool {
	if !a.config.ATProtoOAuthEnabled {
		http.NotFound(w, r)
		return false
	}
	if a.atprotoErr != nil || a.atprotoFlowErr != nil || a.atprotoStore == nil {
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
