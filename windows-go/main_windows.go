//go:build windows

// BluetoothPhoneAlerts Windows client. Pure Windows APIs (no runtime dependencies).
// Windows 10/11 + Bluetooth Classic RFCOMM. Android hosts a paired secure RFCOMM server.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

const (
	AF_BTH          = 32
	SOCK_STREAM     = 1
	BTHPROTO_RFCOMM = 3
	INVALID_SOCKET  = ^uintptr(0)
	// WinSock functions such as connect/recv/send return a C int. On 64-bit
	// Windows, syscall.Proc.Call zero-extends that 32-bit -1 to 0xFFFFFFFF,
	// rather than returning an all-bits-set uintptr like an invalid SOCKET.
	SOCKET_ERROR = uintptr(0xffffffff)

	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	WS_CHILD            = 0x40000000
	WS_VSCROLL          = 0x00200000
	WS_BORDER           = 0x00800000
	WS_EX_CLIENTEDGE    = 0x00000200
	CBS_DROPDOWNLIST    = 3
	ES_MULTILINE        = 4
	ES_AUTOVSCROLL      = 0x40
	ES_READONLY         = 0x800
	WS_HSCROLL          = 0x00100000
	WM_DESTROY          = 0x0002
	WM_CLOSE            = 0x0010
	WM_TIMER            = 0x0113
	WM_COMMAND          = 0x0111
	WM_SETFONT          = 0x0030
	WM_CTLCOLORSTATIC   = 0x0138
	WM_APP              = 0x8000
	WM_LBUTTONDBLCLK    = 0x0203
	WM_RBUTTONUP        = 0x0205
	SW_HIDE             = 0
	SW_SHOW             = 5
	SW_RESTORE          = 9
	SW_SHOWNOACTIVATE   = 4
	WS_POPUP            = 0x80000000
	WS_CAPTION          = 0x00C00000
	WS_SYSMENU          = 0x00080000
	HWND_TOPMOST        = ^uintptr(0)
	SWP_NOACTIVATE      = 0x0010
	SM_CXSCREEN         = 0
	SM_CYSCREEN         = 1
	MB_OK               = 0
	MB_ICONWARNING      = 0x30
	MB_ICONINFORMATION  = 0x40
	MB_SETFOREGROUND    = 0x00010000
	MB_TOPMOST          = 0x00040000
	CB_ADDSTRING        = 0x0143
	CB_RESETCONTENT     = 0x014B
	CB_GETCURSEL        = 0x0147
	CB_SETCURSEL        = 0x014E
	EM_SETSEL           = 0x00B1
	EM_REPLACESEL       = 0x00C2
	NIM_ADD             = 0
	NIM_MODIFY          = 1
	NIM_DELETE          = 2
	NIF_MESSAGE         = 1
	NIF_ICON            = 2
	NIF_TIP             = 4
	NIF_INFO            = 0x10
	NIIF_INFO           = 1
	WM_TRAY             = WM_APP + 1
	WM_LOG              = WM_APP + 2
	WM_ALERT            = WM_APP + 3

	IDC_REFRESH    = 100
	IDC_CONNECT    = 101
	IDC_STOP       = 102
	IDC_DEVICE     = 103
	IDC_TEST       = 104
	IDC_EXIT       = 105
	IDC_CLEAR      = 106
	WM_USER_STATUS = WM_APP + 4
	POPUP_TIMER    = 1
)

type guid struct {
	data1 uint32
	data2 uint16
	data3 uint16
	data4 [8]byte
}

var serviceGUID = guid{0xbac20533, 0x10ac, 0x4c14, [8]byte{0xb8, 0xe9, 0x45, 0xa1, 0xb2, 0x9f, 0xeb, 0x17}}

type sockAddrBth struct {
	family  uint16
	address [8]byte
	classID [16]byte
	port    [4]byte
}

