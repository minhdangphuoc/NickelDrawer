package nmconfig

import "strings"

// legacyLocalSendTitles are the hand-written menu items that the localsend app replaces.
var legacyLocalSendTitles = []string{":Nhận sách LocalSend ", ":Dừng LocalSend "}

// RemoveLegacyLocalSendItems drops the two hand-written LocalSend menu_item blocks
// (each item line plus its indented chain_* lines) and leaves every other byte untouched.
func RemoveLegacyLocalSendItems(items string) string {
	lines := strings.SplitAfter(items, "\n")
	var kept []string
	isDropping := false
	for _, line := range lines {
		if isLegacyItemLine(line) {
			isDropping = true
			continue
		}
		if isDropping && isIndentedChainLine(line) {
			continue
		}
		isDropping = false
		kept = append(kept, line)
	}
	return strings.Join(kept, "")
}

func isLegacyItemLine(line string) bool {
	if !strings.HasPrefix(line, "menu_item ") {
		return false
	}
	for _, title := range legacyLocalSendTitles {
		if strings.Contains(line, title) {
			return true
		}
	}
	return false
}

func isIndentedChainLine(line string) bool {
	return (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) && strings.Contains(line, "chain_")
}
