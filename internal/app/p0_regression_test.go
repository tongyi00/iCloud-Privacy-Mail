package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFileStoreMutationRollsBackWhenSaveFails(t *testing.T) {
	store := newTestStore(t)
	user, err := store.CreateUser("rollback-user", "password")
	if err != nil {
		t.Fatal(err)
	}
	before := store.Snapshot()
	originalPath := store.path
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	store.path = filepath.Join(blocker, "state.json")

	if _, err := store.AddAccountForOwner(user.ID, "failed", "failed@example.com", ""); err == nil {
		t.Fatal("AddAccountForOwner() error = nil, want persistence failure")
	}
	if got := store.Snapshot(); !reflect.DeepEqual(got, before) {
		t.Fatalf("Snapshot changed after failed save\n got: %#v\nwant: %#v", got, before)
	}
	store.path = originalPath
}

func TestFailedMutationIsNotPersistedByLaterSuccessfulMutation(t *testing.T) {
	store := newTestStore(t)
	originalPath := store.path
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	store.path = filepath.Join(blocker, "state.json")
	if _, err := store.AddAccount("failed", "failed@example.com", ""); err == nil {
		t.Fatal("failed mutation unexpectedly persisted")
	}
	store.path = originalPath
	if _, err := store.AddAccount("saved", "saved@example.com", ""); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewFileStore(originalPath)
	if err != nil {
		t.Fatal(err)
	}
	accounts := reloaded.Snapshot().Accounts
	if len(accounts) != 1 || accounts[0].Label != "saved" {
		t.Fatalf("persisted accounts = %+v, want only saved mutation", accounts)
	}
}

func TestLogoutKeepsCookieWhenSessionDeleteCannotPersist(t *testing.T) {
	store := newTestStore(t)
	handler := NewServer(Config{}, store, discardLogger())
	cookie, _ := registerTestUser(t, handler, "logout-admin", "password")
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	store.path = filepath.Join(blocker, "state.json")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(cookie)
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("logout = %d body=%s, want 500", rr.Code, rr.Body.String())
	}
	if values := rr.Header().Values("Set-Cookie"); len(values) != 0 {
		t.Fatalf("logout cleared cookie after failed persistence: %v", values)
	}
	if _, _, ok := store.WebSessionByToken(cookie.Value); !ok {
		t.Fatal("session disappeared from memory after failed persistence")
	}
}

func TestMailboxSyncBatchPersistsMessagesAndCursorsOnce(t *testing.T) {
	oldInterval := mailboxMailSyncMinInterval
	mailboxMailSyncMinInterval = 0
	t.Cleanup(func() { mailboxMailSyncMinInterval = oldInterval })
	store := newTestStore(t)
	ownerID := "batch-owner"
	accountID := "batch-account"
	if err := store.SaveICloudSessionForOwner(ownerID, testIMAPSession(ownerID, accountID, "batch-owner@icloud.com")); err != nil {
		t.Fatal(err)
	}
	mailbox, err := store.AddMailboxForOwner(ownerID, accountID, "batch", "batch@icloud.com")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewServer(Config{}, store, discardLogger())
	server := handler.(*Server)
	server.syncCodeMailboxBatchWithCursor = func(context.Context, LoginState, []Mailbox, time.Time, string, int) (iCloudIMAPSyncResult, error) {
		return iCloudIMAPSyncResult{
			LastUID: "42",
			MessagesByMailbox: map[string][]ICloudSyncedMessage{mailbox.ID: {{
				RemoteID: "imap:42", UID: "42", Subject: "ChatGPT code", Body: "Use 123456 to continue.", ReceivedAt: time.Now(),
			}}},
		}, nil
	}
	before := store.saveCount
	if _, err := server.syncMailbox(context.Background(), mailbox, time.Time{}, "ChatGPT"); err != nil {
		t.Fatal(err)
	}
	if got := store.saveCount - before; got != 1 {
		t.Fatalf("sync state saves = %d, want 1", got)
	}
}

