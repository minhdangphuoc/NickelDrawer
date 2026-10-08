package nmconfig

import (
	"os"
	"strings"
	"testing"
)

func TestRemoveLegacyLocalSendItemsOnRealItemsFile(t *testing.T) {
	raw, err := os.ReadFile("testdata/legacy-items.txt")
	if err != nil {
		t.Fatal(err)
	}
	original := string(raw)
	got := RemoveLegacyLocalSendItems(original)

	for _, gone := range []string{"Nhận sách LocalSend", "Dừng LocalSend", ".adds/localsend/"} {
		if strings.Contains(got, gone) {
			t.Errorf("%q still present", gone)
		}
	}
	// Exactly the four legacy lines are gone; all others are byte-identical and in order.
	want := strings.Replace(original, `menu_item :main :Nhận sách LocalSend :nickel_wifi :autoconnect
  chain_success :cmd_output :9500 :/mnt/onboard/.adds/localsend/start.sh
menu_item :main :Dừng LocalSend :cmd_output :3000 :/mnt/onboard/.adds/localsend/stop.sh
  chain_success :nickel_misc :rescan_books
`, "", 1)
	if got != want {
		t.Fatalf("unexpected result:\n%s", got)
	}
	if !strings.Contains(got, "Library Reload") || !strings.Contains(got, "chain_success : cmd_spawn : quiet: mkdir") {
		t.Error("unrelated items were damaged")
	}
}

func TestRemoveLegacyLocalSendItemsIsIdempotent(t *testing.T) {
	once := RemoveLegacyLocalSendItems("menu_item :main :Dừng LocalSend :cmd_output :3000 :x\n  chain_success :y\nkeep\n")
	if once != "keep\n" || RemoveLegacyLocalSendItems(once) != once {
		t.Fatalf("got %q", once)
	}
}
