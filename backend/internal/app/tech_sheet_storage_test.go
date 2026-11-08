package app

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestTechSheetsStorageRequiresSeparatePrivateBucket(t *testing.T) {
	for _, bucket := range []string{"", "public"} {
		client := &recordingMinioClient{}
		media := &s3MediaStorage{client: client, bucket: "public", privateBucket: bucket, publicBaseURL: "https://public.example"}
		if _, err := media.UploadStagePlot(context.Background(), "event", "act", "image/png", bytes.NewBufferString("png"), 3); err == nil || client.called {
			t.Fatal("stage plot used public/missing bucket")
		}
		if _, err := media.ReadStagePlot(context.Background(), "key"); err == nil {
			t.Fatal("private read used public/missing bucket")
		}
		if err := media.RemoveStagePlot(context.Background(), "key"); err == nil {
			t.Fatal("private deletion used public/missing bucket")
		}
	}
	client := &recordingMinioClient{}
	media := &s3MediaStorage{client: client, bucket: "public", privateBucket: "private", publicBaseURL: "https://public.example"}
	key, err := media.UploadStagePlot(context.Background(), "event", "act", "image/png", bytes.NewBufferString("png"), 3)
	if err != nil || client.bucket != "private" || !strings.HasPrefix(key, "events/event/tech/act/") || strings.Contains(key, "https:") || client.contentType != "image/png" {
		t.Fatalf("private upload key=%q bucket=%q err=%v", key, client.bucket, err)
	}
}

func TestTechSheetsValidationRejectsAmbiguousSharedChannelsAndOversizeFields(t *testing.T) {
	version := 0
	valid := func() putTechSheetRequest {
		return putTechSheetRequest{ExpectedVersion: &version, Inputs: []actInputDTO{{Label: "Kick", SourceType: "mic", SharedKey: "kick"}}}
	}
	for _, change := range []func(*putTechSheetRequest){
		func(r *putTechSheetRequest) { r.ExpectedVersion = nil },
		func(r *putTechSheetRequest) { r.MonitorNotes = strings.Repeat("x", 2001) },
		func(r *putTechSheetRequest) { r.Inputs[0].Label = "" },
		func(r *putTechSheetRequest) { r.Inputs[0].SourceType = "url" },
		func(r *putTechSheetRequest) { r.Inputs[0].SharedKey = "invalid key" },
		func(r *putTechSheetRequest) { r.Inputs[0].Notes = "nul\x00note" },
		func(r *putTechSheetRequest) { r.Inputs = append(r.Inputs, r.Inputs[0]) },
		func(r *putTechSheetRequest) { r.Inputs = make([]actInputDTO, 65) },
	} {
		req := valid()
		change(&req)
		if validateTechSheet(&req) {
			t.Fatal("invalid sheet accepted")
		}
	}
	req := valid()
	if !validateTechSheet(&req) || req.Inputs[0].ID == "" {
		t.Fatal("valid input not assigned identity")
	}
}
