package drawer

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"koboapps/internal/uikit"
)

const (
	maxGalleryTiles   = 500
	maxPendingTiles   = 100
	defaultGridColumn = 4
	defaultGridRows   = 3
	pictureSuffix     = ".png"
)

var tileIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

// HeaderIconAction is an icon button at the right of the header.
type HeaderIconAction struct {
	Icon   string `json:"icon"`
	Label  string `json:"label"`
	Push   string `json:"push,omitempty"`
	Action string `json:"action,omitempty"`
}

// GalleryTile is one picture of the grid. Thumb is "" when the file is missing or unsafe.
type GalleryTile struct {
	ID      string `json:"id"`
	Thumb   string `json:"thumb"`
	Label   string `json:"label"`
	Info    string `json:"info,omitempty"`
	Current bool   `json:"current,omitempty"`
}

// ViewerTile is one picture of the viewer. Image is "" when the file is missing or unsafe.
type ViewerTile struct {
	ID      string `json:"id"`
	Image   string `json:"image"`
	Label   string `json:"label"`
	Current bool   `json:"current,omitempty"`
}

// TileTarget is what a tile tap opens or runs; exactly one field is set.
type TileTarget struct {
	Push   string `json:"push,omitempty"`
	Action string `json:"action,omitempty"`
}

// TileMenuItem is one entry of the long-press menu: exactly one of Action, Push or Info.
type TileMenuItem struct {
	Label   string        `json:"label"`
	Action  string        `json:"action,omitempty"`
	Push    string        `json:"push,omitempty"`
	Info    bool          `json:"info,omitempty"`
	Confirm *ScreenDialog `json:"confirm,omitempty"`
}

// GalleryEmpty is the gallery's own empty state.
type GalleryEmpty struct {
	Icon   string `json:"icon,omitempty"`
	Title  string `json:"title"`
	Text   string `json:"text,omitempty"`
	Button string `json:"button,omitempty"`
	Action string `json:"action,omitempty"`
}

// ViewerButton is the viewer's primary or secondary bottom-bar button.
type ViewerButton struct {
	Label        string        `json:"label"`
	LabelCurrent string        `json:"label_current,omitempty"`
	Action       string        `json:"action"`
	Confirm      *ScreenDialog `json:"confirm,omitempty"`
}

func (s screenContext) headerIconActions(declared []map[string]any) []HeaderIconAction {
	var actions []HeaderIconAction
	for _, fields := range declared {
		actions = append(actions, HeaderIconAction{Icon: stringProp(fields, "icon"), Label: s.text(fields["label"]),
			Push: stringProp(fields, "push"), Action: stringProp(fields, "action")})
	}
	return actions
}

func (s screenContext) galleryComponent(c uikit.Component) ScreenComponent {
	pending := s.pendingCount(c.Prop("pending"))
	resolved := ScreenComponent{
		Bind: bindName(c), Columns: intProp(c.Props, "columns", defaultGridColumn), Rows: intProp(c.Props, "rows", defaultGridRows),
		Tiles: s.galleryTiles(c), Menu: s.tileMenu(c.Prop("on_long_press")), Pending: &pending,
	}
	if tap, isObject := c.Prop("on_tap").(map[string]any); isObject {
		resolved.Tap = &TileTarget{Push: stringProp(tap, "push"), Action: stringProp(tap, "action")}
	}
	if empty, isObject := c.Prop("empty").(map[string]any); isObject {
		resolved.Empty = &GalleryEmpty{Icon: stringProp(empty, "icon"), Title: s.text(empty["title"]),
			Text: s.text(empty["text"]), Button: s.text(empty["button"]), Action: stringProp(empty, "action")}
	}
	return resolved
}

func (s screenContext) viewerComponent(c uikit.Component) ScreenComponent {
	tiles := s.viewerTiles(c)
	resolved := ScreenComponent{
		Bind: bindName(c), Tiles: tiles, Value: s.viewerValue(tiles), CurrentStatus: s.text(c.Prop("current_status")),
	}
	if primary, isObject := c.Prop("primary").(map[string]any); isObject {
		resolved.Primary = s.viewerButton(primary)
	}
	if secondary, isObject := c.Prop("secondary").(map[string]any); isObject {
		resolved.Secondary = s.viewerButton(secondary)
	}
	return resolved
}