type bluetoothSearchParams struct {
	size              uint32
	authenticated     uint32
	remembered        uint32
	unknown           uint32
	connected         uint32
	inquiry           uint32
	timeoutMultiplier byte
	_                 [7]byte
	radio             uintptr
}
type systemTime struct{ year, month, dayOfWeek, day, hour, minute, second, milliseconds uint16 }
type bluetoothDeviceInfo struct {
	size          uint32
	_             uint32
	address       uint64
	classOfDevice uint32
	connected     uint32
	remembered    uint32
	authenticated uint32
	lastSeen      systemTime
	lastUsed      systemTime
	name          [248]uint16
}
type wndClassEx struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   uintptr
	icon       uintptr
	cursor     uintptr
	background uintptr
	menuName   *uint16
	className  *uint16
	iconSmall  uintptr
}
type point struct{ x, y int32 }
type winMsg struct {
	hwnd           uintptr
	message        uint32
	_              uint32
	wParam, lParam uintptr
	time           uint32
	pt             point
	lPrivate       uint32
}
type notifyIconData struct {
	size        uint32
	_           uint32
	hwnd        uintptr
	id          uint32
	flags       uint32
	callbackMsg uint32
	_           uint32
	icon        uintptr
	tip         [128]uint16
	state       uint32
	stateMask   uint32
	info        [256]uint16
	version     uint32
	infoTitle   [64]uint16
	infoFlags   uint32
	itemGuid    guid
	balloonIcon uintptr
}
type btDevice struct {
	address uint64
	name    string
}
type alertMessage struct{ kind, title, body string }

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	bthapi   = syscall.NewLazyDLL("BluetoothApis.dll")
	ws2      = syscall.NewLazyDLL("ws2_32.dll")

	registerClass    = user32.NewProc("RegisterClassExW")
	createWindow     = user32.NewProc("CreateWindowExW")
	defWindowProc    = user32.NewProc("DefWindowProcW")
	getMessage       = user32.NewProc("GetMessageW")
	translateMessage = user32.NewProc("TranslateMessage")
	dispatchMessage  = user32.NewProc("DispatchMessageW")
	postQuitMessage  = user32.NewProc("PostQuitMessage")
	postMessage      = user32.NewProc("PostMessageW")
	sendMessage      = user32.NewProc("SendMessageW")
	showWindow       = user32.NewProc("ShowWindow")
	updateWindow     = user32.NewProc("UpdateWindow")
	setWindowText    = user32.NewProc("SetWindowTextW")
	setWindowPos     = user32.NewProc("SetWindowPos")
	getSystemMetrics = user32.NewProc("GetSystemMetrics")
	setTimer         = user32.NewProc("SetTimer")
	killTimer        = user32.NewProc("KillTimer")
	messageBox       = user32.NewProc("MessageBoxW")
	destroyWindow    = user32.NewProc("DestroyWindow")
	loadIcon         = user32.NewProc("LoadIconW")
	loadCursor       = user32.NewProc("LoadCursorW")
	getModuleHandle  = kernel32.NewProc("GetModuleHandleW")
	getStockObject   = gdi32.NewProc("GetStockObject")
	createSolidBrush = gdi32.NewProc("CreateSolidBrush")
	createFont       = gdi32.NewProc("CreateFontW")
	setBkMode        = gdi32.NewProc("SetBkMode")
	setTextColor     = gdi32.NewProc("SetTextColor")
	notifyIcon       = shell32.NewProc("Shell_NotifyIconW")
	findFirstDevice  = bthapi.NewProc("BluetoothFindFirstDevice")
	findNextDevice   = bthapi.NewProc("BluetoothFindNextDevice")
	findDeviceClose  = bthapi.NewProc("BluetoothFindDeviceClose")
	wsaStartup       = ws2.NewProc("WSAStartup")
	wsaCleanup       = ws2.NewProc("WSACleanup")
	wsaSocket        = ws2.NewProc("socket")
	wsaConnect       = ws2.NewProc("connect")
	wsaRecv          = ws2.NewProc("recv")
	wsaSend          = ws2.NewProc("send")
	wsaClose         = ws2.NewProc("closesocket")
	wsaGetLastError  = ws2.NewProc("WSAGetLastError")

	window          uintptr
	combo           uintptr
	logEdit         uintptr
	statusLabel     uintptr
	btDevices       []btDevice
	uiMu            sync.Mutex
	uiMessages      []string
	uiAlerts        []alertMessage
	connMu          sync.Mutex
	activeSocket    uintptr = INVALID_SOCKET
	cancel          chan struct{}
	closing         bool
	trayIcon        uintptr
	backgroundBrush uintptr
	headerBrush     uintptr
	uiFont          uintptr
	titleFont       uintptr
	headerPanel     uintptr
	headerTitle     uintptr
	headerSubtitle  uintptr
	alertWindow     uintptr
	alertHeader     uintptr
	alertTitle      uintptr
	alertBody       uintptr
	seenMu          sync.Mutex
	seenIds         = make(map[string]bool)
	seenOrder       []string
)

