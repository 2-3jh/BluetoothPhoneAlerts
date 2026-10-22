//go:build windows

package main

import (
	"strings"
	"testing"
)

func TestRenderWebPageEmbedsFrontendAssets(t *testing.T) {
	page := renderWebPage()
	for _, fragment := range []string{"<style>", "<script>", "蓝牙提醒助手", "backdrop-filter", "appState"} {
		if !strings.Contains(page, fragment) {
			t.Fatalf("rendered page is missing %q", fragment)
		}
	}
	if strings.Contains(page, `href="app.css"`) || strings.Contains(page, `src="app.js"`) {
		t.Fatal("rendered page still contains external frontend asset references")
	}
}
