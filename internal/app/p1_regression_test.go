package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestIMAPSelectParsesUIDValidity(t *testing.T) {
	lines := []string{"* OK [UIDVALIDITY 3857529045] UIDs valid", "A002 OK SELECT completed"}
	if got := imapSelectUIDValidity(lines); got != "3857529045" {
		t.Fatalf("UIDVALIDITY = %q, want 3857529045", got)
	}
}

func TestIMAPUIDValidityChangeResetsCursorAndUsesLookback(t *testing.T) {
	state := LoginState{IMAPLastSyncUID: "100", IMAPUIDValidity: "old"}
	mailboxes := []Mailbox{{ID: "one", LastSyncUID: "100"}, {ID: "two", LastSyncUID: "120"}}
	nextState, nextMailboxes, reset := prepareIMAPGeneration(state, mailboxes, "new")
	if !reset || nextState.IMAPLastSyncUID != "" || nextState.IMAPUIDValidity != "new" {
		t.Fatalf("prepared state = %+v reset=%v", nextState, reset)
	}
	for _, mailbox := range nextMailboxes {
		if mailbox.LastSyncUID != "" {
			t.Fatalf("mailbox cursor was not reset: %+v", mailbox)
		}
	}
	after := time.Now().Add(-24 * time.Hour)
	if command := imapSearchCommand(nextState, nextMailboxes, after); !strings.HasPrefix(command, "UID SEARCH SINCE ") {
		t.Fatalf("generation reset search = %q, want lookback", command)
	}
}

func TestIMAPEmptyRangeDoesNotMoveCursorBackward(t *testing.T) {
	oldInterval := mailboxMailSyncMinInterval
	mailboxMailSyncMinInterval = 0
	t.Cleanup(func() { mailboxMailSyncMinInterval = oldInterval })
	store := newTestStore(t)
	ownerID, accountID := "empty-generation-owner", "empty-generation-account"
	session := testIMAPSession(ownerID, accountID, "empty-generation@icloud.com")
	state, _ := iCloudIMAPLoginState(session)
	state.IMAPLastSyncUID = "100"
	state.IMAPUIDValidity = "generation"
	session = withICloudIMAPLoginState(session, state)
	if err := store.SaveICloudSessionForOwner(ownerID, session); err != nil {
		t.Fatal(err)
	}
	mailbox, err := store.AddMailboxForOwner(ownerID, accountID, "empty", "empty-generation-alias@icloud.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetMailboxSyncCursor(mailbox.ID, time.Now(), "100"); err != nil {
		t.Fatal(err)
	}
	server := NewServer(Config{}, store, discardLogger()).(*Server)
	server.syncCodeMailboxBatchWithCursor = func(context.Context, LoginState, []Mailbox, time.Time, string, int) (iCloudIMAPSyncResult, error) {
		return iCloudIMAPSyncResult{MessagesByMailbox: map[string][]ICloudSyncedMessage{}, UIDValidity: "generation"}, nil
	}
	before := store.saveCount
	if _, err := server.syncMailbox(context.Background(), mailbox, time.Time{}, "ChatGPT"); err != nil {
		t.Fatal(err)
	}
	if store.saveCount != before {
		t.Fatalf("empty same-generation sync saved state %d time(s)", store.saveCount-before)
	}
	updated, _ := store.FindMailboxByID(mailbox.ID)
	if updated.LastSyncUID != "100" {
		t.Fatalf("mailbox cursor = %q, want 100", updated.LastSyncUID)
	}
}

func TestKeepAliveDoesNotOverwriteNewerIMAPCursor(t *testing.T) {
	store := newTestStore(t)
	ownerID, accountID := "keepalive-owner", "keepalive-account"
	session := testIMAPSession(ownerID, accountID, "keepalive@icloud.com")
	imapState, _ := iCloudIMAPLoginState(session)
	imapState.IMAPLastSyncUID = "100"
	session = withICloudIMAPLoginState(session, imapState)
	session = withAppleAccountLoginState(session, LoginState{Kind: LoginStateAppleAccount, Scnt: "old", APIKey: "key", LastCheckOK: true})
	if err := store.SaveICloudSessionForOwner(ownerID, session); err != nil {
		t.Fatal(err)
	}
	server := NewServer(Config{}, store, discardLogger()).(*Server)
	server.appleAccountKeepAliveInterval = time.Second
	server.keepAliveAppleAccountState = func(_ context.Context, state LoginState) (LoginState, error) {
		if _, err := store.SetICloudIMAPSyncCursor(ownerID, accountID, imapStateKey(imapState), time.Now(), "200"); err != nil {
			t.Fatal(err)
		}
		state.Scnt = "fresh"
		state.LastCheckedAt = time.Now()
		state.LastCheckOK = true
		return state, nil
	}
	server.keepAliveAppleAccountRound(context.Background())
	updated, ok := store.ICloudSessionForOwnerAccount(ownerID, accountID)
	if !ok {
		t.Fatal("session missing")
	}
	got, _ := iCloudIMAPLoginState(updated)
	if got.IMAPLastSyncUID != "200" {
		t.Fatalf("IMAP cursor = %q, want 200", got.IMAPLastSyncUID)
	}
}

