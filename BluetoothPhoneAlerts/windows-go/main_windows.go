//go:build windows

// BluetoothPhoneAlerts Windows client. Pure Windows APIs (no runtime dependencies).
// Windows 10/11 + Bluetooth Classic RFCOMM. Android hosts a paired secure RFCOMM server.
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
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
    AF_BTH = 32
    SOCK_STREAM = 1
    BTHPROTO_RFCOMM = 3
    BT_PORT_ANY = 0xffffffff
    INVALID_SOCKET = ^uintptr(0)

    WS_OVERLAPPEDWINDOW = 0x00CF0000
    WS_VISIBLE = 0x10000000
    WS_CHILD = 0x40000000
    WS_VSCROLL = 0x00200000
    WS_BORDER = 0x00800000
    WS_EX_CLIENTEDGE = 0x00000200
    CBS_DROPDOWNLIST = 3
    ES_MULTILINE = 4
    ES_AUTOVSCROLL = 0x40
    ES_READONLY = 0x800
    WS_HSCROLL = 0x00100000
    WM_DESTROY = 0x0002
    WM_CLOSE = 0x0010
    WM_COMMAND = 0x0111
    WM_SETFONT = 0x0030
    WM_APP = 0x8000
    WM_LBUTTONDBLCLK = 0x0203
    WM_RBUTTONUP = 0x0205
    SW_HIDE = 0
    SW_SHOW = 5
    SW_RESTORE = 9
    MB_OK = 0
    MB_ICONWARNING = 0x30
    CB_ADDSTRING = 0x0143
    CB_RESETCONTENT = 0x014B
    CB_GETCURSEL = 0x0147
    CB_SETCURSEL = 0x014E
    EM_SETSEL = 0x00B1
    EM_REPLACESEL = 0x00C2
    NIM_ADD = 0
    NIM_MODIFY = 1
    NIM_DELETE = 2
    NIF_MESSAGE = 1
    NIF_ICON = 2
    NIF_TIP = 4
    NIF_INFO = 0x10
    NIIF_INFO = 1
    WM_TRAY = WM_APP + 1
    WM_LOG = WM_APP + 2
    WM_ALERT = WM_APP + 3

    IDC_REFRESH = 100
    IDC_CONNECT = 101
    IDC_STOP = 102
    IDC_DEVICE = 103
    IDC_TEST = 104
    IDC_EXIT = 105
    WM_USER_STATUS = WM_APP + 4
)

type guid struct { data1 uint32; data2 uint16; data3 uint16; data4 [8]byte }
var serviceGUID = guid{0xbac20533, 0x10ac, 0x4c14, [8]byte{0xb8,0xe9,0x45,0xa1,0xb2,0x9f,0xeb,0x17}}

type sockAddrBth struct {
    family uint16
    _ [6]byte
    address uint64
    classID guid
    port uint32
    _ [4]byte
}

type bluetoothSearchParams struct {
    size uint32
    authenticated uint32
    remembered uint32
    unknown uint32
    connected uint32
    inquiry uint32
    timeoutMultiplier byte
    _ [7]byte
    radio uintptr
}
type systemTime struct { year, month, dayOfWeek, day, hour, minute, second, milliseconds uint16 }
type bluetoothDeviceInfo struct {
    size uint32
    _ uint32
    address uint64
    classOfDevice uint32
    connected uint32
    remembered uint32
    authenticated uint32
    lastSeen systemTime
    lastUsed systemTime
    name [248]uint16
}
type wndClassEx struct {
    size uint32; style uint32; wndProc uintptr
    clsExtra int32; wndExtra int32
    instance uintptr; icon uintptr; cursor uintptr; background uintptr
    menuName *uint16; className *uint16; iconSmall uintptr
}
type point struct { x, y int32 }
type winMsg struct {
    hwnd uintptr; message uint32; _ uint32
    wParam, lParam uintptr
    time uint32; pt point; lPrivate uint32
}
type notifyIconData struct {
    size uint32; _ uint32
    hwnd uintptr
    id uint32; flags uint32; callbackMsg uint32; _ uint32
    icon uintptr
    tip [128]uint16
    state uint32; stateMask uint32
    info [256]uint16
    version uint32
    infoTitle [64]uint16
    infoFlags uint32
    itemGuid guid
    balloonIcon uintptr
}
type btDevice struct { address uint64; name string }

