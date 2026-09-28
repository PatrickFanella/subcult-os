package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

const testAndroidFingerprint = "14:6D:E9:83:C5:73:06:50:D8:EE:B9:95:2F:34:FC:64:16:A0:83:42:E6:1D:BE:A8:8A:04:96:B2:3F:CF:44:E5"

func appLinkTestApp(config Config) *App {
	config.AppEnv = "test"
	config.PublicWebURL = "https://example.test"
	config.SessionSecret = "test-secret"
	return New(config, nil)
}

func serveAppLink(t *testing.T, app *App, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestAppLinkDocumentsAreAbsentUntilConfigured(t *testing.T) {
	app := appLinkTestApp(Config{})
	for _, path := range []string{"/.well-known/apple-app-site-association", "/.well-known/assetlinks.json"} {
		if rec := serveAppLink(t, app, path); rec.Code != http.StatusNotFound {
			t.Fatalf("%s: expected 404 when unconfigured, got %d: %s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestAppleAppSiteAssociationClaimsOnlyIdentityLinks(t *testing.T) {
	app := appLinkTestApp(Config{MobileAppleAppIDs: []string{"ABCDE12345.tv.clpr.subcultos"}})
	rec := serveAppLink(t, app, "/.well-known/apple-app-site-association")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json without redirect, got %q", got)
	}
	var document struct {
		Applinks struct {
			Details []struct {
				AppIDs     []string `json:"appIDs"`
				Components []struct {
					Path  string            `json:"/"`
					Query map[string]string `json:"?"`
				} `json:"components"`
			} `json:"details"`
		} `json:"applinks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode association: %v", err)
	}
	if len(document.Applinks.Details) != 1 || !reflect.DeepEqual(document.Applinks.Details[0].AppIDs, []string{"ABCDE12345.tv.clpr.subcultos"}) {
		t.Fatalf("unexpected app IDs: %+v", document.Applinks.Details)
	}
	var paths []string
	for _, component := range document.Applinks.Details[0].Components {
		if component.Query["token"] != "?*" {
			t.Fatalf("component %s must require a token, got %v", component.Path, component.Query)
		}
		paths = append(paths, component.Path)
	}
	if !reflect.DeepEqual(paths, []string{"/verify-email", "/recover-password"}) {
		t.Fatalf("association must claim only identity links, got %v", paths)
	}
}

func TestAndroidAssetLinksNameThePackageAndSigningKey(t *testing.T) {
	app := appLinkTestApp(Config{MobileAndroidPackage: "tv.clpr.subcultos", MobileAndroidCertFingerprints: []string{testAndroidFingerprint}})
	rec := serveAppLink(t, app, "/.well-known/assetlinks.json")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var statements []struct {
		Relation []string `json:"relation"`
		Target   struct {
			Namespace    string   `json:"namespace"`
			PackageName  string   `json:"package_name"`
			Fingerprints []string `json:"sha256_cert_fingerprints"`
		} `json:"target"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &statements); err != nil {
		t.Fatalf("decode asset links: %v", err)
	}
	if len(statements) != 1 {
		t.Fatalf("expected one statement, got %d", len(statements))
	}
	statement := statements[0]
	if !reflect.DeepEqual(statement.Relation, []string{"delegate_permission/common.handle_all_urls"}) ||
		statement.Target.Namespace != "android_app" || statement.Target.PackageName != "tv.clpr.subcultos" ||
		!reflect.DeepEqual(statement.Target.Fingerprints, []string{testAndroidFingerprint}) {
		t.Fatalf("unexpected asset links statement: %+v", statement)
	}
}

func TestAppLinkConfigParsingAndValidation(t *testing.T) {
	t.Setenv("MOBILE_APPLE_APP_IDS", " ABCDE12345.tv.clpr.subcultos , ")
	t.Setenv("MOBILE_ANDROID_PACKAGE", "tv.clpr.subcultos")
	t.Setenv("MOBILE_ANDROID_CERT_SHA256", strings.ToLower(testAndroidFingerprint))
	config := LoadConfig()
	if !reflect.DeepEqual(config.MobileAppleAppIDs, []string{"ABCDE12345.tv.clpr.subcultos"}) {
		t.Fatalf("unexpected Apple app IDs: %v", config.MobileAppleAppIDs)
	}
	if !reflect.DeepEqual(config.MobileAndroidCertFingerprints, []string{testAndroidFingerprint}) {
		t.Fatalf("fingerprints must be normalized to uppercase, got %v", config.MobileAndroidCertFingerprints)
	}
	if problems := config.appLinkProblems(); len(problems) != 0 {
		t.Fatalf("expected valid app link config, got %v", problems)
	}

	cases := map[string]Config{
		"MOBILE_APPLE_APP_IDS":        {MobileAppleAppIDs: []string{"tv.clpr.subcultos"}},
		"MOBILE_ANDROID_PACKAGE must": {MobileAndroidPackage: "subcultos", MobileAndroidCertFingerprints: []string{testAndroidFingerprint}},
		"MOBILE_ANDROID_CERT_SHA256":  {MobileAndroidPackage: "tv.clpr.subcultos", MobileAndroidCertFingerprints: []string{"14:6D"}},
		"set together":                {MobileAndroidPackage: "tv.clpr.subcultos"},
	}
	for want, invalid := range cases {
		if problems := strings.Join(invalid.appLinkProblems(), "; "); !strings.Contains(problems, want) {
			t.Fatalf("expected problem containing %q, got %q", want, problems)
		}
	}
}
