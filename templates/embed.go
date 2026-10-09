// Package templates holds the files `appsctl new` copies into a fresh app folder.
package templates

import "embed"

// App is the app skeleton. "all:" keeps dot files; .tmpl files are rendered by appsctl.
//
//go:embed all:app
var App embed.FS
