package main

import "testing"

func TestInstalledDrawerItemsListVisibleAppsWithOwnNames(t *testing.T) {
	items, err := installedDrawerItems("../../apps", "vi", "icon.png")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, item := range items {
		names = append(names, item.name)
		if item.name == "UI Kit" || item.name == "Bộ giao diện" {
			t.Errorf("hidden app %q is listed", item.name)
		}
	}
	if len(items) == 0 || items[0].name != "LocalSend" {
		t.Errorf("names = %v, want LocalSend first", names)
	}
	for _, name := range names {
		if name == "Ghi chú" || name == "Từ điển" {
			t.Errorf("sample name %q in installed items", name)
		}
	}
}

func TestInstalledDrawerItemsFailOnEmptyFolder(t *testing.T) {
	if _, err := installedDrawerItems(t.TempDir(), "en", "icon.png"); err == nil {
		t.Error("an apps folder without apps gave no error")
	}
}
