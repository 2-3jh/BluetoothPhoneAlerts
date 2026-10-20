# Windows 客户端 (Go, 无外部依赖)

- 用 Go 1.23 直接交叉编译 Windows amd64 程序：`GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-H=windowsgui -s -w' -o BluetoothPhoneAlerts.exe main_windows.go`
- Windows 蓝牙经典 RFCOMM，直接使用 Windows 系统蓝牙设备枚举、WinSock AF_BTH、自定义 UUID、系统托盘气泡通知。
- 不安装 .NET/Java/Node/Python，可放到任何文件夹双击运行；不联网。
- 点击“测试弹窗”无需 Android 即可测试 Windows 通知权限。点“退出程序”真正退出；关闭窗口隐藏到托盘。
- 这是 Linux 交叉编译生成的 Windows GUI 程序，暂未在 Windows 11 真机或蓝牙连接环境下进行实测。
