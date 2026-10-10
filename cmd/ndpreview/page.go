package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
	"image/png"
	"net/url"
	"strings"
)

// viewOptions are the query parameters of the preview page.
type viewOptions struct {
	Screen  string
	Locale  string
	Page    int
	Scale   int // 50 or 100 (percent)
	Grey    int // 0 for pure black and white, 16 for 16 grey levels
	Toast   string
	Dialog  string
	Confirm string
	Value   string // screen value (viewer picture id)
	Chrome  string // viewer chrome: shown or hidden
}

func (o viewOptions) query(changes map[string]string) string {
	values := url.Values{}
	set := func(key, value string) {
		if override, has := changes[key]; has {
			value = override
		}
		if value != "" {
			values.Set(key, value)
		}
	}
	set("screen", o.Screen)
	set("locale", o.Locale)
	set("page", fmt.Sprint(o.Page))
	set("scale", fmt.Sprint(o.Scale))
	set("grey", fmt.Sprint(o.Grey))
	set("toast", o.Toast)
	set("dialog", o.Dialog)
	set("confirm", o.Confirm)
	set("value", o.Value)
	set("chrome", o.Chrome)
	return "?" + values.Encode()
}

const fontStacks = `
.sans{font-family:"Helvetica Neue","Segoe UI","Noto Sans","DejaVu Sans","Arial Unicode MS",Arial,sans-serif}
.serif{font-family:"Noto Serif","Georgia","Times New Roman","DejaVu Serif",serif}
.mono{font-family:"SF Mono",Menlo,"DejaVu Sans Mono",Consolas,monospace}`

const pageStyle = `
:root{color-scheme:light}
*{box-sizing:border-box}
body{margin:0;background:#d9d9d9;color:#000;font:14px/1.4 -apple-system,"Segoe UI",sans-serif;display:flex;min-height:100vh}
aside{width:340px;flex:none;padding:16px;background:#fff;border-right:1px solid #000;overflow:auto;height:100vh;position:sticky;top:0}
main{flex:1;padding:24px;overflow:auto}
h1{font-size:16px;margin:0 0 12px}h2{font-size:12px;text-transform:uppercase;letter-spacing:.06em;margin:16px 0 6px}
aside a{color:#000}aside .on{font-weight:700;text-decoration:underline}
textarea{width:100%;height:180px;font:12px/1.3 Menlo,monospace}
button.apply{margin-top:6px;padding:6px 12px}
pre{white-space:pre-wrap;font:12px/1.3 Menlo,monospace;background:#f2f2f2;padding:8px;margin:6px 0;max-height:240px;overflow:auto}
.warn{color:#000;border-left:3px solid #000;padding-left:8px;margin:4px 0}
.frame{position:relative;background:#fff;border:2px solid #000;overflow:hidden}
.screen{position:absolute;left:0;top:0;width:1264px;height:1680px;background:#fff;transform-origin:0 0;color:#000}
.i{position:absolute;margin:0;overflow:hidden;white-space:pre;animation:none;transition:none}
.e{cursor:pointer}
.pager{display:flex;gap:12px;margin-bottom:12px;align-items:center}
.pager a{border:1px solid #000;padding:2px 12px;text-decoration:none}
` + fontStacks

const quantizeFilter = `<svg width="0" height="0" style="position:absolute"><filter id="q16" color-interpolation-filters="sRGB"><feColorMatrix type="saturate" values="0"/><feComponentTransfer>` +
	`<feFuncR type="discrete" tableValues="0 .067 .133 .2 .267 .333 .4 .467 .533 .6 .667 .733 .8 .867 .933 1"/>` +
	`<feFuncG type="discrete" tableValues="0 .067 .133 .2 .267 .333 .4 .467 .533 .6 .667 .733 .8 .867 .933 1"/>` +
	`<feFuncB type="discrete" tableValues="0 .067 .133 .2 .267 .333 .4 .467 .533 .6 .667 .733 .8 .867 .933 1"/>` +
	`</feComponentTransfer></filter></svg>`