func u16(s string) *uint16 { return syscall.StringToUTF16Ptr(s) }
func copy16(dest []uint16, value string) {
	s := utf16.Encode([]rune(value))
	if len(s) >= len(dest) {
		s = s[:len(dest)-1]
	}
	copy(dest, s)
}
func str16(a []uint16) string {
	n := 0
	for n < len(a) && a[n] != 0 {
		n++
	}
	return string(utf16.Decode(a[:n]))
}
func call(p *syscall.LazyProc, args ...uintptr) uintptr { r, _, _ := p.Call(args...); return r }
func syscallError() string                              { return fmt.Sprintf("WinSock 错误 %d", call(wsaGetLastError)) }

func listPairedDevices() ([]btDevice, error) {
	params := bluetoothSearchParams{size: uint32(unsafe.Sizeof(bluetoothSearchParams{})), authenticated: 1, remembered: 1}
	var info bluetoothDeviceInfo
	info.size = uint32(unsafe.Sizeof(info))
	handle := call(findFirstDevice, uintptr(unsafe.Pointer(&params)), uintptr(unsafe.Pointer(&info)))
	if handle == 0 {
		return nil, fmt.Errorf("没有发现已配对蓝牙设备（或电脑蓝牙未开启）")
	}
	defer call(findDeviceClose, handle)
	result := make([]btDevice, 0)
	for {
		if info.authenticated != 0 || info.remembered != 0 {
			result = append(result, btDevice{address: info.address, name: str16(info.name[:])})
		}
		info.size = uint32(unsafe.Sizeof(info))
		if call(findNextDevice, handle, uintptr(unsafe.Pointer(&info))) == 0 {
			break
		}
	}
	return result, nil
}

func loadSavedAddress() uint64 {
	data, err := os.ReadFile(savedPath())
	if err != nil {
		return 0
	}
	number, err := strconv.ParseUint(strings.TrimSpace(string(data)), 16, 64)
	if err != nil {
		return 0
	}
	return number
}
func savedPath() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "BluetoothPhoneAlerts", "paired-device-go.txt")
}
func saveAddress(value uint64) {
	path := savedPath()
	_ = os.MkdirAll(filepath.Dir(path), 0700)
	_ = os.WriteFile(path, []byte(fmt.Sprintf("%012X", value)), 0600)
}

func winClass(name string) *uint16 { return u16(name) }
func control(class, title string, style uint32, x, y, w, h int32, id uintptr) uintptr {
	hnd := call(createWindow, 0, uintptr(unsafe.Pointer(winClass(class))), uintptr(unsafe.Pointer(u16(title))),
		uintptr(style|WS_CHILD|WS_VISIBLE), uintptr(x), uintptr(y), uintptr(w), uintptr(h), window, id, 0, 0)
	font := uiFont
	if font == 0 {
		font = call(getStockObject, 17)
	}
	call(sendMessage, hnd, WM_SETFONT, font, 1)
	return hnd
}
func logUI(s string) {
	line := fmt.Sprintf("[%s] %s\r\n", time.Now().Format("15:04:05"), s)
	call(sendMessage, logEdit, EM_SETSEL, ^uintptr(0), ^uintptr(0))
	call(sendMessage, logEdit, EM_REPLACESEL, 0, uintptr(unsafe.Pointer(u16(line))))
}
func setStatus(s string) { call(setWindowText, statusLabel, uintptr(unsafe.Pointer(u16(s)))) }

