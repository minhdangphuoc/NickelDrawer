package uikit

import "strings"

// The view flow of an app is derived from the screens' push links (header action, header icon actions, gallery tap and long-press menu); there is no
// separate flow field. Rules (docs/spec/ui-json.md "Navigation"): the root screen is "main"
// (and "start" names it), every screen is reachable from it, nothing is pushed deeper than
// MaxPushDepth below it, no push forms a cycle (a screen pushing itself included), and a push
// names an existing screen (ui.unknown_screen, checked with the other references).

// RootScreenID is the screen every app opens on.
const RootScreenID = "main"

// MaxPushDepth is how many screens may be stacked above main (main → sub → detail). The
// drawer mod enforces the same limit at run time (mod/drawer/navigation.h).
const MaxPushDepth = 2

const (
	CodeFlowNoMain       = "ui.flow_no_main"
	CodeFlowStartNotMain = "ui.flow_start_not_main"
	CodeFlowCycle        = "ui.flow_cycle"
	CodeFlowTooDeep      = "ui.flow_too_deep"
	CodeFlowUnreachable  = "ui.flow_unreachable"
)

// Flow is the screen graph: for each screen, the screens its header pushes.
type Flow struct {
	Root  string
	Edges map[string][]string
}

// flowOf reads the push links of raw screen objects; links to unknown screens are left out
// (they are reported as ui.unknown_screen). A screen can have several links.
func flowOf(screens map[string]any) Flow {
	flow := Flow{Root: RootScreenID, Edges: map[string][]string{}}
	for _, id := range sortedKeys(screens) {
		flow.Edges[id] = nil
		for _, target := range pushTargets(screens[id]) {
			if _, exists := screens[target]; exists && !contains(flow.Edges[id], target) {
				flow.Edges[id] = append(flow.Edges[id], target)
			}
		}
	}
	return flow
}

// pushTargets lists the screens a screen can open: its header action, header icon actions,
// gallery tap and gallery long-press menu.
func pushTargets(screen any) []string {
	object, _ := screen.(map[string]any)
	targets := []string{pushOf(object["action"])}
	icons, _ := object["actions"].([]any)
	for _, icon := range icons {
		targets = append(targets, pushOf(icon))
	}
	for _, component := range componentObjects(object["body"], object["footer"]) {
		targets = append(targets, componentPushTargets(component)...)
	}
	return targets
}

func componentObjects(lists ...any) []map[string]any {
	var objects []map[string]any
	for _, list := range lists {
		items, _ := list.([]any)
		for _, item := range items {
			if object, isObject := item.(map[string]any); isObject {
				objects = append(objects, object)
			}
		}
	}
	return objects
}

func componentPushTargets(component map[string]any) []string {
	if _, isRow := component["row"]; isRow {
		return []string{pushOf(component)}
	}
	if _, isGallery := component["gallery"]; !isGallery {
		return nil
	}
	targets := []string{pushOf(component["on_tap"])}
	menu, _ := component["on_long_press"].([]any)
	for _, item := range menu {
		targets = append(targets, pushOf(item))
	}
	return targets
}

func pushOf(raw any) string {
	object, _ := raw.(map[string]any)
	target, _ := object["push"].(string)
	return target
}

// Depths returns how far below the root each reachable screen can be pushed (the longest
// chain, which is what the back stack holds); unreachable screens are absent.
func (f Flow) Depths() map[string]int {
	depths := map[string]int{}
	var walk func(id string, depth int, onPath map[string]bool)
	walk = func(id string, depth int, onPath map[string]bool) {
		if onPath[id] {
			return // a cycle; reported by Cycles
		}
		if known, seen := depths[id]; seen && known >= depth {
			return
		}
		depths[id] = depth
		onPath[id] = true
		for _, next := range f.Edges[id] {
			walk(next, depth+1, onPath)
		}
		delete(onPath, id)
	}
	if _, exists := f.Edges[f.Root]; exists {
		walk(f.Root, 0, map[string]bool{})
	}
	return depths
}

// Cycles returns each push cycle once, as the screen ids in push order starting at the
// smallest id (a self-push is a cycle of one).
func (f Flow) Cycles() [][]string {
	var cycles [][]string
	for _, start := range sortedKeys(f.Edges) {
		cycles = append(cycles, f.cyclesThrough(start)...)
	}
	return cycles
}

// cyclesThrough finds the cycles whose smallest screen id is start, so each cycle is found
// exactly once whatever the number of edges per screen.
func (f Flow) cyclesThrough(start string) [][]string {
	var found [][]string
	var walk func(current string, path []string)
	walk = func(current string, path []string) {
		for _, next := range f.Edges[current] {
			switch {
			case next == start:
				found = append(found, append([]string{}, path...))
			case next > start && !contains(path, next):
				walk(next, append(path, next))
			}
		}
	}
	walk(start, []string{start})
	return found
}

// checkFlow applies the navigation rules to the merged screens.
func (v *validator) checkFlow(root map[string]any, screens map[string]any) {
	if len(screens) == 0 {
		return // reported as a missing screen
	}
	flow := flowOf(screens)
	if _, hasMain := screens[RootScreenID]; !hasMain {
		v.fail(CodeFlowNoMain, "screens", "needs a screen with the id %q: the app opens on it; screens: %s", RootScreenID, strings.Join(sortedKeys(screens), ", "))
		return
	}
	if start, isString := root["start"].(string); isString && start != RootScreenID {
		v.fail(CodeFlowStartNotMain, "start", "must be %q (the app opens on main; other screens are opened from it with a header \"push\")", RootScreenID)
	}
	for _, cycle := range flow.Cycles() {
		v.fail(CodeFlowCycle, "screens."+cycle[0], "screens push each other in a circle (%s → %s); back would never end: remove one push", strings.Join(cycle, " → "), cycle[0])
	}
	depths := flow.Depths()
	for _, id := range sortedKeys(screens) {
		depth, isReachable := depths[id]
		switch {
		case !isReachable:
			v.fail(CodeFlowUnreachable, "screens."+id, "no screen opens %q: push it from main or from a screen main opens, or delete it", id)
		case depth > MaxPushDepth:
			v.fail(CodeFlowTooDeep, "screens."+id, "is %d screens below main; at most %d are allowed (main → screen → detail)", depth, MaxPushDepth)
		}
	}
}