func TestStoreRejectsMailboxForMissingOwnerOrAccount(t *testing.T) {
	store := newTestStore(t)
	mailbox, err := store.AddMailboxForOwner("future-owner", "missing-account", "invalid", "invalid@icloud.com")
	if err != nil {
		t.Fatal(err)
	}
	admin, _, _, err := store.BootstrapAdmin("validation-admin", "password", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	store.state.Mailboxes[store.mailboxIndexLocked(mailbox.ID)].OwnerID = admin.ID
	store.mu.Unlock()
	account, err := store.AddAccountForOwner(admin.ID, "valid", "valid@example.com", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMailboxForOwner("missing-owner", account.ID, "missing-owner", "missing-owner@icloud.com"); err == nil {
		t.Fatal("AddMailboxForOwner() accepted a missing owner")
	}
	if _, err := store.AddMailboxForOwner(admin.ID, "missing-account", "missing-account", "missing-account@icloud.com"); err == nil {
		t.Fatal("AddMailboxForOwner() accepted a missing account")
	}
	if _, err := store.CommitMailboxSync("missing-owner", nil, nil); err == nil {
		t.Fatal("CommitMailboxSync() accepted a missing owner")
	}
	validMailbox, err := store.AddMailboxForOwner(admin.ID, account.ID, "valid", "valid@icloud.com")
	if err != nil {
		t.Fatal(err)
	}
	before := store.Snapshot()
	updates := []MailboxSyncUpdate{
		{MailboxID: validMailbox.ID, Messages: []ICloudSyncedMessage{{RemoteID: "imap:1", Subject: "ChatGPT", Body: "123456"}}},
		{MailboxID: mailbox.ID},
	}
	if _, err := store.CommitMailboxSync(admin.ID, updates, nil); err == nil {
		t.Fatal("CommitMailboxSync() accepted a missing account")
	}
	if after := store.Snapshot(); !reflect.DeepEqual(after, before) {
		t.Fatal("failed batch left partial changes in memory")
	}
}

func TestPublicRegistrationIsClosed(t *testing.T) {
	server := NewServer(Config{}, newTestStore(t), discardLogger())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"username":"public-user","password":"password"}`))
	req.Header.Set("Content-Type", "application/json")
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden || !strings.Contains(rr.Body.String(), `"code":"registration_closed"`) {
		t.Fatalf("register = %d body=%s, want registration_closed", rr.Code, rr.Body.String())
	}
}

func TestBootstrapRequiresEmptyStoreAndToken(t *testing.T) {
	t.Setenv("IPM_BOOTSTRAP_TOKEN", "bootstrap-secret")
	server := NewServer(Config{}, newTestStore(t), discardLogger())
	body := `{"username":"admin","password":"password"}`

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/bootstrap", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	server.ServeHTTP(rr, req)
	if rr.Code == http.StatusCreated {
		t.Fatalf("bootstrap without bearer token unexpectedly succeeded: %s", rr.Body.String())
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/auth/bootstrap", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer bootstrap-secret")
	server.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("bootstrap with token = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestBootstrapCannotRunTwice(t *testing.T) {
	t.Setenv("IPM_BOOTSTRAP_TOKEN", "bootstrap-secret")
	server := NewServer(Config{}, newTestStore(t), discardLogger())
	bootstrap := func(username string) *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/bootstrap", strings.NewReader(`{"username":"`+username+`","password":"password"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer bootstrap-secret")
		server.ServeHTTP(rr, req)
		return rr
	}
	if first := bootstrap("first-admin"); first.Code != http.StatusCreated {
		t.Fatalf("first bootstrap = %d body=%s", first.Code, first.Body.String())
	}
	if second := bootstrap("second-admin"); second.Code != http.StatusConflict || !strings.Contains(second.Body.String(), `"code":"bootstrap_completed"`) {
		t.Fatalf("second bootstrap = %d body=%s", second.Code, second.Body.String())
	}
}

func TestAdminCreatesNormalUser(t *testing.T) {
	store := newTestStore(t)
	handler := NewServer(Config{}, store, discardLogger())
	adminCookie, _ := registerTestUser(t, handler, "create-admin", "password")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", strings.NewReader(`{"username":"normal-user","password":"password"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create user = %d body=%s", rr.Code, rr.Body.String())
	}
	users := store.Users()
	if len(users) != 2 || users[1].IsAdmin {
		t.Fatalf("created users = %+v, want second normal user", users)
	}
}

