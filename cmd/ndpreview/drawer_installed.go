package main

import (
	"fmt"
	"path/filepath"
	"slices"

	"koboapps/internal/i18n"
	"koboapps/internal/manifest"
)

// installedDrawerItems are the apps of appsDir the drawer would list (hidden ones left out), with
// their own names in the locale and their own icons, in the sample order. Sample statuses fill the
// second line so the picture shows live status; they are the same strings as the sample items.
// Used for documentation pictures, where invented app names would mislead.
func installedDrawerItems(appsDir, locale, fileName string) ([]drawerItem, error) {
	scan, err := manifest.LoadAll(appsDir)
	if err != nil {
		return nil, err
	}
	var apps []manifest.App
	for _, app := range scan.Apps {
		if !app.IsHidden() {
			apps = append(apps, app)
		}
	}
	if len(apps) == 0 {
		return nil, fmt.Errorf("no visible app in %s", appsDir)
	}
	slices.SortStableFunc(apps, func(a, b manifest.App) int { return sampleRank(a.ID) - sampleRank(b.ID) })
	chrome := chromeFor(locale)
	items := make([]drawerItem, 0, len(apps))
	for index, app := range apps {
		if bundle, err := i18n.LoadBundle(filepath.Join(app.Dir, manifest.LocalesDirName), locale); err == nil {
			app = app.Localize(bundle)
		}
		status := chrome.statuses[index%len(chrome.statuses)]
		items = append(items, drawerItem{
			name: app.Name, status: status, hasAction: true,
			hasUpdate: status != "" && index%len(chrome.statuses) == 3,
			iconPath:  filepath.Join(app.Dir, fileName),
		})
	}
	return items, nil
}

// sampleRank orders the bundled apps like the samples; other apps follow.
func sampleRank(id string) int {
	if rank := slices.Index(sampleIconOrder, id); rank >= 0 {
		return rank
	}
	return len(sampleIconOrder)
}

// drawerItemsFor is the sample items, or with --installed the real apps of --apps.
func drawerItemsFor(flags drawerFlags, fileName string) ([]drawerItem, error) {
	if flags.installed {
		return installedDrawerItems(flags.appsDir, flags.locale, fileName)
	}
	return sampleDrawerItems(flags.appsDir, flags.locale, flags.count, fileName), nil
}
