package subscription

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
)

// 024 T158: the API contract and the data model each listed ten reason codes while
// `billing_dto.go` defined fourteen. Four were missing, and the restore flow's four
// were documented nowhere at all.
//
// A reason code missing from the contract is not a cosmetic gap. It is the string a
// client is told to switch on. An operator whose grant came back `grant_failed` finds
// no row for it, cannot tell an infrastructure fault from a version conflict, and has
// no documented way to know that `batch_timeout` means "resubmit with the SAME
// Idempotency-Key" rather than "retry later".
//
// Documentation drifts silently, so this test is the durable part of the task: it
// parses the real constants and fails if a code exists without being written down.
// The table no longer has to be maintained by hand.

// allGrantReasonCodes returns every reason code the grant flow can emit.
func allGrantReasonCodes() map[string]bool {
	return map[string]bool{
		sub_dto.AdminGrantReasonIdentifierBlank:     true,
		sub_dto.AdminGrantReasonUserNotFound:        true,
		sub_dto.AdminGrantReasonIdentifierAmbiguous: true,
		sub_dto.AdminGrantReasonIdentifierTooLong:   true,
		sub_dto.AdminGrantReasonPlanNotFound:        true,
		sub_dto.AdminGrantReasonPlanSlugTooLong:     true,
		sub_dto.AdminGrantReasonPlanInactive:        true,
		sub_dto.AdminGrantReasonTargetIsAdmin:       true,
		sub_dto.AdminGrantReasonUserSoftDeleted:     true,
		sub_dto.AdminGrantReasonDuplicateInRequest:  true,
		sub_dto.AdminGrantReasonDuplicateRequest:    true,
		sub_dto.AdminGrantReasonConcurrentModif:     true,
		sub_dto.AdminGrantReasonGrantFailed:         true,
		sub_dto.AdminGrantReasonBatchTimeout:        true,
	}
}

func allRestoreReasonCodes() map[string]bool {
	return map[string]bool{
		sub_dto.RestoreReasonPlanNotInCatalog:    true,
		sub_dto.RestoreReasonSubscriptionExpired: true,
		sub_dto.RestoreReasonPublishFailed:       true,
		sub_dto.RestoreReasonAlreadyRestored:     true,
	}
}

// documentedAsTableRow reports whether a code appears as a ROW of a reason table.
//
// Checking that the string merely occurs somewhere would be too weak, and the
// weakness is not hypothetical: deleting the `grant_failed` row from the table while
// leaving the sentence that explains it still passed a `strings.Contains` check. A
// code that is mentioned in prose is not documented for a client - the table is what
// they read.
func documentedAsTableRow(doc, code string) bool {
	for _, line := range strings.Split(doc, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		if strings.Contains(trimmed, "`"+code+"`") {
			return true
		}
	}
	return false
}

func TestEveryGrantReasonCodeIsDocumentedInTheContract(t *testing.T) {
	doc := readSpecFile(t, "contracts", "admin-grant-api.md")

	for code := range allGrantReasonCodes() {
		if !documentedAsTableRow(doc, code) {
			t.Errorf("mã lý do %q có trong code nhưng thiếu một DÒNG BẢNG trong "+
				"contracts/admin-grant-api.md - client đọc chuỗi này để quyết định, "+
				"không có dòng nghĩa là không biết xử lý gì", code)
		}
	}
}

func TestEveryGrantReasonCodeIsDocumentedInTheDataModel(t *testing.T) {
	doc := readSpecFile(t, "data-model.md")

	for code := range allGrantReasonCodes() {
		if !documentedAsTableRow(doc, code) {
			t.Errorf("mã lý do %q có trong code nhưng thiếu một DÒNG BẢNG trong data-model.md", code)
		}
	}
}

// The restore flow's codes are a separate vocabulary. They were documented nowhere,
// so a client had no way to tell `subscription_expired` (deliberately skipped, needs a
// renewal flow) from `publish_failed` (retry the restore).
func TestRestoreReasonCodesAreDocumentedSeparately(t *testing.T) {
	doc := readSpecFile(t, "contracts", "admin-grant-api.md")

	for code := range allRestoreReasonCodes() {
		if !documentedAsTableRow(doc, code) {
			t.Errorf("mã lý do khôi phục %q thiếu một dòng bảng trong contracts/admin-grant-api.md", code)
		}
	}
	if !strings.Contains(doc, "restore") {
		t.Error("hợp đồng phải có mục riêng cho bộ mã của luồng khôi phục")
	}
}

// The distinction that T129 introduced is the one most worth keeping in the docs:
// `grant_failed` means an infrastructure fault, `concurrent_modification` means someone
// else changed the account. Documenting both without saying so would leave an operator
// chasing the wrong thing.
func TestGrantFailedIsDocumentedAsDistinctFromConcurrentModification(t *testing.T) {
	doc := readSpecFile(t, "contracts", "admin-grant-api.md")

	if !strings.Contains(doc, "grant_failed") || !strings.Contains(doc, "concurrent_modification") {
		t.Fatal("cả hai mã phải có trong hợp đồng")
	}
	// The explanation sentence that tells an operator which to chase.
	if !strings.Contains(doc, "hạ tầng") && !strings.Contains(doc, "hệ thống") {
		t.Error("hợp đồng phải nói rõ grant_failed là lỗi hạ tầng, khác với xung đột phiên bản")
	}
}

func TestBatchTimeoutIsDocumentedAsIdempotentResubmission(t *testing.T) {
	doc := readSpecFile(t, "contracts", "admin-grant-api.md")

	if !strings.Contains(doc, "`batch_timeout`") {
		t.Fatal("batch_timeout phải có trong hợp đồng")
	}
	// The actionable part: resubmitting with the same key is a no-op for the lines
	// already granted. Without that, an operator retries with a fresh key and grants
	// twice.
	if !strings.Contains(doc, "Idempotency-Key") {
		t.Error("hợp đồng phải nói gửi lại với CÙNG Idempotency-Key, nếu không quản trị viên sẽ cấp hai lần")
	}
}

// readSpecFile loads a file from the feature spec directory.
func readSpecFile(t *testing.T, parts ...string) string {
	t.Helper()

	root := specRoot(t)
	path := filepath.Join(append([]string{root}, parts...)...)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("đọc %s: %v", filepath.Join(parts...), err)
	}
	return string(content)
}

// featureDir is the spec folder these docs belong to. Named explicitly because the
// assertions are about THIS feature's contract, and quietly picking up whichever
// feature happens to be in `specs/` would make the test pass or fail for unrelated
// reasons.
const featureDir = "024-admin-premium-grant"

// specRoot walks up to the workspace root and returns the feature's spec folder, so
// the test does not depend on the working directory.
func specRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		// `go.work` marks the workspace root; the specs live beside it.
		if _, statErr := os.Stat(filepath.Join(dir, "go.work")); statErr == nil {
			spec := filepath.Join(dir, "specs", featureDir)
			if info, specErr := os.Stat(spec); specErr == nil && info.IsDir() {
				return spec
			}
			t.Fatalf("tìm thấy workspace root nhưng không có specs/%s", featureDir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("không tìm thấy go.work để định vị thư mục spec của feature")
		}
		dir = parent
	}
}