func TestLoginRateLimitByUsernameAndClientIP(t *testing.T) {
	store := newTestStore(t)
	handler := NewServer(Config{}, store, discardLogger())
	registerTestUser(t, handler, "limited-user", "password")
	for attempt := 1; attempt <= authFailureLimit; attempt++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"limited-user","password":"wrong-password"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.10:1234"
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d = %d body=%s", attempt, rr.Code, rr.Body.String())
		}
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"limited-user","password":"password"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.10:1234"
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests || !strings.Contains(rr.Body.String(), `"code":"auth_rate_limited"`) {
		t.Fatalf("blocked login = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAuthRateLimiterPrunesExpiredEntries(t *testing.T) {
	limiter := &authRateLimiter{entries: make(map[string]authRateEntry)}
	startedAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	limiter.recordLoginFailure("expired-user", "203.0.113.10", startedAt)
	if !limiter.allowAction("bootstrap:203.0.113.11", startedAt) {
		t.Fatal("first action was rejected")
	}

	limiter.loginBlocked("trigger-cleanup", "203.0.113.12", startedAt.Add(authBlockDuration+time.Second))
	if len(limiter.entries) != 0 {
		t.Fatalf("entries after cleanup = %d, want 0", len(limiter.entries))
	}
}

func TestAuthRateLimiterKeepsActiveBlockDuringCleanup(t *testing.T) {
	limiter := &authRateLimiter{entries: make(map[string]authRateEntry)}
	startedAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	for range authFailureLimit {
		limiter.recordLoginFailure("blocked-user", "203.0.113.20", startedAt)
	}

	now := startedAt.Add(time.Minute)
	limiter.allowAction("bootstrap:203.0.113.21", now)
	if !limiter.loginBlocked("blocked-user", "203.0.113.20", now) {
		t.Fatal("active login block was removed during cleanup")
	}
	for _, key := range []string{"login-user:blocked-user", "login-ip:203.0.113.20"} {
		if _, ok := limiter.entries[key]; !ok {
			t.Fatalf("active entry %q was removed", key)
		}
	}
}

func TestAuthRateLimiterCleanupPreservesThresholds(t *testing.T) {
	limiter := &authRateLimiter{entries: make(map[string]authRateEntry)}
	startedAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	for attempt := 1; attempt < authFailureLimit; attempt++ {
		limiter.recordLoginFailure("threshold-user", "203.0.113.30", startedAt)
		if limiter.loginBlocked("threshold-user", "203.0.113.30", startedAt) {
			t.Fatalf("login blocked after failure %d", attempt)
		}
	}
	limiter.recordLoginFailure("threshold-user", "203.0.113.30", startedAt)
	if !limiter.loginBlocked("threshold-user", "203.0.113.30", startedAt) {
		t.Fatal("login was not blocked after threshold failure")
	}

	for attempt := 1; attempt <= authActionLimit; attempt++ {
		if !limiter.allowAction("create-user:203.0.113.31", startedAt) {
			t.Fatalf("action %d was rejected", attempt)
		}
	}
	if limiter.allowAction("create-user:203.0.113.31", startedAt) {
		t.Fatal("action after threshold was allowed")
	}
	if !limiter.allowAction("create-user:203.0.113.31", startedAt.Add(authActionWindow)) {
		t.Fatal("action was rejected after window expiry")
	}
}

func TestAuthRateLimiterCleanupHandlesClockRollback(t *testing.T) {
	limiter := &authRateLimiter{entries: make(map[string]authRateEntry)}
	rollbackTime := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	limiter.allowAction("bootstrap:future", rollbackTime.Add(time.Hour))
	limiter.entries["legacy-expired"] = authRateEntry{}

	limiter.loginBlocked("trigger-cleanup", "203.0.113.40", rollbackTime)
	if _, ok := limiter.entries["legacy-expired"]; ok {
		t.Fatal("expired entry survived cleanup after clock rollback")
	}
}

func TestPasswordVerificationDoesNotHoldStoreLock(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.CreateUser("lock-user", "password"); err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	store.passwordVerifier = func(_, _ string) bool {
		close(entered)
		<-release
		return false
	}
	done := make(chan struct{})
	go func() {
		_, _, _, _ = store.AuthenticateUserAndCreateSession("lock-user", "wrong-password", time.Hour)
		close(done)
	}()
	<-entered
	snapshotDone := make(chan struct{})
	go func() {
		_ = store.Snapshot()
		close(snapshotDone)
	}()
	select {
	case <-snapshotDone:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("Snapshot blocked while password verification was running")
	}
	close(release)
	<-done
}

func TestUnknownAndKnownUserLoginHaveEquivalentHashWork(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.CreateUser("known-user", "password"); err != nil {
		t.Fatal(err)
	}
	calls := 0
	store.passwordVerifier = func(_, encoded string) bool {
		calls++
		if encoded == "" {
			t.Fatal("password verifier received an empty hash")
		}
		return false
	}
	_, _, _, _ = store.AuthenticateUserAndCreateSession("known-user", "wrong-password", time.Hour)
	knownCalls := calls
	_, _, _, _ = store.AuthenticateUserAndCreateSession("missing-user", "wrong-password", time.Hour)
	unknownCalls := calls - knownCalls
	if knownCalls != 1 || unknownCalls != 1 {
		t.Fatalf("hash calls known/unknown = %d/%d, want 1/1", knownCalls, unknownCalls)
	}
}

func TestApple2FAPendingRejectsDifferentOwner(t *testing.T) {
	store := newTestStore(t)
	handler := NewServer(Config{}, store, discardLogger())
	server := handler.(*Server)
	ownerCookie, _ := registerTestUser(t, handler, "pending-owner", "password")
	otherCookie, _ := registerTestUser(t, handler, "pending-other", "password")
	server.startAppleAccountManageLogin = func(_ context.Context, _ string, _ string, pendingStore *appleAuthPendingStore, _ string) (appleAuthStartResult, error) {
		pending, err := pendingStore.put(&appleAuthSession{AppleID: "owner@example.com"})
		return appleAuthStartResult{PendingID: pending.ID, Needs2FA: true}, err
	}
	server.submitAppleAccountManage2FA = func(_ context.Context, _ appleAuthPending, _ string, _ json.RawMessage) (ICloudSession, error) {
		return ICloudSession{AppleID: "owner@example.com"}, nil
	}

	start := httptest.NewRecorder()
	startReq := httptest.NewRequest(http.MethodPost, "/api/apple-account/login/start", strings.NewReader(`{"apple_id":"owner@example.com","password":"password"}`))
	startReq.Header.Set("Content-Type", "application/json")
	startReq.AddCookie(ownerCookie)
	handler.ServeHTTP(start, startReq)
	var started struct {
		PendingID string `json:"pending_id"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &started); err != nil || started.PendingID == "" {
		t.Fatalf("start = %d body=%s err=%v", start.Code, start.Body.String(), err)
	}

	submit := httptest.NewRecorder()
	submitReq := httptest.NewRequest(http.MethodPost, "/api/apple-account/login/2fa", strings.NewReader(`{"pending_id":"`+started.PendingID+`","code":"123456"}`))
	submitReq.Header.Set("Content-Type", "application/json")
	submitReq.AddCookie(otherCookie)
	handler.ServeHTTP(submit, submitReq)
	if submit.Code != http.StatusBadRequest || !strings.Contains(submit.Body.String(), `"code":"apple_login_pending_expired"`) {
		t.Fatalf("cross-owner submit = %d body=%s", submit.Code, submit.Body.String())
	}
}

func TestICloudProtocol2FAPendingRejectsDifferentOwner(t *testing.T) {
	store := newAppleAuthPendingStore()
	pending, err := store.put(&appleAuthSession{})
	if err != nil {
		t.Fatal(err)
	}
	store.setOwner(pending.ID, "owner-a", "icloud_protocol")
	if _, state := store.begin(pending.ID, "owner-b"); state != "expired" {
		t.Fatalf("cross-owner pending state = %q, want expired", state)
	}
}

func TestApple2FARetryAfterPersistFailureDoesNotCallAppleAgain(t *testing.T) {
	store := newTestStore(t)
	handler := NewServer(Config{}, store, discardLogger())
	server := handler.(*Server)
	cookie, _ := registerTestUser(t, handler, "retry-owner", "password")
	server.startAppleAccountManageLogin = func(_ context.Context, _ string, _ string, pendingStore *appleAuthPendingStore, _ string) (appleAuthStartResult, error) {
		pending, err := pendingStore.put(&appleAuthSession{AppleID: "retry@example.com"})
		return appleAuthStartResult{PendingID: pending.ID, Needs2FA: true}, err
	}
	calls := 0
	server.submitAppleAccountManage2FA = func(_ context.Context, _ appleAuthPending, _ string, _ json.RawMessage) (ICloudSession, error) {
		calls++
		return ICloudSession{AppleID: "retry@example.com"}, nil
	}
	start := httptest.NewRecorder()
	startReq := httptest.NewRequest(http.MethodPost, "/api/apple-account/login/start", strings.NewReader(`{"apple_id":"retry@example.com","password":"password"}`))
	startReq.Header.Set("Content-Type", "application/json")
	startReq.AddCookie(cookie)
	handler.ServeHTTP(start, startReq)
	var started struct {
		PendingID string `json:"pending_id"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &started); err != nil {
		t.Fatal(err)
	}
	originalPath := store.path
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	store.path = filepath.Join(blocker, "state.json")
	submit := func() *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/apple-account/login/2fa", strings.NewReader(`{"pending_id":"`+started.PendingID+`","code":"123456"}`))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		handler.ServeHTTP(rr, req)
		return rr
	}
	if first := submit(); first.Code != http.StatusInternalServerError {
		t.Fatalf("first submit = %d body=%s", first.Code, first.Body.String())
	}
	store.path = originalPath
	if second := submit(); second.Code != http.StatusOK {
		t.Fatalf("retry submit = %d body=%s", second.Code, second.Body.String())
	}
	if calls != 1 {
		t.Fatalf("Apple submit calls = %d, want 1", calls)
	}
}