var (
    user32 = syscall.NewLazyDLL("user32.dll")
    kernel32 = syscall.NewLazyDLL("kernel32.dll")
    gdi32 = syscall.NewLazyDLL("gdi32.dll")
    shell32 = syscall.NewLazyDLL("shell32.dll")
    bthapi = syscall.NewLazyDLL("BluetoothApis.dll")
    ws2 = syscall.NewLazyDLL("ws2_32.dll")

    registerClass = user32.NewProc("RegisterClassExW")
    createWindow = user32.NewProc("CreateWindowExW")
    defWindowProc = user32.NewProc("DefWindowProcW")
    getMessage = user32.NewProc("GetMessageW")
    translateMessage = user32.NewProc("TranslateMessage")
    dispatchMessage = user32.NewProc("DispatchMessageW")
    postQuitMessage = user32.NewProc("PostQuitMessage")
    postMessage = user32.NewProc("PostMessageW")
    sendMessage = user32.NewProc("SendMessageW")
    showWindow = user32.NewProc("ShowWindow")
    updateWindow = user32.NewProc("UpdateWindow")
    setWindowText = user32.NewProc("SetWindowTextW")
    messageBox = user32.NewProc("MessageBoxW")
    destroyWindow = user32.NewProc("DestroyWindow")
    loadIcon = user32.NewProc("LoadIconW")
    loadCursor = user32.NewProc("LoadCursorW")
    getModuleHandle = kernel32.NewProc("GetModuleHandleW")
    getStockObject = gdi32.NewProc("GetStockObject")
    notifyIcon = shell32.NewProc("Shell_NotifyIconW")
    findFirstDevice = bthapi.NewProc("BluetoothFindFirstDevice")
    findNextDevice = bthapi.NewProc("BluetoothFindNextDevice")
    findDeviceClose = bthapi.NewProc("BluetoothFindDeviceClose")
    wsaStartup = ws2.NewProc("WSAStartup")
    wsaCleanup = ws2.NewProc("WSACleanup")
    wsaSocket = ws2.NewProc("socket")
    wsaConnect = ws2.NewProc("connect")
    wsaRecv = ws2.NewProc("recv")
    wsaSend = ws2.NewProc("send")
    wsaClose = ws2.NewProc("closesocket")
    wsaGetLastError = ws2.NewProc("WSAGetLastError")

    window uintptr
    combo uintptr
    logEdit uintptr
    statusLabel uintptr
    btDevices []btDevice
    uiMu sync.Mutex
    uiMessages []string
    uiAlerts []string
    connMu sync.Mutex
    activeSocket uintptr = INVALID_SOCKET
    cancel chan struct{}
    closing bool
    trayIcon uintptr
    seenMu sync.Mutex
    seenIds = make(map[string]bool)
    seenOrder []string
)

func u16(s string) *uint16 { return syscall.StringToUTF16Ptr(s) }
func copy16(dest []uint16, value string) { s := utf16.Encode([]rune(value)); if len(s)>=len(dest) {s=s[:len(dest)-1]}; copy(dest,s) }
func str16(a []uint16) string { n:=0;for n<len(a)&&a[n]!=0 {n++}; return string(utf16.Decode(a[:n])) }
func call(p *syscall.LazyProc, args ...uintptr) uintptr { r,_,_:=p.Call(args...);return r }
func syscallError() string { return fmt.Sprintf("WinSock 错误 %d",call(wsaGetLastError)) }

func listPairedDevices() ([]btDevice,error) {
    params := bluetoothSearchParams{size:uint32(unsafe.Sizeof(bluetoothSearchParams{})), authenticated:1, remembered:1}
    var info bluetoothDeviceInfo
    info.size = uint32(unsafe.Sizeof(info))
    handle := call(findFirstDevice, uintptr(unsafe.Pointer(&params)),uintptr(unsafe.Pointer(&info)))
    if handle==0 {return nil,fmt.Errorf("没有发现已配对蓝牙设备（或电脑蓝牙未开启）")}
    defer call(findDeviceClose,handle)
    result:=make([]btDevice,0)
    for {
        if info.authenticated!=0 || info.remembered!=0 {
            result=append(result,btDevice{address:info.address,name:str16(info.name[:])})
        }
        info.size=uint32(unsafe.Sizeof(info))
        if call(findNextDevice,handle,uintptr(unsafe.Pointer(&info)))==0 {break}
    }
    return result,nil
}

func loadSavedAddress() uint64 {
    data,err:=os.ReadFile(savedPath()); if err!=nil{return 0}
    number,err:=strconv.ParseUint(strings.TrimSpace(string(data)),16,64)
    if err!=nil{return 0};return number
}
func savedPath() string { return filepath.Join(os.Getenv("LOCALAPPDATA"),"BluetoothPhoneAlerts","paired-device-go.txt") }
func saveAddress(value uint64) {
    path:=savedPath();_ = os.MkdirAll(filepath.Dir(path),0700)
    _ = os.WriteFile(path,[]byte(fmt.Sprintf("%012X",value)),0600)
}