func updateDevices() {
	previous := uint64(0)
	selected := int(call(sendMessage, combo, CB_GETCURSEL, 0, 0))
	if selected >= 0 && selected < len(btDevices) {
		previous = btDevices[selected].address
	}
	call(sendMessage, combo, CB_RESETCONTENT, 0, 0)
	devices, err := listPairedDevices()
	if err != nil {
		btDevices = nil
		logUI(err.Error())
		setStatus("请先在 Windows 蓝牙设置中配对手机")
		return
	}
	btDevices = devices
	index := 0
	saved := loadSavedAddress()
	for i, d := range devices {
		label := fmt.Sprintf("%s (%012X)", d.name, d.address)
		call(sendMessage, combo, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(u16(label))))
		if d.address == previous || (previous == 0 && d.address == saved) {
			index = i
		}
	}
	if len(devices) > 0 {
		call(sendMessage, combo, CB_SETCURSEL, uintptr(index), 0)
	}
	logUI(fmt.Sprintf("已发现 %d 个已配对蓝牙设备", len(devices)))
}

func sendUI(msg string) {
	uiMu.Lock()
	uiMessages = append(uiMessages, msg)
	uiMu.Unlock()
	call(postMessage, window, WM_LOG, 0, 0)
}
func sendAlert(kind, title, body string) {
	uiMu.Lock()
	uiAlerts = append(uiAlerts, alertMessage{kind: kind, title: title, body: body})
	uiMu.Unlock()
	call(postMessage, window, WM_ALERT, 0, 0)
}
func sendStatus(msg string) {
	uiMu.Lock()
	uiMessages = append(uiMessages, "状态："+msg)
	uiMu.Unlock()
	call(postMessage, window, WM_USER_STATUS, 0, 0)
}
func handleMessages(setState bool) {
	uiMu.Lock()
	messages := uiMessages
	uiMessages = nil
	uiMu.Unlock()
	for _, msg := range messages {
		if strings.HasPrefix(msg, "状态：") {
			setStatus(strings.TrimPrefix(msg, "状态："))
		} else {
			logUI(msg)
		}
	}
}

func tray(action uint32, title, msg string) {
	var n notifyIconData
	n.size = uint32(unsafe.Sizeof(n))
	n.hwnd = window
	n.id = 1
	n.callbackMsg = WM_TRAY
	n.icon = trayIcon
	n.flags = NIF_MESSAGE | NIF_ICON | NIF_TIP
	copy16(n.tip[:], "蓝牙来电短信提醒")
	if action == NIM_MODIFY {
		n.flags |= NIF_INFO
		copy16(n.infoTitle[:], title)
		copy16(n.info[:], msg)
		n.infoFlags = NIIF_INFO
	}
	call(notifyIcon, uintptr(action), uintptr(unsafe.Pointer(&n)))
}
func showAlertsNow() { sendAlert("test", "", "") }
func showAlerts() {
	uiMu.Lock()
	alerts := uiAlerts
	uiAlerts = nil
	uiMu.Unlock()
	for _, alert := range alerts {
		kind := alert.kind
		title, body := "蓝牙测试成功", "手机与电脑蓝牙连接正常"
		if kind == "sms" {
			title, body = "手机收到新短信", "请打开手机查看短信"
		}
		if kind == "call" {
			title, body = "手机有来电", "请查看手机上的来电"
		}
		if alert.title != "" {
			title = alert.title
		}
		if alert.body != "" {
			body = alert.body
		}
		if kind == "status" {
			setStatus(body)
			logUI(title + "：" + body)
			continue
		}
		logUI(title + "：" + body)
		appendHistory(title, body)
		tray(NIM_MODIFY, title, body)
		showAlertCard(title, body)
	}
}

func alertControl(class, value string, style uint32, x, y, width, height int32) uintptr {
	h := call(createWindow, 0, uintptr(unsafe.Pointer(u16(class))), uintptr(unsafe.Pointer(u16(value))),
		uintptr(style|WS_CHILD|WS_VISIBLE), uintptr(x), uintptr(y), uintptr(width), uintptr(height), alertWindow, 0, 0, 0)
	call(sendMessage, h, WM_SETFONT, uiFont, 1)
	return h
}

