package utils_test

import (
	"testing"

	"my-app/utils"
)

func TestParseHashtags(t *testing.T) {
	body := "Fixed #RustFS ListObjects #s3 403 issue"
	got := utils.ParseHashtags(body)
	if len(got) != 2 {
		t.Fatalf("expected 2 tags, got %v", got)
	}
	if got[0] != "rustfs" || got[1] != "s3" {
		t.Fatalf("unexpected tags: %v", got)
	}
}

func TestMergeTags(t *testing.T) {
	got := utils.MergeTags([]string{"oauth"}, []string{"#OAuth", "rust"}, []string{"rust"})
	if len(got) != 2 {
		t.Fatalf("expected 2 merged tags, got %v", got)
	}
}