func TestApple2FAPendingAllowsSingleSubmitter(t *testing.T) {
	store := newTestStore(t)
	handler := NewServer(Config{}, store, discardLogger())
	server := handler.(*Server)
	cookie, user := registerTestUser(t, handler, "single-owner", "password")
	pending, err := server.appleAccountLogins.put(&appleAuthSession{})
	if err != nil {
		t.Fatal(err)
	}
	server.appleAccountLogins.setOwner(pending.ID, user.ID, "apple_account")
	entered := make(chan struct{})
	release := make(chan struct{})
	calls := 0
	server.submitAppleAccountManage2FA = func(_ context.Context, _ appleAuthPending, _ string, _ json.RawMessage) (ICloudSession, error) {
		calls++
		close(entered)
		<-release
		return ICloudSession{AppleID: "single@example.com"}, nil
	}
	submit := func() *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/apple-account/login/2fa", strings.NewReader(`{"pending_id":"`+pending.ID+`","code":"123456"}`))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		handler.ServeHTTP(rr, req)
		return rr
	}
	firstDone := make(chan *httptest.ResponseRecorder, 1)
	go func() { firstDone <- submit() }()
	<-entered
	second := submit()
	if second.Code != http.StatusConflict || !strings.Contains(second.Body.String(), `"code":"apple_login_in_progress"`) {
		close(release)
		t.Fatalf("concurrent submit = %d body=%s", second.Code, second.Body.String())
	}
	close(release)
	if first := <-firstDone; first.Code != http.StatusOK {
		t.Fatalf("first submit = %d body=%s", first.Code, first.Body.String())
	}
	if calls != 1 {
		t.Fatalf("Apple submit calls = %d, want 1", calls)
	}
}