func TestLoginStateUpdateOnlyTouchesRequestedKind(t *testing.T) {
	store := newTestStore(t)
	ownerID, accountID := "state-owner", "state-account"
	session := testIMAPSession(ownerID, accountID, "state@icloud.com")
	session.LoginStates = append(session.LoginStates, LoginState{Kind: LoginStateICloudWeb, Scnt: "web"}, LoginState{Kind: LoginStateAppleAccount, Scnt: "old", APIKey: "old"})
	if err := store.SaveICloudSessionForOwner(ownerID, session); err != nil {
		t.Fatal(err)
	}
	before, _ := store.ICloudSessionForOwnerAccount(ownerID, accountID)
	if _, err := store.UpdateICloudLoginStateForOwner(ownerID, accountID, LoginState{Kind: LoginStateAppleAccount, Scnt: "new", APIKey: "new"}); err != nil {
		t.Fatal(err)
	}
	after, _ := store.ICloudSessionForOwnerAccount(ownerID, accountID)
	beforeIMAP, _ := iCloudIMAPLoginState(before)
	afterIMAP, _ := iCloudIMAPLoginState(after)
	if !reflect.DeepEqual(beforeIMAP, afterIMAP) {
		t.Fatalf("IMAP state changed: before=%+v after=%+v", beforeIMAP, afterIMAP)
	}
	if state, _ := appleAccountLoginState(after); state.APIKey != "new" {
		t.Fatalf("Apple state = %+v", state)
	}
}

