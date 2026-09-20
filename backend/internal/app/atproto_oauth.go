package app

import "net/http"

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

func writeOAuthDocument(w http.ResponseWriter, document []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(document)
}