func TestRecipientMatchingIgnoresAliasInSubject(t *testing.T) {
	raw := "From: target@icloud.com\r\n" +
		"To: someone-else@icloud.com\r\n" +
		"Subject: target@icloud.com ChatGPT verification code\r\n" +
		"Content-Type: text/plain\r\n\r\nCode: 123456\r\n"
	got := iCloudIMAPMessagesByMailbox(
		[]iCloudIMAPFetchedMessage{{UID: "1", Raw: []byte(raw)}},
		[]Mailbox{{ID: "target", Email: "target@icloud.com"}},
		time.Time{},
		"ChatGPT",
	)
	if len(got["target"]) != 0 {
		t.Fatalf("subject/from alias matched recipient: %+v", got["target"])
	}
}

func TestRecipientMatchingRequiresExactAddress(t *testing.T) {
	got := matchingMailboxIDs("aa@icloud.com", map[string]string{"short": "a@icloud.com"})
	if len(got) != 0 {
		t.Fatalf("prefix alias matched recipient: %v", got)
	}
}

func TestIMAPBacklogFetchesOldestBatchFirst(t *testing.T) {
	uids := []int{101, 102, 103, 104, 105, 106, 107, 108, 109, 110}
	got := firstIntValues(uids, 8)
	want := []int{101, 102, 103, 104, 105, 106, 107, 108}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selected UIDs = %v, want oldest batch %v", got, want)
	}
}

func TestIMAPFiltersUIDAtOrBeforeCursor(t *testing.T) {
	got := imapUIDsAfter([]int{99, 100, 101, 102}, 100)
	if want := []int{101, 102}; !reflect.DeepEqual(got, want) {
		t.Fatalf("filtered UIDs = %v, want %v", got, want)
	}
}

func TestIMAPCursorStopsAtLastProcessedUID(t *testing.T) {
	processed := map[int]struct{}{101: {}, 102: {}, 104: {}}
	if got := imapLastProcessedUID([]int{101, 102, 103, 104}, processed); got != "102" {
		t.Fatalf("last processed UID = %q, want 102", got)
	}
}