func TestStaleSessionSaveCannotRecreateDeletedOwner(t *testing.T) {
	store := newTestStore(t)
	admin, _, _, err := store.BootstrapAdmin("stale-admin", "password", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	user, err := store.CreateUser("stale-user", "password")
	if err != nil {
		t.Fatal(err)
	}
	session := testIMAPSession(user.ID, "", "stale@icloud.com")
	session = withAppleAccountLoginState(session, LoginState{Kind: LoginStateAppleAccount, APIKey: "old"})
	if err := store.SaveICloudSessionForOwner(user.ID, session); err != nil {
		t.Fatal(err)
	}
	stored, _ := store.ICloudSessionForOwner(user.ID)
	if _, err := store.DeleteUser(user.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateICloudLoginStateForOwner(user.ID, stored.AccountID, LoginState{Kind: LoginStateAppleAccount, APIKey: "stale"}); err == nil {
		t.Fatal("stale update recreated deleted owner")
	}
	if _, ok := store.UserByID(admin.ID); !ok || len(store.ICloudSessionsForOwner(user.ID)) != 0 {
		t.Fatal("deleted owner state was recreated")
	}
}

func TestMailWatcherSignatureChangesWithIMAPCredential(t *testing.T) {
	state := LoginState{IMAPEmail: "owner@icloud.com", IMAPUsername: "owner", IMAPHost: "imap.mail.me.com", IMAPPort: 993, IMAPAppPassword: "old"}
	mailboxes := []Mailbox{{ID: "mbx", Email: "alias@icloud.com"}}
	before := mailWatcherIMAPGroupSignature(state, mailboxes)
	state.IMAPAppPassword = "new"
	after := mailWatcherIMAPGroupSignature(state, mailboxes)
	if before == after || strings.Contains(after, "new") {
		t.Fatalf("credential signature before=%q after=%q", before, after)
	}
}

func TestMailWatcherSignatureStillIgnoresSyncCursor(t *testing.T) {
	state := LoginState{IMAPEmail: "owner@icloud.com", IMAPAppPassword: "secret", IMAPLastSyncUID: "1", IMAPUIDValidity: "one"}
	mailboxes := []Mailbox{{ID: "mbx", Email: "alias@icloud.com", LastSyncUID: "1"}}
	before := mailWatcherIMAPGroupSignature(state, mailboxes)
	state.IMAPLastSyncUID = "2"
	state.IMAPUIDValidity = "two"
	mailboxes[0].LastSyncUID = "2"
	if after := mailWatcherIMAPGroupSignature(state, mailboxes); before != after {
		t.Fatalf("sync cursor changed watcher signature: %q vs %q", before, after)
	}
}

func TestMailWatcherRestartsWorkerAfterAppPasswordChange(t *testing.T) {
	store := newTestStore(t)
	ownerID, accountID := "watcher-owner", "watcher-account"
	session := testIMAPSession(ownerID, accountID, "watcher@icloud.com")
	state, _ := iCloudIMAPLoginState(session)
	state.IMAPAppPassword = "old-password"
	session = withICloudIMAPLoginState(session, state)
	if err := store.SaveICloudSessionForOwner(ownerID, session); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMailboxForOwner(ownerID, accountID, "watcher", "watcher.alias@icloud.com"); err != nil {
		t.Fatal(err)
	}
	server := NewServer(Config{}, store, discardLogger()).(*Server)
	passwords := make(chan string, 2)
	server.watchIMAPExists = func(ctx context.Context, state LoginState, _ func()) error {
		passwords <- state.IMAPAppPassword
		<-ctx.Done()
		return ctx.Err()
	}
	workers := make(map[string]mailboxWatcherIdleWorker)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		for _, worker := range workers {
			worker.cancel()
			<-worker.done
		}
	}()

	server.ensureMailWatcherIdleWorkers(ctx, workers)
	if got := <-passwords; got != "old-password" {
		t.Fatalf("first worker password = %q", got)
	}
	state.IMAPAppPassword = "new-password"
	if _, err := store.UpdateICloudLoginStateForOwner(ownerID, accountID, state); err != nil {
		t.Fatal(err)
	}
	server.ensureMailWatcherIdleWorkers(ctx, workers)
	if got := <-passwords; got != "new-password" {
		t.Fatalf("restarted worker password = %q", got)
	}
}

func TestDeleteUserWaitsForOwnerJobsBeforeCommit(t *testing.T) {
	store := newTestStore(t)
	handler := NewServer(Config{}, store, discardLogger())
	server := handler.(*Server)
	adminCookie, _ := registerTestUser(t, handler, "drain-admin", "password")
	_, user := registerTestUser(t, handler, "drain-user", "password")
	_, release, err := server.acquireOwnerRuntime(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/api/admin/users/"+user.ID, nil)
		req.AddCookie(adminCookie)
		handler.ServeHTTP(rr, req)
		done <- rr
	}()
	time.Sleep(20 * time.Millisecond)
	if _, ok := store.UserByID(user.ID); !ok {
		t.Fatal("user deleted before owner job released")
	}
	release()
	if rr := <-done; rr.Code != http.StatusOK {
		t.Fatalf("delete = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestDeleteUserReturnsConflictWhenOwnerJobDoesNotStop(t *testing.T) {
	oldTimeout := ownerDrainTimeout
	ownerDrainTimeout = 20 * time.Millisecond
	t.Cleanup(func() { ownerDrainTimeout = oldTimeout })
	store := newTestStore(t)
	handler := NewServer(Config{}, store, discardLogger())
	server := handler.(*Server)
	adminCookie, _ := registerTestUser(t, handler, "busy-admin", "password")
	_, user := registerTestUser(t, handler, "busy-user", "password")
	_, release, err := server.acquireOwnerRuntime(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/admin/users/"+user.ID, nil)
	req.AddCookie(adminCookie)
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict || !strings.Contains(rr.Body.String(), `"code":"user_busy"`) {
		t.Fatalf("delete = %d body=%s", rr.Code, rr.Body.String())
	}
	if _, ok := store.UserByID(user.ID); !ok {
		t.Fatal("busy user was deleted")
	}
	release()
	if _, releaseAgain, err := server.acquireOwnerRuntime(context.Background(), user.ID); err != nil {
		t.Fatalf("owner gate did not recover: %v", err)
	} else {
		releaseAgain()
	}
}

func TestServerShutdownCancelsAndWaitsForSchedulers(t *testing.T) {
	server := NewServer(Config{}, newTestStore(t), discardLogger()).(*Server)
	started := make(chan struct{})
	stopped := make(chan struct{})
	if !server.startBackground(func(ctx context.Context) {
		close(started)
		<-ctx.Done()
		close(stopped)
	}) {
		t.Fatal("background task was not started")
	}
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-stopped:
	default:
		t.Fatal("Shutdown returned before background task stopped")
	}
}

func TestBackgroundWriteCannotRecreateDeletedOwner(t *testing.T) {
	store := newTestStore(t)
	_, _, _, err := store.BootstrapAdmin("background-admin", "password", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	user, err := store.CreateUser("background-user", "password")
	if err != nil {
		t.Fatal(err)
	}
	session := testIMAPSession(user.ID, "", "background@icloud.com")
	if err := store.SaveICloudSessionForOwner(user.ID, session); err != nil {
		t.Fatal(err)
	}
	session, _ = store.ICloudSessionForOwner(user.ID)
	mailbox, err := store.AddMailboxForOwner(user.ID, session.AccountID, "background", "background.alias@icloud.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.DeleteUser(user.ID); err != nil {
		t.Fatal(err)
	}
	_, err = store.CommitMailboxSync(user.ID, []MailboxSyncUpdate{{
		MailboxID: mailbox.ID,
		Messages:  []ICloudSyncedMessage{{RemoteID: "stale", Body: "stale"}},
		SyncedAt:  time.Now(),
	}}, nil)
	if err == nil {
		t.Fatal("background sync recreated deleted owner data")
	}
	if _, ok := store.UserByID(user.ID); ok || len(store.ICloudSessionsForOwner(user.ID)) != 0 {
		t.Fatal("deleted owner state was recreated")
	}
}

func TestManifestUpdateRequiresSHA256(t *testing.T) {
	manifest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"version":"9999.1","assets":[{"name":"panel","os":"`+runtime.GOOS+`","arch":"`+runtime.GOARCH+`","url":"https://example.invalid/panel"}]}`)
	}))
	defer manifest.Close()
	server := NewServer(Config{UpdateEnabled: true, UpdateManifestURL: manifest.URL}, newTestStore(t), discardLogger()).(*Server)
	candidate, err := server.fetchLatestUpdateCandidate(context.Background(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !candidate.Status.AssetAvailable || candidate.Status.ApplyAvailable {
		t.Fatalf("update status = %+v", candidate.Status)
	}
}

func TestGitHubReleaseLoadsMatchingChecksum(t *testing.T) {
	data := []byte("binary")
	sum := sha256.Sum256(data)
	checksum := hex.EncodeToString(sum[:])
	serverHTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/owner/repo/releases/latest":
			_, _ = io.WriteString(w, `{"tag_name":"v9999","assets":[{"name":"panel_`+runtime.GOOS+`_`+runtime.GOARCH+`","browser_download_url":"https://example.invalid/bin"},{"name":"other.sha256","browser_download_url":"`+"http://"+r.Host+`/other"},{"name":"checksums.txt","browser_download_url":"`+"http://"+r.Host+`/checksums"}]}`)
		case "/other":
			_, _ = io.WriteString(w, strings.Repeat("0", 64)+"\n")
		case "/checksums":
			_, _ = io.WriteString(w, checksum+"  panel_"+runtime.GOOS+"_"+runtime.GOARCH+"\n")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer serverHTTP.Close()
	oldBase := updateGitHubAPIBaseURL
	updateGitHubAPIBaseURL = serverHTTP.URL
	t.Cleanup(func() { updateGitHubAPIBaseURL = oldBase })
	server := &Server{cfg: Config{UpdateEnabled: true, UpdateRepository: "owner/repo"}}
	candidate, err := server.fetchGitHubReleaseUpdateCandidate(context.Background(), publicUpdateStatus{Current: currentVersionInfo()})
	if err != nil {
		t.Fatal(err)
	}
	if candidate.SHA256 != checksum || !candidate.Status.ApplyAvailable {
		t.Fatalf("candidate = %+v", candidate)
	}
}

func TestUpdateRejectsMissingOrMismatchedChecksum(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "panel")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	download := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "new") }))
	defer download.Close()
	for _, checksum := range []string{"", strings.Repeat("0", 64)} {
		if err := downloadAndReplaceExecutable(context.Background(), download.URL, checksum, exe); err == nil {
			t.Fatalf("checksum %q was accepted", checksum)
		}
		data, err := os.ReadFile(exe)
		if err != nil || string(data) != "old" {
			t.Fatalf("original executable changed: %q err=%v", data, err)
		}
	}
}

func TestUpdateButtonRequiresApplyAvailable(t *testing.T) {
	page, err := webFS.ReadFile("templates/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), "status.update_available && status.apply_available") {
		t.Fatal("update button does not require a verified update asset")
	}
}

func TestGenerateAppleHashcashHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	if _, err := generateAppleHashcash(ctx, 24, "challenge", time.Now()); err == nil || time.Since(started) > 200*time.Millisecond {
		t.Fatalf("canceled hashcash err=%v elapsed=%v", err, time.Since(started))
	}
}

