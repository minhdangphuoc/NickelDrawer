package library

import (
	"strconv"
	"time"

	"koboapps/internal/i18n"
)

const (
	groupsPerPage  = 4
	lastScanLayout = "02/01 15:04"
)

// State builds the variable bindings for ui.json (the drawer's `state` action prints them as
// one JSON object). It only reads small files, so it fits the 1 s budget.
func (service *Service) State(messages *i18n.Bundle) map[string]any {
	summary, scanned, _ := LoadSummary(service.Paths)
	running := IsRunning(service.Paths)
	progress, hasProgress := ReadProgress(service.Paths)
	page, pages := service.page(len(summary.Duplicates))
	selected := service.Selected()
	state := map[string]any{
		"books":         strconv.Itoa(summary.Counts.Files),
		"wrong":         strconv.Itoa(summary.Counts.WrongLanguage),
		"not_nfc":       strconv.Itoa(summary.Counts.NotNFC),
		"groups":        strconv.Itoa(summary.Counts.DuplicateGroups),
		"last_scan":     lastScanText(messages, summary.ScannedAt, scanned),
		"running":       running,
		"status":        "",
		"phase":         messages.Message("msg.phase_idle", nil),
		"done":          strconv.Itoa(summary.Counts.Files),
		"total":         strconv.Itoa(summary.Counts.Files),
		"detail":        "",
		"can_fix":       scanned && !running && summary.Counts.WrongLanguage > 0,
		"can_undo":      !running && service.HasBackups(),
		"has_pages":     pages > 1,
		"has_selection": selected != "" && !running,
		"selected":      selectedText(messages, selected),
		"duplicates":    duplicateRows(messages, summary.Duplicates, page),
	}
	if !scanned {
		state["phase"] = messages.Message("msg.phase_never", nil)
	}
	if running {
		applyProgress(state, messages, progress, hasProgress)
	}
	return state
}

func applyProgress(state map[string]any, messages *i18n.Bundle, progress Progress, hasProgress bool) {
	state["status"] = messages.Message("msg.status_running", nil)
	state["phase"] = messages.Message("msg.phase_working", nil)
	if !hasProgress {
		state["done"], state["total"] = "0", "0"
		return
	}
	key := "msg.phase_scan"
	if progress.Phase == phaseFix {
		key = "msg.phase_fix"
	}
	state["phase"] = messages.Message(key, nil)
	state["done"], state["total"] = strconv.Itoa(progress.Done), strconv.Itoa(progress.Total)
	state["detail"] = progress.Detail
}

func lastScanText(messages *i18n.Bundle, scannedAt time.Time, scanned bool) string {
	if !scanned {
		return messages.Message("msg.never", nil)
	}
	return scannedAt.Local().Format(lastScanLayout)
}

func selectedText(messages *i18n.Bundle, selected string) string {
	if selected == "" {
		return messages.Message("msg.nothing_selected", nil)
	}
	return messages.Message("msg.selected", i18n.Params("path", selected))
}

// page returns the current page (clamped) and the number of pages; the page number is kept in
// data/page.txt by NextPage.
func (service *Service) page(groups int) (page, pages int) {
	pages = (groups + groupsPerPage - 1) / groupsPerPage
	if pages == 0 {
		pages = 1
	}
	page = service.readPage() % pages
	return page, pages
}

func (service *Service) readPage() int {
	content, err := readSmallFile(service.Paths.Page())
	if err != nil {
		return 0
	}
	page, err := strconv.Atoi(content)
	if err != nil || page < 0 {
		return 0
	}
	return page
}

// NextPage moves to the next page of duplicate groups, wrapping to the first.
func (service *Service) NextPage() error {
	summary, _, err := LoadSummary(service.Paths)
	if err != nil {
		return err
	}
	page, pages := service.page(len(summary.Duplicates))
	return writeFileAtomic(service.Paths.Page(), []byte(strconv.Itoa((page+1)%pages)))
}

// duplicateRows lists the files of one page of groups as [path, "Group 2 · identical · 1.2 MB"].
func duplicateRows(messages *i18n.Bundle, groups []DuplicateGroup, page int) [][]string {
	rows := [][]string{}
	first := page * groupsPerPage
	for index := first; index < len(groups) && index < first+groupsPerPage; index++ {
		group := groups[index]
		kindKey := "msg.kind_probable"
		if group.Kind == KindIdentical {
			kindKey = "msg.kind_identical"
		}
		for _, path := range group.Paths {
			detail := messages.Message("msg.row_detail", i18n.Params(
				"group", strconv.Itoa(group.ID), "kind", messages.Message(kindKey, nil), "size", fileSize(path)))
			rows = append(rows, []string{path, detail})
		}
	}
	return rows
}