func TestIMAPBacklogEventuallyProcessesEveryUID(t *testing.T) {
	pending := []int{101, 102, 103, 104, 105, 106, 107, 108, 109, 110}
	first := firstIntValues(imapUIDsAfter(pending, 100), 8)
	second := firstIntValues(imapUIDsAfter(pending, first[len(first)-1]), 8)
	got := append(append([]int(nil), first...), second...)
	if !reflect.DeepEqual(got, pending) {
		t.Fatalf("processed UIDs = %v, want %v", got, pending)
	}
}

func TestIMAPFetchFailureDoesNotAdvanceCursor(t *testing.T) {
	oldInterval := mailboxMailSyncMinInterval
	mailboxMailSyncMinInterval = 0
	t.Cleanup(func() { mailboxMailSyncMinInterval = oldInterval })
	store := newTestStore(t)
	ownerID := "fetch-failure-owner"
	accountID := "fetch-failure-account"
	if err := store.SaveICloudSessionForOwner(ownerID, testIMAPSession(ownerID, accountID, "fetch-failure@icloud.com")); err != nil {
		t.Fatal(err)
	}
	mailbox, err := store.AddMailboxForOwner(ownerID, accountID, "fetch-failure", "fetch-failure-alias@icloud.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetMailboxSyncCursor(mailbox.ID, time.Now().Add(-time.Minute), "100"); err != nil {
		t.Fatal(err)
	}
	before := store.Snapshot()
	handler := NewServer(Config{}, store, discardLogger())
	server := handler.(*Server)
	server.syncCodeMailboxBatchWithCursor = func(context.Context, LoginState, []Mailbox, time.Time, string, int) (iCloudIMAPSyncResult, error) {
		return iCloudIMAPSyncResult{}, errors.New("fetch failed")
	}
	if _, err := server.syncMailbox(context.Background(), mailbox, time.Time{}, "ChatGPT"); err == nil {
		t.Fatal("syncMailbox() error = nil, want fetch failure")
	}
	if after := store.Snapshot(); !reflect.DeepEqual(after, before) {
		t.Fatalf("state changed after fetch failure\n got: %#v\nwant: %#v", after, before)
	}
}

func TestDecodeJSONRejectsTrailingValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"value":1}{"value":2}`))
	var target struct {
		Value int `json:"value"`
	}
	if err := decodeJSON(req, &target); err == nil {
		t.Fatal("decodeJSON() error = nil, want trailing value rejection")
	}
}

func TestDecodeJSONRejectsBodyOverLimit(t *testing.T) {
	body := `{"value":"` + strings.Repeat("x", (1<<20)+1) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	var target struct {
		Value string `json:"value"`
	}
	err := decodeJSON(req, &target)
	var coded codedError
	if !errors.As(err, &coded) || coded.code != "request_too_large" {
		t.Fatalf("decodeJSON() error = %v, want request_too_large", err)
	}
}

func TestDecodeJSONRejectsTrailingGarbage(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"value":1}garbage`))
	var target struct {
		Value int `json:"value"`
	}
	if err := decodeJSON(req, &target); err == nil {
		t.Fatal("decodeJSON() error = nil, want trailing garbage rejection")
	}
}

func TestDecodeJSONAllowsTrailingWhitespace(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{\"value\":1}\r\n\t "))
	var target struct {
		Value int `json:"value"`
	}
	if err := decodeJSON(req, &target); err != nil || target.Value != 1 {
		t.Fatalf("decodeJSON() target=%+v err=%v", target, err)
	}
}

func TestConcurrentMailboxCodeQueriesReturnSameFreshCode(t *testing.T) {
	store := newTestStore(t)
	mailbox, err := store.AddMailbox("", "parallel", "parallel@icloud.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMessage(mailbox.ID, "ChatGPT code", "noreply@example.com", "Use 654321 to continue.", time.Now()); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewServer(Config{}, store, discardLogger())
	const requests = 8
	results := make(chan string, requests)
	var wg sync.WaitGroup
	for range requests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/mailboxes/"+mailbox.Email+"/code?key="+mailbox.APIToken+"&cache=1", nil)
			handler.ServeHTTP(rr, req)
			var body struct {
				Code string `json:"code"`
			}
			_ = json.Unmarshal(rr.Body.Bytes(), &body)
			results <- body.Code
		}()
	}
	wg.Wait()
	close(results)
	for code := range results {
		if code != "654321" {
			t.Fatalf("concurrent code = %q, want 654321", code)
		}
	}
	after, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("code query changed persisted state")
	}
}
