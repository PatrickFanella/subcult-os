package app

import (
	"net/http"
	"regexp"
	"strings"
)

// appLinkPaths are the only web paths the native app claims. Identity email
// links open the app when it is installed and fall back to the web pages
// otherwise; every other path stays in the browser.
var appLinkPaths = []string{"/verify-email", "/recover-password"}

var (
	appleAppIDPattern         = regexp.MustCompile(`^[A-Z0-9]{10}\.[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)+$`)
	androidPackagePattern     = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)+$`)
	androidFingerprintPattern = regexp.MustCompile(`^([0-9A-F]{2}:){31}[0-9A-F]{2}$`)
)

func parseCommaList(raw string, normalize func(string) string) []string {
	var values []string
	for _, value := range strings.Split(raw, ",") {
		if value = normalize(strings.TrimSpace(value)); value != "" {
			values = append(values, value)
		}
	}
	return values
}

func (c Config) appLinkProblems() []string {
	var problems []string
	for _, id := range c.MobileAppleAppIDs {
		if !appleAppIDPattern.MatchString(id) {
			problems = append(problems, "MOBILE_APPLE_APP_IDS must contain TEAMID.bundle.identifier values")
			break
		}
	}
	if c.MobileAndroidPackage != "" && !androidPackagePattern.MatchString(c.MobileAndroidPackage) {
		problems = append(problems, "MOBILE_ANDROID_PACKAGE must be an Android application ID")
	}
	for _, fingerprint := range c.MobileAndroidCertFingerprints {
		if !androidFingerprintPattern.MatchString(fingerprint) {
			problems = append(problems, "MOBILE_ANDROID_CERT_SHA256 must contain colon-separated SHA-256 fingerprints")
			break
		}
	}
	if (c.MobileAndroidPackage == "") != (len(c.MobileAndroidCertFingerprints) == 0) {
		problems = append(problems, "MOBILE_ANDROID_PACKAGE and MOBILE_ANDROID_CERT_SHA256 must be set together")
	}
	return problems
}

func writeAppLinkDocument(w http.ResponseWriter, document any) {
	// Apple and Google fetch these without redirects and cache them; keep the
	// cache short so a signing-key change reaches devices within the hour.
	w.Header().Set("Cache-Control", "public, max-age=3600")
	writeJSON(w, http.StatusOK, document)
}

func (a *App) handleAppleAppSiteAssociation(w http.ResponseWriter, r *http.Request) {
	if len(a.config.MobileAppleAppIDs) == 0 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	components := make([]map[string]any, 0, len(appLinkPaths))
	for _, path := range appLinkPaths {
		components = append(components, map[string]any{"/": path, "?": map[string]string{"token": "?*"}})
	}
	writeAppLinkDocument(w, map[string]any{
		"applinks": map[string]any{
			"details": []map[string]any{{"appIDs": a.config.MobileAppleAppIDs, "components": components}},
		},
	})
}

func (a *App) handleAndroidAssetLinks(w http.ResponseWriter, r *http.Request) {
	if a.config.MobileAndroidPackage == "" || len(a.config.MobileAndroidCertFingerprints) == 0 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeAppLinkDocument(w, []map[string]any{{
		"relation": []string{"delegate_permission/common.handle_all_urls"},
		"target": map[string]any{
			"namespace":                "android_app",
			"package_name":             a.config.MobileAndroidPackage,
			"sha256_cert_fingerprints": a.config.MobileAndroidCertFingerprints,
		},
	}})
}
