package drive

import "testing"

func TestEscapeDriveString_Apostrophe(t *testing.T) {
	if got, want := escapeDriveString("o'brien"), `o\'brien`; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEscapeDriveString_Backslash(t *testing.T) {
	if got, want := escapeDriveString(`a\b`), `a\\b`; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEscapeDriveString_BackslashThenApostrophe(t *testing.T) {
	// Backslash must be escaped before apostrophe so the doubled-backslash
	// isn't itself re-escaped by the apostrophe pass.
	if got, want := escapeDriveString(`\'`), `\\\'`; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEscapeDriveString_NoSpecial(t *testing.T) {
	if got := escapeDriveString("plain-id_123"); got != "plain-id_123" {
		t.Fatalf("plain string should be unchanged, got %q", got)
	}
}

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
