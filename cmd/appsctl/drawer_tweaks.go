package main

import (
	"errors"
	"io"

	"koboapps/internal/tweaks"
)

func (session drawerSession) tweaksConfig() tweaks.Config {
	config := tweaks.DefaultConfig(session.bundle)
	config.ConfPath = session.options.confPath
	return config
}

func drawerTweaks(session drawerSession) (any, error) {
	if len(session.positions) != 0 {
		return nil, drawerUsageError()
	}
	return session.tweaksConfig().List(), nil
}

func drawerTweak(session drawerSession) (any, error) {
	if len(session.positions) != 1 {
		return nil, drawerUsageError()
	}
	return session.tweaksConfig().Apply(session.positions[0])
}

// writeTweakFailure prints the tweak error document; it reports false when err is not one.
func writeTweakFailure(stdout io.Writer, err error) bool {
	var tweakErr *tweaks.Error
	if !errors.As(err, &tweakErr) {
		return false
	}
	_ = writeDrawerJSON(stdout, tweaks.Failure{V: tweaks.ProtocolVersion, Error: tweakErr.Message})
	return true
}
