package main

import "koboapps/internal/nmconfig"

// Device locations; flags override them so tests and a Mac can use temp folders.
const (
	defaultMenuFile   = "/mnt/onboard/.adds/nm/koboapps"
	defaultErrorsFile = "/mnt/onboard/.adds/koboapps/data/errors.txt"
)

func defaultAppsDir() string     { return nmconfig.DefaultPaths().AppsRoot }
func defaultKoboappsDir() string { return nmconfig.DefaultPaths().KoboappsRoot }