const pageScript = `
const log=document.getElementById('log');
function closeOverlay(){const p=new URLSearchParams(location.search);['toast','dialog','confirm'].forEach(k=>p.delete(k));location.search=p.toString();}
document.addEventListener('keydown',e=>{
  if(e.key==='ArrowRight'){const n=document.getElementById('next');if(n)n.click();}
  if(e.key==='ArrowLeft'){const n=document.getElementById('prev');if(n)n.click();}
});
document.querySelectorAll('.screen .e').forEach(el=>el.addEventListener('click',async()=>{
  if(el.dataset.close==='1' && !el.dataset.event){closeOverlay();return;}
  if(el.dataset.confirm){const p=new URLSearchParams(location.search);p.set('confirm',el.dataset.confirm);location.search=p.toString();return;}
  const r=await fetch('/api/event',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({line:el.dataset.event,action:el.dataset.action})});
  const j=await r.json();
  log.textContent=JSON.stringify(j,null,2)+'\n'+log.textContent;
  if(j.reload||el.dataset.close==='1'){closeOverlay();}
}));
document.getElementById('apply').addEventListener('click',async()=>{
  const r=await fetch('/api/state',{method:'PUT',headers:{'Content-Type':'application/json'},body:document.getElementById('state').value});
  if(r.ok){location.reload();}else{log.textContent=await r.text();}
});
`

// renderItemsHTML turns items into absolutely positioned elements.
func renderItemsHTML(items []Item) string {
	var out strings.Builder
	for _, item := range items {
		out.WriteString(itemHTML(item))
	}
	return out.String()
}

func itemHTML(item Item) string {
	style := fmt.Sprintf("left:%dpx;top:%dpx;width:%dpx;height:%dpx;", item.X, item.Y, item.W, item.H)
	if item.Fill != noFill {
		style += "background:" + greyCSS(item.Fill) + ";"
	}
	if item.Border > 0 {
		borderStyle := "solid"
		if item.Dashed {
			borderStyle = "dashed"
		}
		style += fmt.Sprintf("border:%dpx %s %s;", item.Border, borderStyle, greyCSS(item.Ink))
	}
	if item.Shape == "ellipse" {
		style += "border-radius:50%;"
	}
	classes := "i"
	attributes := ""
	if item.Shape == "text" {
		style += textStyleCSS(item)
		classes += " " + item.Family
	}
	if item.Event != "" || item.Confirm != "" || item.Close {
		classes += " e"
		attributes = fmt.Sprintf(` data-event='%s' data-action='%s' data-confirm='%s' data-close='%s' title='%s'`,
			html.EscapeString(item.Event), html.EscapeString(item.Action), html.EscapeString(item.Confirm), closeFlag(item.Close), html.EscapeString(item.Label))
	}
	return fmt.Sprintf(`<div class="%s" style="%s"%s>%s</div>`, classes, style, attributes, itemContentHTML(item))
}

// itemContentHTML is the text of a text item, or the picture of an image item as a data URI so
// the page needs no file server.
func itemContentHTML(item Item) string {
	if item.Shape != "image" {
		return html.EscapeString(strings.Join(item.Lines, "\n"))
	}
	var encoded bytes.Buffer
	if item.Picture == nil || png.Encode(&encoded, item.Picture) != nil {
		return ""
	}
	return `<img alt="" style="width:100%;height:100%;display:block" src="data:image/png;base64,` + base64.StdEncoding.EncodeToString(encoded.Bytes()) + `">`
}

func closeFlag(isClose bool) string {
	if isClose {
		return "1"
	}
	return ""
}

func textStyleCSS(item Item) string {
	style := fmt.Sprintf("font-size:%dpx;line-height:%dpx;color:%s;", item.Size, item.LineH, greyCSS(item.Ink))
	if item.Bold {
		style += "font-weight:700;"
	}
	if item.Underline {
		style += "text-decoration:underline;"
	}
	switch item.Align {
	case "center":
		style += "text-align:center;"
	case "end":
		style += "text-align:right;"
	}
	return style
}

func greyCSS(level int) string {
	return fmt.Sprintf("#%02x%02x%02x", level, level, level)
}

// sidebarData is what the sidebar needs besides the page.
type sidebarData struct {
	AppName    string
	ScreenIDs  []string
	Locales    []string
	StateJSON  string
	Warnings   []string
	ToastIDs   []string
	DialogIDs  []string
	ExecActive bool
}