func TestGenerateAppleHashcashContinuesPastExpectedWork(t *testing.T) {
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	got, err := generateAppleHashcash(context.Background(), 8, "boundary-2", now)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(got, ":")
	if len(parts) != 6 || parts[5] != "7g" {
		t.Fatalf("hashcash = %q, want first valid counter 7g (268)", got)
	}
}

func TestStorePrunesExpiredWebSessionsOnLoadAndCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	now := time.Now()
	state := State{NextID: 2, Users: []User{{ID: "usr_1", Username: "user", Status: StatusActive}}, WebSessions: []WebSession{
		{TokenHash: "expired", UserID: "usr_1", ExpiresAt: now.Add(-time.Hour)},
		{TokenHash: "valid", UserID: "usr_1", ExpiresAt: now.Add(time.Hour)},
	}}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if sessions := store.Snapshot().WebSessions; len(sessions) != 1 || sessions[0].TokenHash != "valid" {
		t.Fatalf("loaded sessions = %+v", sessions)
	}
	store.mu.Lock()
	store.state.WebSessions = append(store.state.WebSessions, WebSession{TokenHash: "expired-again", UserID: "usr_1", ExpiresAt: now.Add(-time.Minute)})
	store.mu.Unlock()
	if _, _, err := store.CreateWebSession("usr_1", false, time.Hour); err != nil {
		t.Fatal(err)
	}
	for _, session := range store.Snapshot().WebSessions {
		if !session.ExpiresAt.After(time.Now()) {
			t.Fatalf("expired session remained: %+v", session)
		}
	}
}

