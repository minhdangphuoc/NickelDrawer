// Package nmconfig knows where the NickelMenu-era files live on the Kobo and cleans up what
// older versions wrote there. NickelDrawer no longer generates NickelMenu items.
package nmconfig

// Paths are the device locations of the apps and the host.
type Paths struct {
	AppsRoot     string // folder holding one subfolder per app
	KoboappsRoot string // folder holding bin/appsctl
}

// DefaultPaths returns the locations on the Kobo.
func DefaultPaths() Paths {
	return Paths{
		AppsRoot:     "/mnt/onboard/.adds/apps",
		KoboappsRoot: "/mnt/onboard/.adds/koboapps",
	}
}
