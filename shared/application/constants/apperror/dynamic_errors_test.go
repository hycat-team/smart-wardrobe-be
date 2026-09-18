package apperror

import "testing"

func TestNewRequestEntityTooLargeMapsStatusAndTitle(t *testing.T) {
	err := NewRequestEntityTooLarge("Dữ liệu gửi lên vượt quá giới hạn 2MB.")

	if err.Status != 413 {
		t.Fatalf("expected status 413, got %d", err.Status)
	}
	if err.Title != "Dữ liệu gửi lên quá lớn" {
		t.Fatalf("unexpected title: %q", err.Title)
	}
	if err.Message != "Dữ liệu gửi lên vượt quá giới hạn 2MB." {
		t.Fatalf("unexpected message: %q", err.Message)
	}
}