func TestIMAPParallelDialClosesLosingSuccessfulConnections(t *testing.T) {
	winner := &countingConn{}
	loser := &countingConn{}
	started := make(chan struct{}, 2)
	releaseWinner := make(chan struct{})
	dial := func(ctx context.Context, address string) (net.Conn, error) {
		started <- struct{}{}
		if strings.HasPrefix(address, "192.0.2.1:") {
			<-releaseWinner
			return winner, nil
		}
		<-ctx.Done()
		return loser, nil
	}
	done := make(chan net.Conn, 1)
	go func() {
		conn, _ := dialICloudIMAPTLSIPsWithDialer(context.Background(), "imap.example", 993, []net.IP{net.ParseIP("192.0.2.1"), net.ParseIP("192.0.2.2")}, dial)
		done <- conn
	}()
	<-started
	<-started
	close(releaseWinner)
	if got := <-done; got != winner {
		t.Fatalf("winner = %v, want first connection", got)
	}
	if loser.closed.Load() != 1 || winner.closed.Load() != 0 {
		t.Fatalf("close counts winner=%d loser=%d", winner.closed.Load(), loser.closed.Load())
	}
}

type countingConn struct {
	closed atomic.Int32
}

func (c *countingConn) Read([]byte) (int, error)         { return 0, io.EOF }
func (c *countingConn) Write(p []byte) (int, error)      { return len(p), nil }
func (c *countingConn) Close() error                     { c.closed.Add(1); return nil }
func (c *countingConn) LocalAddr() net.Addr              { return testAddr("local") }
func (c *countingConn) RemoteAddr() net.Addr             { return testAddr("remote") }
func (c *countingConn) SetDeadline(time.Time) error      { return nil }
func (c *countingConn) SetReadDeadline(time.Time) error  { return nil }
func (c *countingConn) SetWriteDeadline(time.Time) error { return nil }

type testAddr string

func (a testAddr) Network() string { return string(a) }
func (a testAddr) String() string  { return string(a) }
