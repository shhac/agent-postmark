package mockpostmark

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func NewServer() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handle)
	return mux
}

func Routes() []string {
	return []string{
		"GET  /healthz",
		"GET  /servers",
		"GET  /servers/{id}",
		"GET  /servers/{id}/message-streams",
		"GET  /domains",
		"GET  /domains/{id}",
		"POST /domains/{id}/verifyDkim",
		"POST /domains/{id}/verifySPF",
		"GET  /senders",
		"GET  /senders/{id}",
		"GET  /webhooks",
		"GET  /webhooks/{id}",
		"GET  /deliverystats",
		"GET  /messages/outbound",
		"GET  /messages/outbound/opens",
		"GET  /messages/outbound/clicks",
		"GET  /messages/outbound/{id}/details",
		"GET  /messages/inbound",
		"GET  /messages/inbound/{id}/details",
		"PUT  /messages/inbound/{id}/retry",
		"PUT  /messages/inbound/{id}/bypass",
		"GET  /bounces",
		"GET  /bounces/{id}",
		"GET  /message-streams/{stream}/suppressions/dump",
		"POST /message-streams/{stream}/suppressions",
		"POST /message-streams/{stream}/suppressions/delete",
	}
}

func handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Mockpostmark", "true")
	if r.URL.Path == "/" {
		write(w, http.StatusOK, map[string]any{"service": "mockpostmark", "routes": Routes()})
		return
	}
	if r.URL.Path == "/healthz" {
		write(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	if !authorized(r) {
		write(w, http.StatusUnauthorized, map[string]any{"ErrorCode": 10, "Message": "Bad or missing API token"})
		return
	}
	dispatch(w, r)
}

func authorized(r *http.Request) bool {
	account := r.Header.Get("X-Postmark-Account-Token")
	server := r.Header.Get("X-Postmark-Server-Token")
	if strings.Contains(account, "invalid") || strings.Contains(server, "invalid") {
		return false
	}
	return account != "" || server != ""
}

func dispatch(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	switch {
	case r.Method == http.MethodGet && path == "/servers":
		writeList(w, "Servers", servers(), r)
	case r.Method == http.MethodGet && path == "/servers/101":
		write(w, http.StatusOK, servers()[0])
	case r.Method == http.MethodGet && path == "/servers/101/message-streams":
		writeList(w, "MessageStreams", streams(), r)
	case r.Method == http.MethodGet && path == "/domains":
		writeList(w, "Domains", domains(), r)
	case r.Method == http.MethodGet && path == "/domains/501":
		write(w, http.StatusOK, domains()[0])
	case r.Method == http.MethodPost && path == "/domains/501/verifyDkim":
		write(w, http.StatusOK, map[string]any{"DKIMVerified": true})
	case r.Method == http.MethodPost && path == "/domains/501/verifySPF":
		write(w, http.StatusOK, map[string]any{"SPFVerified": true})
	case r.Method == http.MethodGet && path == "/senders":
		writeList(w, "SenderSignatures", senders(), r)
	case r.Method == http.MethodGet && path == "/senders/601":
		write(w, http.StatusOK, senders()[0])
	case r.Method == http.MethodGet && path == "/webhooks":
		writeList(w, "Webhooks", webhooks(), r)
	case r.Method == http.MethodGet && path == "/webhooks/701":
		write(w, http.StatusOK, webhooks()[0])
	case r.Method == http.MethodGet && path == "/deliverystats":
		write(w, http.StatusOK, map[string]any{"InactiveMails": 2, "Bounces": []any{map[string]any{"Type": "HardBounce", "Name": "Hard bounce", "Count": 1}}})
	case r.Method == http.MethodGet && path == "/messages/outbound":
		writeList(w, "Messages", filterMessages(r), r)
	case r.Method == http.MethodGet && path == "/messages/outbound/opens":
		writeList(w, "Opens", opens(), r)
	case r.Method == http.MethodGet && path == "/messages/outbound/clicks":
		writeList(w, "Clicks", clicks(), r)
	case r.Method == http.MethodGet && path == "/messages/outbound/msg-1/details":
		write(w, http.StatusOK, messages()[0])
	case r.Method == http.MethodGet && path == "/messages/inbound":
		writeList(w, "InboundMessages", inboundMessages(), r)
	case r.Method == http.MethodGet && path == "/messages/inbound/in-1/details":
		write(w, http.StatusOK, map[string]any{"MessageID": "in-1", "From": "reply@example.com", "To": "support@example.com", "TextBody": "hello"})
	case r.Method == http.MethodPut && path == "/messages/inbound/in-1/retry":
		write(w, http.StatusOK, map[string]any{"MessageID": "in-1", "Status": "Queued"})
	case r.Method == http.MethodPut && path == "/messages/inbound/in-1/bypass":
		write(w, http.StatusOK, map[string]any{"MessageID": "in-1", "Status": "Bypassed"})
	case r.Method == http.MethodGet && path == "/bounces":
		writeList(w, "Bounces", filterBounces(r), r)
	case r.Method == http.MethodGet && path == "/bounces/9001":
		write(w, http.StatusOK, bounces()[0])
	case r.Method == http.MethodGet && path == "/message-streams/outbound/suppressions/dump":
		writeList(w, "Suppressions", suppressions(), r)
	case r.Method == http.MethodPost && path == "/message-streams/outbound/suppressions":
		write(w, http.StatusOK, map[string]any{"Status": "created", "EmailAddress": "manual@example.com"})
	case r.Method == http.MethodPost && path == "/message-streams/outbound/suppressions/delete":
		write(w, http.StatusOK, map[string]any{"Status": "deleted"})
	default:
		write(w, http.StatusNotFound, map[string]any{"ErrorCode": 12, "Message": "No mock route for " + r.Method + " " + r.URL.Path})
	}
}

func servers() []any {
	return []any{
		map[string]any{"ID": 101, "Name": "Production", "Color": "Blue", "ServerLink": "https://postmarkapp.com/servers/101"},
	}
}

func streams() []any {
	return []any{
		map[string]any{"ID": "outbound", "Name": "Transactional", "ServerID": 101, "MessageStreamType": "Transactional", "ArchivedAt": nil},
		map[string]any{"ID": "broadcasts", "Name": "Broadcasts", "ServerID": 101, "MessageStreamType": "Broadcasts", "ArchivedAt": nil},
	}
}

func domains() []any {
	return []any{map[string]any{"ID": 501, "Name": "example.com", "DKIMVerified": true, "SPFVerified": true, "ReturnPathDomainVerified": true}}
}

func senders() []any {
	return []any{map[string]any{"ID": 601, "Name": "Support", "EmailAddress": "support@example.com", "Confirmed": true, "DKIMVerified": true}}
}

func webhooks() []any {
	return []any{map[string]any{"ID": 701, "Url": "https://example.com/postmark", "MessageStream": "outbound", "Triggers": map[string]any{"Delivery": true, "Bounce": true}}}
}

func messages() []any {
	return []any{
		map[string]any{"MessageID": "msg-1", "To": "user@example.com", "From": "support@example.com", "Subject": "Welcome", "Status": "Sent", "MessageStream": "outbound", "ReceivedAt": "2026-05-31T10:00:00Z", "HtmlBody": "<p>secret body</p>", "LastOpen": "2026-05-31T10:03:00Z", "LastClick": "2026-05-31T10:04:00Z"},
	}
}

func inboundMessages() []any {
	return []any{
		map[string]any{"MessageID": "in-1", "From": "reply@example.com", "To": "support@example.com", "Subject": "Re: Welcome", "Status": "Processed", "MailboxHash": "support", "Date": "2026-05-31T10:05:00Z", "TextBody": "secret inbound body"},
	}
}

func bounces() []any {
	return []any{
		map[string]any{"ID": 9001, "Type": "HardBounce", "Name": "Hard bounce", "Email": "user@example.com", "MessageID": "msg-1", "Inactive": true, "CanActivate": true, "MessageStream": "outbound", "BouncedAt": "2026-05-31T10:01:00Z"},
	}
}

func opens() []any {
	return []any{map[string]any{"MessageID": "msg-1", "ReceivedAt": "2026-05-31T10:00:00Z", "FirstOpen": "2026-05-31T10:03:00Z", "LastOpen": "2026-05-31T10:03:00Z", "TotalOpens": 1, "UniqueOpens": 1, "OS": "macOS", "Platform": "Apple Mail"}}
}

func clicks() []any {
	return []any{map[string]any{"MessageID": "msg-1", "ReceivedAt": "2026-05-31T10:00:00Z", "FirstClick": "2026-05-31T10:04:00Z", "LastClick": "2026-05-31T10:04:00Z", "TotalClicks": 1, "UniqueClicks": 1, "OriginalLink": "https://example.com/welcome", "OS": "macOS", "Platform": "Apple Mail"}}
}

func suppressions() []any {
	return []any{map[string]any{"EmailAddress": "user@example.com", "SuppressionReason": "HardBounce", "Origin": "Recipient", "CreatedAt": "2026-05-31T10:01:00Z"}}
}

func filterMessages(r *http.Request) []any {
	items := messages()
	if to := r.URL.Query().Get("recipient"); to != "" && to != "user@example.com" {
		return []any{}
	}
	return items
}

func filterBounces(r *http.Request) []any {
	items := bounces()
	if email := r.URL.Query().Get("emailFilter"); email != "" && email != "user@example.com" {
		return []any{}
	}
	return items
}

func writeList(w http.ResponseWriter, field string, items []any, r *http.Request) {
	count, _ := strconv.Atoi(r.URL.Query().Get("count"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if count <= 0 {
		count = len(items)
	}
	if offset > len(items) {
		offset = len(items)
	}
	end := offset + count
	if end > len(items) {
		end = len(items)
	}
	write(w, http.StatusOK, map[string]any{"TotalCount": len(items), field: items[offset:end]})
}

func write(w http.ResponseWriter, status int, payload any) {
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(payload)
}