func (s screenContext) viewerButton(fields map[string]any) *ViewerButton {
	button := &ViewerButton{Label: s.text(fields["label"]), LabelCurrent: s.text(fields["label_current"]),
		Action: stringProp(fields, "action")}
	if confirm, hasConfirm := fields["confirm"].(map[string]any); hasConfirm {
		question := s.dialog(confirm)
		button.Confirm = &question
	}
	return button
}

// viewerValue is the screen value when it names a tile, else the first tile, else "".
func (s screenContext) viewerValue(tiles []ViewerTile) string {
	if s.value != nil {
		for _, tile := range tiles {
			if tile.ID == *s.value {
				return tile.ID
			}
		}
	}
	if len(tiles) > 0 {
		return tiles[0].ID
	}
	return ""
}

func (s screenContext) tileMenu(raw any) []TileMenuItem {
	declared, _ := raw.([]any)
	var menu []TileMenuItem
	for _, entry := range declared {
		fields, isObject := entry.(map[string]any)
		if !isObject {
			continue
		}
		item := TileMenuItem{Label: s.text(fields["label"]), Action: stringProp(fields, "action"),
			Push: stringProp(fields, "push"), Info: fields["info"] == true}
		if confirm, hasConfirm := fields["confirm"].(map[string]any); hasConfirm {
			question := s.dialog(confirm)
			item.Confirm = &question
		}
		menu = append(menu, item)
	}
	return menu
}

// pendingCount reads the number of placeholder tiles from a literal or a binding.
func (s screenContext) pendingCount(raw any) int {
	count, err := strconv.Atoi(strings.TrimSpace(s.text(raw)))
	if err != nil {
		return 0
	}
	return max(0, min(count, maxPendingTiles))
}

// pictureItems returns the maps of the bound array that have a valid id, at most maxGalleryTiles.
func (s screenContext) pictureItems(c uikit.Component) []map[string]any {
	values, _ := uikit.Lookup(s.vars, bindName(c))
	raw, _ := values.([]any)
	var items []map[string]any
	for _, entry := range raw {
		fields, isObject := entry.(map[string]any)
		if id, _ := fields["id"].(string); isObject && tileIDPattern.MatchString(id) && len(items) < maxGalleryTiles {
			items = append(items, fields)
		}
	}
	return items
}

func (s screenContext) galleryTiles(c uikit.Component) []GalleryTile {
	tiles := []GalleryTile{}
	for _, fields := range s.pictureItems(c) {
		tiles = append(tiles, GalleryTile{ID: stringProp(fields, "id"), Thumb: s.safePicturePath(stringProp(fields, "thumb")),
			Label: stringProp(fields, "label"), Info: stringProp(fields, "info"), Current: fields["badge"] == "current"})
	}
	return tiles
}

func (s screenContext) viewerTiles(c uikit.Component) []ViewerTile {
	tiles := []ViewerTile{}
	for _, fields := range s.pictureItems(c) {
		tiles = append(tiles, ViewerTile{ID: stringProp(fields, "id"), Image: s.safePicturePath(stringProp(fields, "image")),
			Label: stringProp(fields, "label"), Current: fields["badge"] == "current"})
	}
	return tiles
}

// safePicturePath returns path when it is an absolute, already clean .png path below
// <appDir>/data/, else "". The check is lexical on purpose: the mod reads the file itself.
func (s screenContext) safePicturePath(path string) string {
	dataDir, err := filepath.Abs(filepath.Join(s.appDir, "data"))
	if err != nil || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return ""
	}
	if !strings.HasSuffix(path, pictureSuffix) || !strings.HasPrefix(path, dataDir+string(filepath.Separator)) {
		return ""
	}
	return path
}

func bindName(c uikit.Component) string {
	reference, _ := c.Main().(string)
	return strings.TrimPrefix(reference, "$")
}

func intProp(object map[string]any, key string, fallback int) int {
	if declared, isNumber := object[key].(float64); isNumber {
		return int(declared)
	}
	return fallback
}
