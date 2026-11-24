package main

import (
	"bytes"
	_ "embed"
)

//go:embed index.html
var indexHTML []byte

//go:embed app.css
var appCSS []byte

//go:embed app-live.css
var appLiveCSS []byte

//go:embed app-history.css
var appHistoryCSS []byte

//go:embed app-maps.css
var appMapsCSS []byte

//go:embed js/00-core.js
var jsCore []byte

//go:embed js/01-live.js
var jsLive []byte

//go:embed js/02-history.js
var jsHistory []byte

//go:embed js/03-mid.js
var jsMid []byte

//go:embed js/04-maps.js
var jsMaps []byte

//go:embed js/05-rest.js
var jsRest []byte

//go:embed js/06-rest.js
var jsTail []byte

// appJS is the live, history, and maps files joined with the shared core.
// One script keeps the shared names. A syntax error in one file is still checked on its own.
var appJS []byte

func init() {
	appJS = bytes.Join([][]byte{jsCore, jsLive, jsHistory, jsMid, jsMaps, jsRest, jsTail}, nil)
	appJS = bytes.ReplaceAll(appJS, []byte("__VALVET__"), []byte(appLabel))
	appJS = bytes.ReplaceAll(appJS, []byte("__MAPS__"), mapNamesJSON())
	indexHTML = bytes.ReplaceAll(indexHTML, []byte("__VALVET__"), []byte(appLabel))
}
