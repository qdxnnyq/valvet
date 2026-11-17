package main

import (
	"embed"
	"io/fs"
	"net/http"
	"strconv"
)

//go:embed pack
var packFiles embed.FS

func packHandler() http.Handler {
	sub, err := fs.Sub(packFiles, "pack")
	if err != nil {
		return http.NotFoundHandler()
	}
	return http.StripPrefix("/pack/", http.FileServer(http.FS(sub)))
}

func localRankIcons() map[int]string {
	icons := map[int]string{}
	for tier := 3; tier <= 27; tier++ {
		icons[tier] = packRankIcon(tier)
	}
	return icons
}

func packRankIcon(tier int) string {
	if tier < 3 || tier > 27 {
		return ""
	}
	return "/pack/rank/" + strconv.Itoa(tier) + ".png"
}
