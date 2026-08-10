package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestMailboxListClampsPageAfterLastItemDeleted(t *testing.T) {
	store := newTestStore(t)
	handler := NewServer(Config{}, store, discardLogger())
	cookie, _ := registerTestUser(t, handler, "page-admin", "password")
	for index := 0; index < 10; index++ {
		createTestMailboxWithCookie(t, handler, cookie, "page", fmt.Sprintf("page-%d@icloud.com", index))
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/mailboxes?page=2&page_size=10", nil)
	req.AddCookie(cookie)
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list mailboxes = %d body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Mailboxes  []publicMailbox  `json:"mailboxes"`
		Pagination publicPagination `json:"pagination"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Pagination.Page != 1 || body.Pagination.TotalPages != 1 || len(body.Mailboxes) != 10 {
		t.Fatalf("pagination = %+v mailboxes=%d, want page 1/1 with 10 rows", body.Pagination, len(body.Mailboxes))
	}
}

func TestFrontendAsyncActionShowsFailureAndPreventsDuplicate(t *testing.T) {
	page, err := webFS.ReadFile("templates/index.html")
	if err != nil {
		t.Fatal(err)
	}
	source := string(page)
	start := strings.Index(source, "async function startProtocolLogin()")
	end := strings.Index(source, "async function submitProtocol2FA()")
	if start < 0 || end <= start {
		t.Fatal("startProtocolLogin function not found")
	}
	functionSource := source[start:end]
	script := `
const vm = require('vm');
const elements = {
  protocolAppleId: {value: 'owner@icloud.com'},
  protocolPassword: {value: 'secret'},
  protocolLoginButton: {disabled: false},
  icloudSessionInfo: {textContent: ''}
};
const logs = [];
let calls = 0;
let rejectRequest;
const context = {
  JSON, Error, Promise,
  protocolPendingID: '',
  $: id => elements[id] || (elements[id] = {value: '', textContent: '', disabled: false}),
  api: () => {
    calls++;
    return new Promise((resolve, reject) => { rejectRequest = reject; });
  },
  twoFactorMethodPayloadValue: () => '',
  log: message => logs.push(String(message)),
  renderICloudSession: () => {},
  renderICloudSessions: () => {},
  refresh: async () => {}
};
vm.createContext(context);
vm.runInContext(` + strconv.Quote(functionSource) + `, context);
(async () => {
  const first = vm.runInContext('startProtocolLogin()', context);
  const second = vm.runInContext('startProtocolLogin()', context);
  if (calls !== 1) throw new Error('API calls=' + calls + ', want 1');
  if (!elements.protocolLoginButton.disabled) throw new Error('button was not disabled');
  rejectRequest(new Error('network failed'));
  await Promise.allSettled([first, second]);
  if (elements.protocolLoginButton.disabled) throw new Error('button was not restored');
  if (!elements.icloudSessionInfo.textContent.includes('network failed')) throw new Error('failure status not visible');
  if (!logs.some(line => line.includes('network failed'))) throw new Error('failure log not visible');
})().catch(err => { console.error(err); process.exitCode = 1; });
`
	command := exec.Command("node", "-e", script)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("frontend async regression failed: %v\n%s", err, output)
	}
}

func TestPanelMailboxCodeUsesWebSessionAndOwnerScope(t *testing.T) {
	store := newTestStore(t)
	handler := NewServer(Config{}, store, discardLogger())
	_, _ = registerTestUser(t, handler, "code-admin", "password")
	ownerCookie, _ := registerTestUser(t, handler, "code-owner", "password")
	otherCookie, _ := registerTestUser(t, handler, "code-other", "password")
	mailbox := createTestMailboxWithCookie(t, handler, ownerCookie, "code", "panel-code@icloud.com")
	if _, err := store.AddMessage(mailbox.ID, "OpenAI verification code", "OpenAI", "Your code is 123456", time.Now()); err != nil {
		t.Fatal(err)
	}

	request := func(cookie *http.Cookie) *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/mailboxes/"+mailbox.ID+"/code?cache=1", nil)
		req.AddCookie(cookie)
		handler.ServeHTTP(rr, req)
		return rr
	}
	if rr := request(ownerCookie); rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"code":"123456"`) {
		t.Fatalf("owner code query = %d body=%s", rr.Code, rr.Body.String())
	}
	if rr := request(otherCookie); rr.Code != http.StatusUnauthorized {
		t.Fatalf("other owner code query = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPanelMailboxCodeURLIsSameOrigin(t *testing.T) {
	page, err := webFS.ReadFile("templates/index.html")
	if err != nil {
		t.Fatal(err)
	}
	source := string(page)
	start := strings.Index(source, "function panelMailboxCodeURL(")
	end := strings.Index(source, "function mailboxCodeURL(")
	if start < 0 || end <= start {
		t.Fatal("panelMailboxCodeURL function not found")
	}
	functionSource := source[start:end]
	script := `
const vm = require('vm');
const context = {URL, location: {origin: 'https://panel.example'}, encodeURIComponent};
vm.createContext(context);
vm.runInContext(` + strconv.Quote(functionSource) + `, context);
const value = vm.runInContext("panelMailboxCodeURL({id:'mbx 1',api_url:'http://external.example/code?key=secret'},{waitMs:12000}).toString()", context);
if (!value.startsWith('https://panel.example/api/mailboxes/mbx%201/code?')) throw new Error('unexpected panel URL: ' + value);
if (value.includes('external.example') || value.includes('key=secret')) throw new Error('external API URL leaked into panel request: ' + value);
`
	if output, err := exec.Command("node", "-e", script).CombinedOutput(); err != nil {
		t.Fatalf("panel code URL regression failed: %v\n%s", err, output)
	}
}

func TestMailboxAPIURLUsesConfiguredPublicBaseURLOrRelativePath(t *testing.T) {
	mailbox := Mailbox{Email: "alias@icloud.com", APIToken: "secret"}
	req := httptest.NewRequest(http.MethodGet, "http://untrusted.example/api/mailboxes", nil)
	req.Host = "attacker.example"
	if got := (&Server{}).mailboxAPIURL(req, mailbox); got != "/api/v1/mailboxes/alias@icloud.com/code?key=secret" {
		t.Fatalf("relative mailbox API URL = %q", got)
	}
	server := &Server{cfg: Config{PublicBaseURL: "https://public.example/base"}}
	if got := server.mailboxAPIURL(req, mailbox); got != "https://public.example/base/api/v1/mailboxes/alias@icloud.com/code?key=secret" {
		t.Fatalf("configured mailbox API URL = %q", got)
	}
}

func TestLoadConfigRejectsInvalidPublicBaseURL(t *testing.T) {
	for _, value := range []string{
		"//missing-scheme.example",
		"javascript://example.com",
		"https:///missing-host",
		"https://user@public.example/base",
		"https://public.example/base?x=1",
		"https://public.example/base?",
		"https://public.example/base#fragment",
		"https://public.example/base#",
	} {
		t.Run(value, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(fmt.Sprintf(`{"public_base_url":%q}`, value)), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadConfig(path); err == nil {
				t.Fatalf("public_base_url %q was accepted", value)
			}
		})
	}
	t.Setenv("IPM_PUBLIC_BASE_URL", "missing-host")
	if _, err := LoadConfig(""); err == nil {
		t.Fatal("invalid environment public_base_url was accepted")
	}
	if _, err := LoadConfig(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("invalid environment public_base_url bypassed validation when config file was missing")
	}
}

func TestLoadConfigAcceptsValidPublicBaseURL(t *testing.T) {
	for _, test := range []struct {
		value string
		want  string
	}{
		{value: "https://public.example", want: "https://public.example"},
		{value: "HTTPS://public.example", want: "HTTPS://public.example"},
		{value: "https://public.example/base/", want: "https://public.example/base"},
		{value: "https://public.example/base%3Fx", want: "https://public.example/base%3Fx"},
	} {
		t.Run(test.value, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(fmt.Sprintf(`{"public_base_url":%q}`, test.value)), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := LoadConfig(path)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.PublicBaseURL != test.want {
				t.Fatalf("public_base_url = %q, want %q", cfg.PublicBaseURL, test.want)
			}
		})
	}
}

func TestTemplateInlineScriptsCompile(t *testing.T) {
	for _, name := range []string{"index.html", "login.html", "manage.html"} {
		t.Run(name, func(t *testing.T) {
			page, err := webFS.ReadFile("templates/" + name)
			if err != nil {
				t.Fatal(err)
			}
			source := string(page)
			start := strings.Index(source, "<script>")
			end := strings.LastIndex(source, "</script>")
			if start < 0 || end <= start {
				t.Fatal("inline script not found")
			}
			path := filepath.Join(t.TempDir(), strings.TrimSuffix(name, ".html")+".js")
			if err := os.WriteFile(path, []byte(source[start+len("<script>"):end]), 0o600); err != nil {
				t.Fatal(err)
			}
			if output, err := exec.Command("node", "--check", path).CombinedOutput(); err != nil {
				t.Fatalf("inline script syntax error: %v\n%s", err, output)
			}
		})
	}
}
