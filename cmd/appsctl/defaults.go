package main

import "koboapps/internal/nmconfig"

func defaultAppsDir() string     { return nmconfig.DefaultPaths().AppsRoot }
func defaultKoboappsDir() string { return nmconfig.DefaultPaths().KoboappsRoot }