func showAlertCard(title, body string) {
	if alertWindow == 0 {
		instance := call(getModuleHandle, 0)
		alertWindow = call(createWindow, 0, uintptr(unsafe.Pointer(u16("BluetoothPhoneAlertsGoWindow"))),
			uintptr(unsafe.Pointer(u16("蓝牙提醒"))), WS_POPUP|WS_CAPTION|WS_SYSMENU,
			0, 0, 460, 230, 0, 0, instance, 0)
		if alertWindow == 0 {
			return
		}
		alertHeader = alertControl("STATIC", "  手机通知", 0, 0, 0, 460, 38)
		alertTitle = alertControl("STATIC", "", 0, 22, 53, 415, 34)
		call(sendMessage, alertTitle, WM_SETFONT, titleFont, 1)
		alertBody = alertControl("STATIC", "", 0, 22, 96, 415, 100)
		alertControl("STATIC", "完整记录请在主窗口查看 · 15 秒后自动收起", 0, 22, 198, 415, 22)
	}
	call(setWindowText, alertTitle, uintptr(unsafe.Pointer(u16(title))))
	// Keep the complete message in history; the preview fits inside the card.
	preview := []rune(body)
	if len(preview) > 180 {
		preview = append(preview[:180], []rune("…")...)
	}
	call(setWindowText, alertBody, uintptr(unsafe.Pointer(u16(string(preview)))))
	x := int32(call(getSystemMetrics, SM_CXSCREEN)) - 480
	y := int32(call(getSystemMetrics, SM_CYSCREEN)) - 270
	call(setWindowPos, alertWindow, HWND_TOPMOST, uintptr(x), uintptr(y), 460, 230, SWP_NOACTIVATE)
	call(showWindow, alertWindow, SW_SHOWNOACTIVATE)
	call(killTimer, alertWindow, POPUP_TIMER)
	call(setTimer, alertWindow, POPUP_TIMER, 15000, 0)
}

func historyPath() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "BluetoothPhoneAlerts", "notification-history.txt")
}

type historyEntry struct {
	when  time.Time
	title string
	body  string
}

var historyHeader = regexp.MustCompile(`(?m)^\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\] ([^\r\n]*)\r?$`)

func readHistory() []historyEntry {
	data, err := os.ReadFile(historyPath())
	if err != nil {
		return nil
	}
	return parseHistory(string(data), time.Now())
}

func parseHistory(content string, now time.Time) []historyEntry {
	matches := historyHeader.FindAllStringSubmatchIndex(content, -1)
	entries := make([]historyEntry, 0, len(matches))
	cutoff := now.Add(-7 * 24 * time.Hour)
	for i, match := range matches {
		when, err := time.ParseInLocation("2006-01-02 15:04:05", content[match[2]:match[3]], time.Local)
		if err != nil || when.Before(cutoff) {
			continue
		}
		end := len(content)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		body := strings.Trim(content[match[1]:end], "\r\n")
		entries = append(entries, historyEntry{when, content[match[4]:match[5]], body})
	}
	if len(entries) > 30 {
		entries = entries[len(entries)-30:]
	}
	return entries
}

func saveHistory(entries []historyEntry) error {
	path := historyPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	var out strings.Builder
	for _, entry := range entries {
		fmt.Fprintf(&out, "[%s] %s\r\n%s\r\n\r\n", entry.when.Format("2006-01-02 15:04:05"), entry.title, entry.body)
	}
	return os.WriteFile(path, []byte(out.String()), 0600)
}

