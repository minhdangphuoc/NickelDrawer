package ghsync

import (
	"html/template"
	"net/http"
	"net/url"
	"strconv"

	"koboapps/internal/i18n"
)

func itoa(number int) string { return strconv.Itoa(number) }

func queryEscape(value string) string { return url.QueryEscape(value) }

// pageForm is everything a pairing page shows. It has no token field on purpose:
// the token can never be echoed back.
type pageForm struct {
	Status string // "", "ok", "locked", "expired", "done"
	Error  string
	Repo   string
	Branch string
	Folder string
}

// pageView is pageForm plus the localized labels.
type pageView struct {
	pageForm
	Lang string
	Text map[string]string
}

// pairPageMessages maps template names to full message keys. Keys are spelled
// out in full so tooling (and the locale-coverage test) can find every one.
var pairPageMessages = map[string]string{
	"title":         "msg.pair_title",
	"intro":         "msg.pair_intro",
	"warning":       "msg.pair_warning",
	"label_code":    "msg.pair_label_code",
	"hint_code":     "msg.pair_hint_code",
	"label_repo":    "msg.pair_label_repo",
	"hint_repo":     "msg.pair_hint_repo",
	"label_branch":  "msg.pair_label_branch",
	"hint_branch":   "msg.pair_hint_branch",
	"label_folder":  "msg.pair_label_folder",
	"hint_folder":   "msg.pair_hint_folder",
	"label_token":   "msg.pair_label_token",
	"hint_token":    "msg.pair_hint_token",
	"submit":        "msg.pair_submit",
	"ok_title":      "msg.pair_ok_title",
	"ok_body":       "msg.pair_ok_body",
	"locked_title":  "msg.pair_locked_title",
	"locked_body":   "msg.pair_locked_body",
	"expired_title": "msg.pair_expired_title",
	"expired_body":  "msg.pair_expired_body",
	"done_title":    "msg.pair_done_title",
	"done_body":     "msg.pair_done_body",
}

func (session *pairSession) render(w http.ResponseWriter, status int, page pageForm) {
	view := pageView{pageForm: page, Lang: session.pageLanguage(), Text: map[string]string{}}
	for name, key := range pairPageMessages {
		view.Text[name] = session.options.Messages.Message(key, i18n.Params("repo", page.Repo))
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := pairPageTemplate.Execute(w, view); err != nil {
		session.options.Log.Printf("pairing: cannot render page: %v", err)
	}
}

func (session *pairSession) pageLanguage() string {
	if session.options.Locale == "" {
		return i18n.SourceLocale
	}
	return session.options.Locale
}

var pairPageTemplate = template.Must(template.New("pair").Parse(`<!doctype html>
<html lang="{{.Lang}}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex">
<title>{{.Text.title}}</title>
<style>
:root{color-scheme:light dark;--bg:#fff;--fg:#1b1b1b;--muted:#5a5a5a;--line:#c8c8c8;--accent:#0b57d0;--warn:#fff3cd;--warnfg:#4d3b00;--bad:#b3261e}
@media(prefers-color-scheme:dark){:root{--bg:#121212;--fg:#ececec;--muted:#a8a8a8;--line:#4a4a4a;--accent:#8ab4f8;--warn:#3b3200;--warnfg:#ffe08a;--bad:#f2b8b5}}
*{box-sizing:border-box}
body{margin:0;padding:16px;background:var(--bg);color:var(--fg);font:17px/1.45 system-ui,-apple-system,sans-serif}
main{max-width:30rem;margin:0 auto}
h1{font-size:1.4rem;margin:.2em 0 .6em}
label{display:block;font-weight:600;margin-top:1em}
small{display:block;color:var(--muted);font-weight:400}
input{width:100%;margin-top:.3em;padding:.7em;font-size:16px;color:var(--fg);background:var(--bg);border:1px solid var(--line);border-radius:8px}
button{width:100%;margin-top:1.4em;padding:.9em;font-size:17px;font-weight:600;color:#fff;background:var(--accent);border:0;border-radius:8px}
@media(prefers-color-scheme:dark){button{color:#000}}
.warn{padding:.8em;background:var(--warn);color:var(--warnfg);border-radius:8px}
.error{padding:.8em;border:2px solid var(--bad);color:var(--bad);border-radius:8px;font-weight:600}
</style>
</head>
<body>
<main>
{{if eq .Status "ok"}}
<h1>{{.Text.ok_title}}</h1>
<p>{{.Text.ok_body}}</p>
{{else if eq .Status "locked"}}
<h1>{{.Text.locked_title}}</h1>
<p>{{.Text.locked_body}}</p>
{{else if eq .Status "expired"}}
<h1>{{.Text.expired_title}}</h1>
<p>{{.Text.expired_body}}</p>
{{else if eq .Status "done"}}
<h1>{{.Text.done_title}}</h1>
<p>{{.Text.done_body}}</p>
{{else}}
<h1>{{.Text.title}}</h1>
<p>{{.Text.intro}}</p>
<p class="warn">{{.Text.warning}}</p>
{{if .Error}}<p class="error" role="alert">{{.Error}}</p>{{end}}
<form method="post" action="/" autocomplete="off">
<label for="code">{{.Text.label_code}}<small>{{.Text.hint_code}}</small></label>
<input id="code" name="code" inputmode="numeric" pattern="[0-9]{6}" maxlength="6" autocomplete="off" required>
<label for="repo">{{.Text.label_repo}}<small>{{.Text.hint_repo}}</small></label>
<input id="repo" name="repo" value="{{.Repo}}" autocapitalize="none" autocorrect="off" spellcheck="false" placeholder="owner/name" required>
<label for="branch">{{.Text.label_branch}}<small>{{.Text.hint_branch}}</small></label>
<input id="branch" name="branch" value="{{.Branch}}" autocapitalize="none" autocorrect="off" spellcheck="false">
<label for="folder">{{.Text.label_folder}}<small>{{.Text.hint_folder}}</small></label>
<input id="folder" name="folder" value="{{.Folder}}" autocapitalize="none" autocorrect="off" spellcheck="false">
<label for="token">{{.Text.label_token}}<small>{{.Text.hint_token}}</small></label>
<input id="token" name="token" type="password" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" required>
<button type="submit">{{.Text.submit}}</button>
</form>
{{end}}
</main>
</body>
</html>
`))