func winClass(name string) *uint16 { return u16(name) }
func control(class,title string, style uint32, x,y,w,h int32, id uintptr) uintptr {
    hnd:=call(createWindow,0,uintptr(unsafe.Pointer(winClass(class))),uintptr(unsafe.Pointer(u16(title))),
        uintptr(style|WS_CHILD|WS_VISIBLE),uintptr(x),uintptr(y),uintptr(w),uintptr(h),window,id,0,0)
    font:=call(getStockObject,17)
    call(sendMessage,hnd,WM_SETFONT,font,1)
    return hnd
}
func logUI(s string) {
    line:=fmt.Sprintf("[%s] %s\r\n",time.Now().Format("15:04:05"),s)
    call(sendMessage,logEdit,EM_SETSEL,^uintptr(0),^uintptr(0))
    call(sendMessage,logEdit,EM_REPLACESEL,0,uintptr(unsafe.Pointer(u16(line))))
}
func setStatus(s string) { call(setWindowText,statusLabel,uintptr(unsafe.Pointer(u16(s)))) }

func updateDevices() {
    previous:=uint64(0)
    selected:=int(call(sendMessage,combo,CB_GETCURSEL,0,0))
    if selected>=0&&selected<len(btDevices){previous=btDevices[selected].address}
    call(sendMessage,combo,CB_RESETCONTENT,0,0)
    devices,err:=listPairedDevices()
    if err!=nil{logUI(err.Error());setStatus("请先在 Windows 蓝牙设置中配对手机");return}
    btDevices=devices
    index:=0
    saved:=loadSavedAddress()
    for i,d:=range devices {
        label:=fmt.Sprintf("%s (%012X)",d.name,d.address)
        call(sendMessage,combo,CB_ADDSTRING,0,uintptr(unsafe.Pointer(u16(label))))
        if d.address==previous||(previous==0&&d.address==saved){index=i}
    }
    if len(devices)>0 {call(sendMessage,combo,CB_SETCURSEL,uintptr(index),0)}
    logUI(fmt.Sprintf("已发现 %d 个已配对蓝牙设备",len(devices)))
}

func sendUI(msg string) { uiMu.Lock();uiMessages=append(uiMessages,msg);uiMu.Unlock();call(postMessage,window,WM_LOG,0,0) }
func sendAlert(kind string) { uiMu.Lock();uiAlerts=append(uiAlerts,kind);uiMu.Unlock();call(postMessage,window,WM_ALERT,0,0) }
func sendStatus(msg string) { uiMu.Lock();uiMessages=append(uiMessages,"状态："+msg);uiMu.Unlock();call(postMessage,window,WM_USER_STATUS,0,0) }
func handleMessages(setState bool) {
    uiMu.Lock();messages:=uiMessages;uiMessages=nil;uiMu.Unlock()
    for _,msg:=range messages {
        if strings.HasPrefix(msg,"状态：") {setStatus(strings.TrimPrefix(msg,"状态："))} else {logUI(msg)}
    }
}

func tray(action uint32, title,msg string) {
    var n notifyIconData
    n.size=uint32(unsafe.Sizeof(n));n.hwnd=window;n.id=1
    n.callbackMsg=WM_TRAY;n.icon=trayIcon
    n.flags=NIF_MESSAGE|NIF_ICON|NIF_TIP
    copy16(n.tip[:],"蓝牙来电短信提醒")
    if action==NIM_MODIFY {
        n.flags|=NIF_INFO
        copy16(n.infoTitle[:],title)
        copy16(n.info[:],msg)
        n.infoFlags=NIIF_INFO
    }
    call(notifyIcon,uintptr(action),uintptr(unsafe.Pointer(&n)))
}
func showAlertsNow() { sendAlert("test") }
func showAlerts() {
    uiMu.Lock();alerts:=uiAlerts;uiAlerts=nil;uiMu.Unlock()
    for _,kind:=range alerts {
        title,body:="蓝牙测试成功","手机与电脑蓝牙连接正常"
        if kind=="sms"{title,body="手机收到新短信","请打开手机查看短信"}
        if kind=="call"{title,body="手机有来电","请查看手机上的来电"}
        logUI(title)
        tray(NIM_MODIFY,title,body)
    }
}
func startConnection() {
    connMu.Lock()
    if cancel!=nil{connMu.Unlock();return}
    selected:=int(call(sendMessage,combo,CB_GETCURSEL,0,0))
    if selected<0||selected>=len(btDevices){connMu.Unlock();call(messageBox,window,uintptr(unsafe.Pointer(u16("请先配对并选择手机"))),uintptr(unsafe.Pointer(u16("提示"))),MB_OK|MB_ICONWARNING);return}
    device:=btDevices[selected]
    saveAddress(device.address)
    cancel=make(chan struct{})
    c:=cancel
    connMu.Unlock()
    logUI("正在连接："+device.name)
    go connectionLoop(device,c)
}
func stopConnection() {
    connMu.Lock()
    if cancel!=nil {
        close(cancel)
        cancel=nil
        if activeSocket!=INVALID_SOCKET {call(wsaClose,activeSocket);activeSocket=INVALID_SOCKET}
    }
    connMu.Unlock()
    setStatus("已断开")
}