func appendHistory(title, body string) {
	entries := append(readHistory(), historyEntry{time.Now(), title, body})
	if len(entries) > 30 {
		entries = entries[len(entries)-30:]
	}
	if err := saveHistory(entries); err != nil {
		logUI("保存通知记录失败：" + err.Error())
	}
}
func loadHistory() {
	entries := readHistory()
	if err := saveHistory(entries); err != nil {
		logUI("清理过期记录失败：" + err.Error())
		return
	}
	data, _ := os.ReadFile(historyPath())
	call(setWindowText, logEdit, uintptr(unsafe.Pointer(u16(string(data)))))
}
func clearHistory() {
	if err := os.Remove(historyPath()); err != nil && !os.IsNotExist(err) {
		logUI("清空记录失败：" + err.Error())
		return
	}
	call(setWindowText, logEdit, uintptr(unsafe.Pointer(u16(""))))
	logUI("历史通知已清空")
}
func startConnection() {
	connMu.Lock()
	if cancel != nil {
		connMu.Unlock()
		return
	}
	selected := int(call(sendMessage, combo, CB_GETCURSEL, 0, 0))
	if selected < 0 || selected >= len(btDevices) {
		connMu.Unlock()
		call(messageBox, window, uintptr(unsafe.Pointer(u16("请先配对并选择手机"))), uintptr(unsafe.Pointer(u16("提示"))), MB_OK|MB_ICONWARNING)
		return
	}
	device := btDevices[selected]
	saveAddress(device.address)
	cancel = make(chan struct{})
	c := cancel
	connMu.Unlock()
	logUI("正在连接：" + device.name)
	go connectionLoop(device, c)
}
func stopConnection() {
	connMu.Lock()
	if cancel != nil {
		close(cancel)
		cancel = nil
		if activeSocket != INVALID_SOCKET {
			call(wsaClose, activeSocket)
			activeSocket = INVALID_SOCKET
		}
	}
	connMu.Unlock()
	setStatus("已断开")
}

func wsaStart() error {
	var data [512]byte
	if err := call(wsaStartup, 0x202, uintptr(unsafe.Pointer(&data[0]))); err != 0 {
		return fmt.Errorf("WSAStartup 失败：%d", err)
	}
	return nil
}
func isCanceled(c <-chan struct{}) bool {
	select {
	case <-c:
		return true
	default:
		return false
	}
}
func registerSocket(sock uintptr, c <-chan struct{}) bool {
	connMu.Lock()
	defer connMu.Unlock()
	if cancel == nil || isCanceled(c) {
		return false
	}
	activeSocket = sock
	return true
}
func releaseSocket(sock uintptr) {
	connMu.Lock()
	if activeSocket == sock {
		activeSocket = INVALID_SOCKET
		call(wsaClose, sock)
	}
	connMu.Unlock()
}

func connectionLoop(device btDevice, c <-chan struct{}) {
	for !isCanceled(c) {
		sendStatus("蓝牙连接中…")
		sock := call(wsaSocket, AF_BTH, SOCK_STREAM, BTHPROTO_RFCOMM)
		if sock == INVALID_SOCKET {
			sendUI("创建蓝牙套接字失败：" + syscallError())
			break
		}
		if !registerSocket(sock, c) {
			call(wsaClose, sock)
			return
		}
		// SOCKADDR_BTH is a packed 30-byte structure. A client that supplies
		// a service UUID leaves port zero so Windows resolves it through SDP.
		addr := sockAddrBth{family: AF_BTH}
		binary.LittleEndian.PutUint64(addr.address[:], device.address)
		binary.LittleEndian.PutUint32(addr.classID[0:4], serviceGUID.data1)
		binary.LittleEndian.PutUint16(addr.classID[4:6], serviceGUID.data2)
		binary.LittleEndian.PutUint16(addr.classID[6:8], serviceGUID.data3)
		copy(addr.classID[8:], serviceGUID.data4[:])
		r, _, connectErr := wsaConnect.Call(sock, uintptr(unsafe.Pointer(&addr)), uintptr(unsafe.Sizeof(addr)))
		if r == SOCKET_ERROR {
			sendUI(fmt.Sprintf("连接失败：WinSock %v；3 秒后重试", connectErr))
		} else {
			sendStatus("已连接：" + device.name)
			sendUI("已建立 RFCOMM 连接，等待手机通知")
			if err := receiveLoop(sock, c); err != nil && !isCanceled(c) {
				sendUI("连接断开：" + err.Error())
			}
		}
		releaseSocket(sock)
		if isCanceled(c) {
			break
		}
		select {
		case <-c:
			return
		case <-time.After(3 * time.Second):
		}
	}
}
func rememberID(id string) bool {
	seenMu.Lock()
	defer seenMu.Unlock()
	if seenIds[id] {
		return false
	}
	seenIds[id] = true
	seenOrder = append(seenOrder, id)
	if len(seenOrder) > 120 {
		delete(seenIds, seenOrder[0])
		seenOrder = seenOrder[1:]
	}
	return true
}
func socketSend(sock uintptr, data []byte) error {
	for len(data) > 0 {
		n, _, sendErr := wsaSend.Call(sock, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), 0)
		if n == SOCKET_ERROR {
			return fmt.Errorf("发送 ACK 失败：%v", sendErr)
		}
		if n == 0 {
			return fmt.Errorf("蓝牙连接已关闭")
		}
		data = data[n:]
	}
	return nil
}
func receiveLoop(sock uintptr, c <-chan struct{}) error {
	var pending []byte
	var buf [1024]byte
	for !isCanceled(c) {
		n, _, recvErr := wsaRecv.Call(sock, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0)
		if n == SOCKET_ERROR {
			return fmt.Errorf("蓝牙接收错误：%v", recvErr)
		}
		if n == 0 {
			return fmt.Errorf("手机关闭了连接")
		}
		pending = append(pending, buf[:n]...)
		for {
			idx := bytes.IndexByte(pending, '\n')
			if idx < 0 {
				break
			}
			line := bytes.TrimSpace(pending[:idx])
			pending = pending[idx+1:]
			var value struct {
				ID    string `json:"id"`
				Type  string `json:"type"`
				Title string `json:"title"`
				Body  string `json:"body"`
			}
			if err := json.Unmarshal(line, &value); err != nil {
				sendUI("忽略无效的手机消息")
				continue
			}
			if len(value.ID) == 0 || len(value.ID) > 100 {
				continue
			}
			if value.Type != "sms" && value.Type != "call" && value.Type != "test" && value.Type != "status" {
				continue
			}
			if err := socketSend(sock, []byte("ACK|"+value.ID+"\n")); err != nil {
				return err
			}
			if rememberID(value.ID) {
				sendAlert(value.Type, value.Title, value.Body)
			}
		}
		if len(pending) > 8192 {
			return fmt.Errorf("收到过长消息")
		}
	}
	return nil
}

