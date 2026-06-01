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
	routes := mockRoutes()
	out := make([]string, 0, len(routes)+1)
	out = append(out, "GET  /healthz")
	for _, route := range routes {
		out = append(out, route.display)
	}
	return out
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
	for _, route := range mockRoutes() {
		if r.Method == route.method && path == route.path {
			route.handle(w, r)
			return
		}
	}
	write(w, http.StatusNotFound, map[string]any{"ErrorCode": 12, "Message": "No mock route for " + r.Method + " " + r.URL.Path})
}

type route struct {
	method  string
	path    string
	display string
	handle  func(http.ResponseWriter, *http.Request)
}

func mockRoutes() []route {
	return []route{
		listRoute(http.MethodGet, "/servers", "Servers", staticList(servers)),
		itemRoute(http.MethodGet, "/servers/101", "GET  /servers/{id}", func() any { return servers()[0] }),
		listRoute(http.MethodGet, "/message-streams", "MessageStreams", staticList(streams)),
		itemRoute(http.MethodGet, "/message-streams/outbound", "GET  /message-streams/{id}", func() any { return streams()[0] }),
		listRoute(http.MethodGet, "/domains", "Domains", staticList(domains)),
		itemRoute(http.MethodGet, "/domains/501", "GET  /domains/{id}", func() any { return domains()[0] }),
		itemRoute(http.MethodPost, "/domains/501/verifyDkim", "POST /domains/{id}/verifyDkim", func() any { return map[string]any{"DKIMVerified": true} }),
		itemRoute(http.MethodPost, "/domains/501/verifySPF", "POST /domains/{id}/verifySPF", func() any { return map[string]any{"SPFVerified": true} }),
		listRoute(http.MethodGet, "/senders", "SenderSignatures", staticList(senders)),
		itemRoute(http.MethodGet, "/senders/601", "GET  /senders/{id}", func() any { return senders()[0] }),
		listRoute(http.MethodGet, "/webhooks", "Webhooks", staticList(webhooks)),
		itemRoute(http.MethodGet, "/webhooks/701", "GET  /webhooks/{id}", func() any { return webhooks()[0] }),
		itemRoute(http.MethodGet, "/deliverystats", "GET  /deliverystats", func() any {
			return map[string]any{"InactiveMails": 2, "Bounces": []any{map[string]any{"Type": "HardBounce", "Name": "Hard bounce", "Count": 1}}}
		}),
		listRoute(http.MethodGet, "/messages/outbound", "Messages", filterMessages),
		listRoute(http.MethodGet, "/messages/outbound/opens", "Opens", staticList(opens)),
		listRoute(http.MethodGet, "/messages/outbound/opens/msg-1", "Opens", staticList(opens), display("GET  /messages/outbound/opens/{id}")),
		listRoute(http.MethodGet, "/messages/outbound/clicks", "Clicks", staticList(clicks)),
		listRoute(http.MethodGet, "/messages/outbound/clicks/msg-1", "Clicks", staticList(clicks), display("GET  /messages/outbound/clicks/{id}")),
		itemRoute(http.MethodGet, "/messages/outbound/msg-1/details", "GET  /messages/outbound/{id}/details", func() any { return messages()[0] }),
		itemRoute(http.MethodGet, "/messages/outbound/msg-1/dump", "GET  /messages/outbound/{id}/dump", func() any {
			return map[string]any{"Body": "raw outbound message with recipient user@example.com"}
		}),
		listRoute(http.MethodGet, "/messages/inbound", "InboundMessages", staticList(inboundMessages)),
		itemRoute(http.MethodGet, "/messages/inbound/in-1/details", "GET  /messages/inbound/{id}/details", func() any {
			return map[string]any{"MessageID": "in-1", "From": "reply@example.com", "To": "support@example.com", "TextBody": "hello"}
		}),
		itemRoute(http.MethodPut, "/messages/inbound/in-1/retry", "PUT  /messages/inbound/{id}/retry", func() any {
			return map[string]any{"MessageID": "in-1", "Status": "Queued"}
		}),
		itemRoute(http.MethodPut, "/messages/inbound/in-1/bypass", "PUT  /messages/inbound/{id}/bypass", func() any {
			return map[string]any{"MessageID": "in-1", "Status": "Bypassed"}
		}),
		listRoute(http.MethodGet, "/bounces", "Bounces", filterBounces),
		itemRoute(http.MethodGet, "/bounces/9001", "GET  /bounces/{id}", func() any { return bounces()[0] }),
		itemRoute(http.MethodGet, "/bounces/9001/dump", "GET  /bounces/{id}/dump", func() any {
			return map[string]any{"Body": "smtp bounce dump with recipient user@example.com"}
		}),
		itemRoute(http.MethodPut, "/bounces/9001/activate", "PUT  /bounces/{id}/activate", func() any {
			return map[string]any{"Message": "OK", "BounceID": 9001}
		}),
		listRoute(http.MethodGet, "/message-streams/outbound/suppressions/dump", "Suppressions", staticList(suppressions), display("GET  /message-streams/{stream}/suppressions/dump")),
		itemRoute(http.MethodPost, "/message-streams/outbound/suppressions", "POST /message-streams/{stream}/suppressions", func() any {
			return map[string]any{"Status": "created", "EmailAddress": "manual@example.com"}
		}),
		itemRoute(http.MethodPost, "/message-streams/outbound/suppressions/delete", "POST /message-streams/{stream}/suppressions/delete", func() any {
			return map[string]any{"Status": "deleted"}
		}),
	}
}

type routeOption func(*route)

func display(value string) routeOption {
	return func(route *route) {
		route.display = value
	}
}

func staticList(items func() []any) func(*http.Request) []any {
	return func(*http.Request) []any {
		return items()
	}
}

func listRoute(method, path, field string, items func(*http.Request) []any, opts ...routeOption) route {
	route := route{
		method:  method,
		path:    path,
		display: displayPath(method, path),
		handle: func(w http.ResponseWriter, r *http.Request) {
			if path == "/servers" && r.URL.Query().Get("offset") == "" {
				write(w, http.StatusUnprocessableEntity, map[string]any{"ErrorCode": 600, "Message": "Parameter 'offset' is required but has been left out"})
				return
			}
			writeList(w, field, items(r), r)
		},
	}
	for _, opt := range opts {
		opt(&route)
	}
	return route
}

func itemRoute(method, path, display string, item func() any) route {
	return route{
		method:  method,
		path:    path,
		display: display,
		handle: func(w http.ResponseWriter, r *http.Request) {
			write(w, http.StatusOK, item())
		},
	}
}

func displayPath(method, path string) string {
	spacing := " "
	if len(method) == 3 {
		spacing = "  "
	}
	return method + spacing + path
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