func wsaStart() error {
    var data [512]byte
    if err:=call(wsaStartup,0x202,uintptr(unsafe.Pointer(&data[0])));err!=0 {return fmt.Errorf("WSAStartup 失败：%d",err)}
    return nil
}
func isCanceled(c <-chan struct{}) bool {select {case <-c:return true;default:return false}}
func registerSocket(sock uintptr,c <-chan struct{}) bool {
    connMu.Lock();defer connMu.Unlock()
    if cancel==nil||isCanceled(c) {return false}
    activeSocket=sock;return true
}
func releaseSocket(sock uintptr) {
    connMu.Lock(); if activeSocket==sock{activeSocket=INVALID_SOCKET;call(wsaClose,sock)};connMu.Unlock()
}

func connectionLoop(device btDevice,c <-chan struct{}) {
    for !isCanceled(c) {
        sendStatus("蓝牙连接中…")
        sock:=call(wsaSocket,AF_BTH,SOCK_STREAM,BTHPROTO_RFCOMM)
        if sock==INVALID_SOCKET {
            sendUI("创建蓝牙套接字失败："+syscallError());break
        }
        if !registerSocket(sock,c) {call(wsaClose,sock);return}
        addr:=sockAddrBth{family:AF_BTH,address:device.address,classID:serviceGUID,port:BT_PORT_ANY}
        r:=call(wsaConnect,sock,uintptr(unsafe.Pointer(&addr)),uintptr(unsafe.Sizeof(addr)))
        if r==^uintptr(0) {
            sendUI("连接失败："+syscallError()+"；3 秒后重试")
        } else {
            sendStatus("已连接："+device.name)
            sendUI("已建立 RFCOMM 连接，等待手机通知")
            if err:=receiveLoop(sock,c);err!=nil&&!isCanceled(c){sendUI("连接断开："+err.Error())}
        }
        releaseSocket(sock)
        if isCanceled(c) {break}
        select {case <-c:return;case <-time.After(3*time.Second):}
    }
}
func rememberID(id string) bool {
    seenMu.Lock();defer seenMu.Unlock()
    if seenIds[id]{return false}
    seenIds[id]=true;seenOrder=append(seenOrder,id)
    if len(seenOrder)>120 {delete(seenIds,seenOrder[0]);seenOrder=seenOrder[1:]}
    return true
}
func socketSend(sock uintptr,data []byte) error {
    for len(data)>0 {
        n:=call(wsaSend,sock,uintptr(unsafe.Pointer(&data[0])),uintptr(len(data)),0)
        if n==^uintptr(0) {return fmt.Errorf("发送 ACK 失败：%s",syscallError())}
        if n==0 {return fmt.Errorf("蓝牙连接已关闭")}
        data=data[n:]
    }
    return nil
}
func receiveLoop(sock uintptr,c <-chan struct{}) error {
    var pending []byte
    var buf [1024]byte
    for !isCanceled(c) {
        n:=call(wsaRecv,sock,uintptr(unsafe.Pointer(&buf[0])),uintptr(len(buf)),0)
        if n==^uintptr(0) {return fmt.Errorf("蓝牙接收错误：%s",syscallError())}
        if n==0 {return fmt.Errorf("手机关闭了连接")}
        pending=append(pending,buf[:n]...)
        if len(pending)>8192 {return fmt.Errorf("收到过长消息")}
        for {
            idx:=bytes.IndexByte(pending,'\n');if idx<0 {break}
            line:=bytes.TrimSpace(pending[:idx]);pending=pending[idx+1:]
            var value struct {ID string `json:"id"`;Type string `json:"type"`}
            if err:=json.Unmarshal(line,&value);err!=nil {sendUI("忽略无效的手机消息");continue}
            if len(value.ID)==0||len(value.ID)>100 {continue}
            if value.Type!="sms"&&value.Type!="call"&&value.Type!="test"{continue}
            if err:=socketSend(sock,[]byte("ACK|"+value.ID+"\n"));err!=nil{return err}
            if rememberID(value.ID) {sendAlert(value.Type)}
        }
    }
    return nil
}