func windowProcedure(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	if hwnd == alertWindow && alertWindow != 0 {
		switch msg {
		case WM_CLOSE, WM_TIMER:
			call(killTimer, alertWindow, POPUP_TIMER)
			call(showWindow, alertWindow, SW_HIDE)
			return 0
		case WM_DESTROY:
			alertWindow = 0
			return 0
		}
	}
	switch msg {
	case WM_COMMAND:
		switch wParam & 0xffff {
		case IDC_REFRESH:
			updateDevices()
		case IDC_CONNECT:
			startConnection()
		case IDC_STOP:
			stopConnection()
			logUI("连接已手动停止")
		case IDC_TEST:
			showAlertsNow()
		case IDC_EXIT:
			closing = true
			call(destroyWindow, window)
		case IDC_CLEAR:
			clearHistory()
		}
		return 0
	case WM_LOG, WM_USER_STATUS:
		handleMessages(true)
		return 0
	case WM_ALERT:
		showAlerts()
		return 0
	case WM_TRAY:
		if lParam == WM_LBUTTONDBLCLK || lParam == WM_RBUTTONUP {
			call(showWindow, window, SW_RESTORE)
			call(updateWindow, window)
		}
		return 0
	case WM_CTLCOLORSTATIC:
		call(setBkMode, wParam, 1)
		if lParam == headerPanel || lParam == headerTitle || lParam == headerSubtitle || lParam == alertHeader {
			if lParam == headerSubtitle {
				call(setTextColor, wParam, 0x00F5E6DB)
			} else {
				call(setTextColor, wParam, 0x00FFFFFF)
			}
			return headerBrush
		}
		call(setTextColor, wParam, 0x0035291E)
		return backgroundBrush
	case WM_CLOSE:
		if !closing {
			call(showWindow, window, SW_HIDE)
			return 0
		}
		return call(defWindowProc, hwnd, uintptr(msg), wParam, lParam)
	case WM_DESTROY:
		stopConnection()
		if alertWindow != 0 {
			call(destroyWindow, alertWindow)
		}
		tray(NIM_DELETE, "", "")
		call(postQuitMessage, 0)
		return 0
	}
	return call(defWindowProc, hwnd, uintptr(msg), wParam, lParam)
}

