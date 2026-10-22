//go:build windows

package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	webview2 "github.com/jchv/go-webview2"
)

//go:embed ui/index.html
var webHTML string

//go:embed ui/app.css
var webCSS string

//go:embed ui/app.js
var webJS string

type webDevice struct{ Name, Address string }
type webNotice struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}
type webState struct {
	Devices         []webDevice `json:"devices"`
	SelectedAddress string      `json:"selectedAddress"`
	Status          string      `json:"status"`
	Logs            []string    `json:"logs"`
	Notices         []webNotice `json:"notices"`
}

var (
	webMode          = true
	webStateMu       sync.Mutex
	webDeviceMu      sync.RWMutex
	webCurrentStatus = "未连接"
	webLogs          []string
	webNotices       []webNotice
	webView          webview2.WebView
	webWindow        uintptr
	webOriginalProc  uintptr
	webProcCallback  uintptr
)

var (
	setWindowLongPtr = user32.NewProc("SetWindowLongPtrW")
	callWindowProc   = user32.NewProc("CallWindowProcW")
)

func webWindowProcedure(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == WM_CLOSE && !closing {
		call(showWindow, hwnd, SW_HIDE)
		return 0
	}
	return call(callWindowProc, webOriginalProc, hwnd, uintptr(msg), wParam, lParam)
}

func appendWebLog(line string) {
	webStateMu.Lock()
	defer webStateMu.Unlock()
	webLogs = append(webLogs, line)
	if len(webLogs) > 120 {
		webLogs = webLogs[len(webLogs)-120:]
	}
}

func clearWebLogs()              { webStateMu.Lock(); webLogs = nil; webStateMu.Unlock() }
func setWebStatus(status string) { webStateMu.Lock(); webCurrentStatus = status; webStateMu.Unlock() }

func processWebAlert(kind, title, body string) {
	if kind == "status" {
		setStatus(body)
		logUI(title + "：" + body)
		return
	}
	if title == "" {
		switch kind {
		case "sms":
			title = "手机收到新短信"
		case "call":
			title = "手机有来电"
		default:
			title = "蓝牙测试成功"
		}
	}
	if body == "" {
		switch kind {
		case "sms":
			body = "请打开手机查看短信"
		case "call":
			body = "请查看手机上的来电"
		default:
			body = "手机与电脑蓝牙连接正常"
		}
	}
	logUI(title + "：" + body)
	appendHistory(title, body)
	tray(NIM_MODIFY, title, body)
	webStateMu.Lock()
	webNotices = append(webNotices, webNotice{time.Now().UnixNano(), title, body})
	if len(webNotices) > 8 {
		webNotices = webNotices[len(webNotices)-8:]
	}
	webStateMu.Unlock()
}

func refreshWebDevices() error {
	devices, err := listPairedDevices()
	if err != nil {
		webDeviceMu.Lock()
		btDevices = nil
		webDeviceMu.Unlock()
		logUI(err.Error())
		setStatus("请先在 Windows 蓝牙设置中配对手机")
		return err
	}
	webDeviceMu.Lock()
	btDevices = devices
	webDeviceMu.Unlock()
	logUI(fmt.Sprintf("已发现 %d 个已配对蓝牙设备", len(devices)))
	return nil
}

func currentWebState() webState {
	webStateMu.Lock()
	state := webState{Status: webCurrentStatus, Logs: append([]string(nil), webLogs...), Notices: append([]webNotice(nil), webNotices...)}
	webStateMu.Unlock()
	saved := loadSavedAddress()
	if saved != 0 {
		state.SelectedAddress = fmt.Sprintf("%012X", saved)
	}
	webDeviceMu.RLock()
	for _, d := range btDevices {
		state.Devices = append(state.Devices, webDevice{d.name, fmt.Sprintf("%012X", d.address)})
	}
	webDeviceMu.RUnlock()
	return state
}

func connectWeb(address string) error {
	webDeviceMu.RLock()
	defer webDeviceMu.RUnlock()
	for _, d := range btDevices {
		if strings.EqualFold(fmt.Sprintf("%012X", d.address), address) {
			startConnectionTo(d)
			return nil
		}
	}
	return fmt.Errorf("所选设备已不在配对列表中，请刷新设备")
}

func renderWebPage() string {
	page := strings.Replace(webHTML, `<link rel="stylesheet" href="app.css" />`, "<style>"+webCSS+"</style>", 1)
	return strings.Replace(page, `<script src="app.js"></script>`, "<script>"+webJS+"</script>", 1)
}

