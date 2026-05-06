package drive

import "testing"

func TestExportMimeFor_Doc(t *testing.T) {
	if got := exportMimeFor(MimeTypeGoogleDoc); got != "text/plain" {
		t.Fatalf("doc → got %q, want text/plain", got)
	}
}

func TestExportMimeFor_Sheet(t *testing.T) {
	if got := exportMimeFor(MimeTypeGoogleSheet); got != "text/csv" {
		t.Fatalf("sheet → got %q, want text/csv", got)
	}
}

func TestExportMimeFor_Slides(t *testing.T) {
	if got := exportMimeFor(MimeTypeGoogleSlides); got != "text/plain" {
		t.Fatalf("slides → got %q, want text/plain", got)
	}
}

func TestExportMimeFor_NonNative(t *testing.T) {
	if got := exportMimeFor("application/pdf"); got != "" {
		t.Fatalf("non-native should return empty, got %q", got)
	}
}

func TestIsTextLike_TextPrefix(t *testing.T) {
	if !isTextLike("text/markdown") {
		t.Fatal("text/* should be text-like")
	}
}

func TestIsTextLike_JSON(t *testing.T) {
	if !isTextLike("application/json") {
		t.Fatal("application/json should be text-like")
	}
}

func TestIsTextLike_Binary(t *testing.T) {
	if isTextLike("image/png") {
		t.Fatal("image/png should not be text-like")
	}
}
