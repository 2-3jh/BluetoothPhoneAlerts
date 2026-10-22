//go:build windows

package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWebDeviceUsesFrontendFieldNames(t *testing.T) {
	data, err := json.Marshal(webDevice{Name: "OPPO Reno9 5G", Address: "7486698C10BA"})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"name":"OPPO Reno9 5G","address":"7486698C10BA"}` {
		t.Fatalf("unexpected device JSON: %s", data)
	}
}

func TestBatteryDoesNotOverwriteConnectionStatus(t *testing.T) {
	webStateMu.Lock()
	oldStatus, oldBattery := webCurrentStatus, webBattery
	webStateMu.Unlock()
	t.Cleanup(func() {
		webStateMu.Lock()
		webCurrentStatus, webBattery = oldStatus, oldBattery
		webStateMu.Unlock()
	})
	setWebStatus("已连接：OPPO Reno9 5G")
	setWebBattery("电量 99% · 正在充电")
	state := currentWebState()
	if state.Status != "已连接：OPPO Reno9 5G" || state.Battery != "电量 99% · 正在充电" {
		t.Fatalf("unexpected status and battery: %+v", state)
	}
}

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