func renderPage(view viewOptions, page Page, side sidebarData) string {
	var out strings.Builder
	scale := float64(view.Scale) / 100
	frameStyle := fmt.Sprintf("width:%dpx;height:%dpx;", int(screenWidth*scale)+4, int(screenHeight*scale)+4)
	filter := ""
	if view.Grey == 16 {
		filter = "filter:url(#q16);"
	}
	fmt.Fprintf(&out, "<!doctype html><html lang=\"%s\"><head><meta charset=\"utf-8\"><title>%s - ndpreview</title><style>%s</style></head><body>",
		html.EscapeString(view.Locale), html.EscapeString(side.AppName), pageStyle)
	out.WriteString(quantizeFilter)
	out.WriteString(sidebarHTML(view, page, side))
	out.WriteString("<main>" + pagerHTML(view, page))
	fmt.Fprintf(&out, `<div class="frame" id="frame" style="%s%s"><div class="screen" id="screen" data-screen="%s" data-page="%d" data-pages="%d" style="transform:scale(%g)">%s</div></div>`,
		frameStyle, filter, html.EscapeString(page.ScreenID), page.Index+1, page.Count, scale, renderItemsHTML(page.Items))
	fmt.Fprintf(&out, "</main><script>%s</script></body></html>", pageScript)
	return out.String()
}

func pagerHTML(view viewOptions, page Page) string {
	var out strings.Builder
	out.WriteString(`<div class="pager">`)
	if page.Index > 0 {
		fmt.Fprintf(&out, `<a id="prev" href="%s" title="Left arrow key">&larr;</a>`, view.query(map[string]string{"page": fmt.Sprint(page.Index - 1)}))
	}
	fmt.Fprintf(&out, `<span>%d / %d</span>`, page.Index+1, page.Count)
	if page.Index < page.Count-1 {
		fmt.Fprintf(&out, `<a id="next" href="%s" title="Right arrow key">&rarr;</a>`, view.query(map[string]string{"page": fmt.Sprint(page.Index + 1)}))
	}
	out.WriteString(`</div>`)
	return out.String()
}

func sidebarHTML(view viewOptions, page Page, side sidebarData) string {
	var out strings.Builder
	out.WriteString("<aside><h1>" + html.EscapeString(side.AppName) + "</h1>")
	out.WriteString("<h2>Screen</h2>" + linkList(view, "screen", side.ScreenIDs, view.Screen, map[string]string{"page": "0", "toast": "", "dialog": "", "confirm": ""}))
	out.WriteString("<h2>Language</h2>" + linkList(view, "locale", side.Locales, view.Locale, nil))
	out.WriteString("<h2>Size</h2>" + linkList(view, "scale", []string{"50", "100"}, fmt.Sprint(view.Scale), nil))
	out.WriteString("<h2>Grey levels</h2>" + linkList(view, "grey", []string{"0", "16"}, fmt.Sprint(view.Grey), nil))
	if len(side.ToastIDs)+len(side.DialogIDs) > 0 {
		out.WriteString("<h2>Overlays</h2>" + linkList(view, "toast", side.ToastIDs, view.Toast, nil) + linkList(view, "dialog", side.DialogIDs, view.Dialog, nil))
	}
	for _, warning := range side.Warnings {
		out.WriteString(`<p class="warn">` + html.EscapeString(warning) + "</p>")
	}
	out.WriteString(`<h2>State (vars)</h2><textarea id="state" spellcheck="false">` + html.EscapeString(side.StateJSON) + `</textarea><br><button class="apply" id="apply">Apply</button>`)
	out.WriteString("<h2>Events to the backend</h2>")
	if side.ExecActive {
		out.WriteString("<p>--exec is on: actions run the app script in a temporary folder.</p>")
	}
	out.WriteString(`<pre id="log">Tap a button or row.</pre></aside>`)
	return out.String()
}

func linkList(view viewOptions, key string, values []string, current string, extra map[string]string) string {
	var parts []string
	for _, value := range values {
		changes := map[string]string{key: value}
		for name, change := range extra {
			changes[name] = change
		}
		if key == "toast" || key == "dialog" {
			changes["toast"], changes["dialog"] = "", ""
			changes[key] = value
		}
		class := ""
		if value == current {
			class = ` class="on"`
		}
		parts = append(parts, fmt.Sprintf(`<a%s href="%s">%s</a>`, class, view.query(changes), html.EscapeString(value)))
	}
	return strings.Join(parts, " · ") + " "
}
