package main

// Device locations of the apps and the host; flags and tests override them.
const (
	deviceAppsDir     = "/mnt/onboard/.adds/apps"
	deviceKoboappsDir = "/mnt/onboard/.adds/koboapps"
)

func defaultAppsDir() string     { return deviceAppsDir }
func defaultKoboappsDir() string { return deviceKoboappsDir }