func loadWebHistory() {
	historyMu.Lock()
	entries := readHistory()
	_ = saveHistory(entries)
	historyMu.Unlock()
	for _, e := range entries {
		appendWebLog(fmt.Sprintf("[%s] %s：%s", e.when.Format("01-02 15:04"), e.title, strings.ReplaceAll(e.body, "\r\n", " ")))
	}
}

func bindWebAPI(w webview2.WebView) {
	_ = w.Bind("appState", func() webState { return currentWebState() })
	_ = w.Bind("appRefresh", func() error { return refreshWebDevices() })
	_ = w.Bind("appConnect", func(address string) error { return connectWeb(address) })
	_ = w.Bind("appStop", func() error { stopConnection(); logUI("连接已手动停止"); return nil })
	_ = w.Bind("appTest", func() error { showAlertsNow(); return nil })
	_ = w.Bind("appClear", func() error { clearHistory(); return nil })
	_ = w.Bind("appExit", func() error {
		w.Dispatch(func() { closing = true; call(destroyWindow, webWindow) })
		return nil
	})
}

func main() {
	runtime.LockOSThread()
	if err := wsaStart(); err != nil {
		call(messageBox, 0, uintptr(unsafe.Pointer(u16(err.Error()))), uintptr(unsafe.Pointer(u16("蓝牙启动失败"))), MB_OK|MB_ICONWARNING)
		return
	}
	defer call(wsaCleanup)

	instance := call(getModuleHandle, 0)
	className := u16("BluetoothPhoneAlertsWebWindow")
	backgroundBrush = call(createSolidBrush, color(234, 242, 255))
	trayIcon = call(loadIcon, 0, 32516)
	wc := wndClassEx{size: uint32(unsafe.Sizeof(wndClassEx{})), style: 3, wndProc: syscall.NewCallback(windowProcedure), instance: instance,
		cursor: call(loadCursor, 0, 32512), icon: trayIcon, iconSmall: trayIcon, background: backgroundBrush, className: className}
	if call(registerClass, uintptr(unsafe.Pointer(&wc))) == 0 {
		return
	}
	window = call(createWindow, 0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(u16("BluetoothPhoneAlertsTray"))),
		WS_POPUP, 0, 0, 1, 1, 0, 0, instance, 0)
	if window == 0 {
		return
	}

	dataPath := filepath.Join(os.Getenv("LOCALAPPDATA"), "BluetoothPhoneAlerts", "WebView2")
	webView = webview2.NewWithOptions(webview2.WebViewOptions{DataPath: dataPath, AutoFocus: true,
		WindowOptions: webview2.WindowOptions{Title: "蓝牙提醒助手", Width: 980, Height: 760, Center: true}})
	if webView == nil {
		call(messageBox, window, uintptr(unsafe.Pointer(u16("电脑缺少 Microsoft Edge WebView2 Runtime，无法打开新版界面。"))), uintptr(unsafe.Pointer(u16("界面组件缺失"))), MB_OK|MB_ICONWARNING)
		call(destroyWindow, window)
		return
	}
	defer webView.Destroy()
	webWindow = uintptr(webView.Window())
	webProcCallback = syscall.NewCallback(webWindowProcedure)
	webOriginalProc = call(setWindowLongPtr, webWindow, ^uintptr(3), webProcCallback)
	cornerPreference, backdrop := uint32(2), uint32(2)
	call(dwmSetAttribute, webWindow, 33, uintptr(unsafe.Pointer(&cornerPreference)), 4)
	call(dwmSetAttribute, webWindow, 38, uintptr(unsafe.Pointer(&backdrop)), 4)
	bindWebAPI(webView)
	webView.SetHtml(renderWebPage())

	loadWebHistory()
	tray(NIM_ADD, "", "")
	_ = refreshWebDevices()
	logUI("新版 Web 界面已启动。关闭窗口后仍会在托盘运行。")
	skipAutoConnect := false
	for _, arg := range os.Args[1:] {
		if strings.EqualFold(arg, "--no-auto-connect") {
			skipAutoConnect = true
		}
	}
	if saved := loadSavedAddress(); !skipAutoConnect && saved != 0 {
		webDeviceMu.RLock()
		for _, d := range btDevices {
			if d.address == saved {
				startConnectionTo(d)
				break
			}
		}
		webDeviceMu.RUnlock()
	}
	webView.Run()
	stopConnection()
	tray(NIM_DELETE, "", "")
	call(destroyWindow, window)
}