func main() {
	runtime.LockOSThread()
	if err := wsaStart(); err != nil {
		call(messageBox, 0, uintptr(unsafe.Pointer(u16(err.Error()))), uintptr(unsafe.Pointer(u16("蓝牙启动失败"))), MB_OK|MB_ICONWARNING)
		return
	}
	defer call(wsaCleanup)
	className := u16("BluetoothPhoneAlertsGoWindow")
	instance := call(getModuleHandle, 0)
	backgroundBrush = call(createSolidBrush, 0x00FBF7F4)
	headerBrush = call(createSolidBrush, 0x00EB6326)
	uiFont = call(createFont, uintptr(^uint32(15)), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0,
		uintptr(unsafe.Pointer(u16("Segoe UI"))))
	titleFont = call(createFont, uintptr(^uint32(27)), 0, 0, 0, 700, 0, 0, 0, 1, 0, 0, 5, 0,
		uintptr(unsafe.Pointer(u16("Segoe UI"))))
	trayIcon = call(loadIcon, 0, 32516) // IDI_INFORMATION
	wc := wndClassEx{size: uint32(unsafe.Sizeof(wndClassEx{})), style: 3, wndProc: syscall.NewCallback(windowProcedure), instance: instance,
		cursor: call(loadCursor, 0, 32512), icon: trayIcon, iconSmall: trayIcon, background: backgroundBrush, className: className}
	if call(registerClass, uintptr(unsafe.Pointer(&wc))) == 0 {
		return
	}
	window = call(createWindow, 0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(u16("蓝牙来电短信提醒（纯蓝牙）"))),
		WS_OVERLAPPEDWINDOW, 120, 100, 720, 550, 0, 0, instance, 0)
	if window == 0 {
		return
	}
	headerPanel = control("STATIC", "", 0, 0, 0, 704, 104, 0)
	headerTitle = control("STATIC", "蓝牙提醒助手", 0, 26, 18, 350, 40, 0)
	call(sendMessage, headerTitle, WM_SETFONT, titleFont, 1)
	headerSubtitle = control("STATIC", "让手机来电与短信及时出现在电脑上", 0, 28, 62, 430, 26, 0)
	control("STATIC", "已配对设备", 0, 24, 127, 95, 26, 0)
	combo = control("COMBOBOX", "", CBS_DROPDOWNLIST|WS_VSCROLL, 124, 120, 550, 250, IDC_DEVICE)
	control("BUTTON", "刷新设备", 0, 24, 164, 112, 38, IDC_REFRESH)
	control("BUTTON", "连接 / 自动重连", 0, 148, 164, 174, 38, IDC_CONNECT)
	control("BUTTON", "停止连接", 0, 334, 164, 120, 38, IDC_STOP)
	control("BUTTON", "测试提醒", 0, 466, 164, 112, 38, IDC_TEST)
	control("BUTTON", "退出", 0, 590, 164, 84, 38, IDC_EXIT)
	control("STATIC", "连接状态", 0, 24, 226, 80, 25, 0)
	statusLabel = control("STATIC", "未连接", 0, 108, 226, 566, 25, 0)
	control("STATIC", "运行记录", 0, 24, 264, 100, 25, 0)
	control("BUTTON", "清空记录", 0, 562, 258, 112, 30, IDC_CLEAR)
	logEdit = control("EDIT", "", WS_BORDER|WS_VSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_READONLY, 24, 292, 650, 164, 0)
	control("STATIC", "关闭窗口后仍会在系统托盘运行 · 双击托盘图标可重新打开", 0, 24, 472, 500, 24, 0)
	loadHistory()
	tray(NIM_ADD, "", "")
	updateDevices()
	logUI("双击托盘图标可以打开窗口。关闭窗口会隐藏到后台。")
	hidden := false
	for _, arg := range os.Args[1:] {
		if strings.EqualFold(arg, "--background") {
			hidden = true
		}
	}
	if !hidden {
		call(showWindow, window, SW_SHOW)
		call(updateWindow, window)
	}
	if loadSavedAddress() != 0 {
		startConnection()
	}
	var m winMsg
	for call(getMessage, uintptr(unsafe.Pointer(&m)), 0, 0, 0) != 0 {
		call(translateMessage, uintptr(unsafe.Pointer(&m)))
		call(dispatchMessage, uintptr(unsafe.Pointer(&m)))
	}
}