func windowProcedure(hwnd uintptr,msg uint32,wParam,lParam uintptr) uintptr {
    switch msg {
    case WM_COMMAND:
        switch wParam&0xffff {
        case IDC_REFRESH: updateDevices()
        case IDC_CONNECT: startConnection()
        case IDC_STOP: stopConnection();logUI("连接已手动停止")
        case IDC_TEST: showAlertsNow()
        case IDC_EXIT: closing=true;call(destroyWindow,window)
        }
        return 0
    case WM_LOG,WM_USER_STATUS:handleMessages(true);return 0
    case WM_ALERT:showAlerts();return 0
    case WM_TRAY:
        if lParam==WM_LBUTTONDBLCLK||lParam==WM_RBUTTONUP {
            call(showWindow,window,SW_RESTORE)
            call(updateWindow,window)
        }
        return 0
    case WM_CLOSE:
        if !closing {call(showWindow,window,SW_HIDE);return 0}
        return call(defWindowProc,hwnd,uintptr(msg),wParam,lParam)
    case WM_DESTROY:
        stopConnection();tray(NIM_DELETE,"","")
        call(postQuitMessage,0)
        return 0
    }
    return call(defWindowProc,hwnd,uintptr(msg),wParam,lParam)
}

func main() {
    runtime.LockOSThread()
    if err:=wsaStart();err!=nil {call(messageBox,0,uintptr(unsafe.Pointer(u16(err.Error()))),uintptr(unsafe.Pointer(u16("蓝牙启动失败"))),MB_OK|MB_ICONWARNING);return}
    defer call(wsaCleanup)
    className:=u16("BluetoothPhoneAlertsGoWindow")
    instance:=call(getModuleHandle,0)
    trayIcon=call(loadIcon,0,32516) // IDI_INFORMATION
    wc:=wndClassEx{size:uint32(unsafe.Sizeof(wndClassEx{})),style:3,wndProc:syscall.NewCallback(windowProcedure),instance:instance,
        cursor:call(loadCursor,0,32512),icon:trayIcon,iconSmall:trayIcon,className:className}
    if call(registerClass,uintptr(unsafe.Pointer(&wc)))==0 {return}
    window=call(createWindow,0,uintptr(unsafe.Pointer(className)),uintptr(unsafe.Pointer(u16("蓝牙来电短信提醒（纯蓝牙）"))),
        WS_OVERLAPPEDWINDOW,120,100,620,426,0,0,instance,0)
    if window==0 {return}
    control("STATIC","先在 Windows 和 Android 蓝牙设置中配对手机，并在手机端启动蓝牙服务",0,18,18,565,27,0)
    control("STATIC","已配对设备：",0,18,60,85,28,0)
    combo=control("COMBOBOX","",CBS_DROPDOWNLIST|WS_VSCROLL,105,55,470,250,IDC_DEVICE)
    control("BUTTON","刷新设备",0,18,97,105,34,IDC_REFRESH)
    control("BUTTON","连接 / 自动重连",0,140,97,156,34,IDC_CONNECT)
    control("BUTTON","停止连接",0,312,97,120,34,IDC_STOP)
    control("BUTTON","测试弹窗",0,447,97,120,34,IDC_TEST)
    statusLabel=control("STATIC","未连接",0,18,145,555,25,0)
    logEdit=control("EDIT","",WS_BORDER|WS_VSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_READONLY,18,175,558,145,0)
    control("BUTTON","退出程序",0,458,329,118,28,IDC_EXIT)
    tray(NIM_ADD,"","")
    updateDevices()
    logUI("双击托盘图标可以打开窗口。关闭窗口会隐藏到后台。")
    hidden := false
    for _,arg := range os.Args[1:] { if strings.EqualFold(arg,"--background") { hidden=true } }
    if !hidden { call(showWindow,window,SW_SHOW);call(updateWindow,window) }
    if loadSavedAddress()!=0 {startConnection()}
    var m winMsg
    for call(getMessage,uintptr(unsafe.Pointer(&m)),0,0,0)!=0 {
        call(translateMessage,uintptr(unsafe.Pointer(&m)))
        call(dispatchMessage,uintptr(unsafe.Pointer(&m)))
    }
}
