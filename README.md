# BluetoothPhoneAlerts — 纯蓝牙手机互联

Android 手机与 Windows 电脑之间的来电、短信、电量同步，以及手机向电脑发送文字和图片。通过已配对的蓝牙连接传输，无需 Wi-Fi、账号或云服务。

当前版本：**v0.5.2 测试版**。[下载安装包](https://github.com/2-3jh/BluetoothPhoneAlerts/releases)，无需登录 GitHub。

## 功能

- 来电提醒：授权后显示号码和联系人名称。
- 短信提醒：显示发送人和短信内容。
- 电量同步：连接时发送，电量或充电状态变化时更新，每分钟补发一次状态。
- 手机发送文字：每条最多 1200 个 UTF-16 字符（部分表情占两个）。
- 手机发送图片：支持多选，一次最多 20 张，每张最多 5 MB，依次传输；多选操作因手机系统文件选择器而异。
- 未连接时在手机排队，重新连接后发送；图片断线后从头重传，完整接收并通过 SHA-256 校验后保存。
- Windows 关闭主窗口后隐藏到托盘，仍能显示独立提醒；点击“退出程序”才会真正退出。
- 手机页面与常驻通知显示连接状态。页面打开时每秒刷新；空闲连接每分钟通过状态消息检查，异常断开可能稍有延迟。

目前文字和图片为 **手机 → 电脑** 单向传输。不监听微信、QQ 等 App 通知。

## 安装和使用

1. 在手机和电脑的系统蓝牙设置中完成配对。
2. 从 Releases 下载并安装 APK，授予附近设备等权限。短信、电话、通话记录、联系人权限用于对应提醒功能。
3. 手机上点击“开始蓝牙服务”。
4. 电脑运行 `BluetoothPhoneAlerts.exe`，选择手机，点击“连接 / 自动重连”。手机随后显示“已连接电脑”。
5. 点击“测试短信”或“测试来电”验证；在“发送到电脑”区域输入文字或选择图片。

图片传输需要两端均为 v0.5.0 或更新版本。v0.5.1 的多选和手机连接状态修复只需升级 APK，电脑 v0.5.0 可继续使用。

Windows 使用内嵌 HTML/CSS/JavaScript 和 WebView2；缺少 WebView2 Runtime 时需先安装该组件。用户无需安装 Go、Java、Python、Node.js 或 .NET 开发环境。`install-windows.bat` 可选地设置 Windows 开机启动。手机重启后请手动启动服务。

## 图片与记录

- 电脑图片目录：Windows“下载”文件夹中的“蓝牙互联”子文件夹，可点击“打开图片文件夹”；支持系统中移动过的下载目录，同名文件自动编号。v0.5.2 之前收到的图片仍在 `%LOCALAPPDATA%\BluetoothPhoneAlerts\Received`，不会自动搬移。
- 电脑通知历史：`%LOCALAPPDATA%\BluetoothPhoneAlerts\notification-history.txt`，最多 30 条、最长 7 天，可点击“清空记录”。
- **清空通知记录不会删除图片；已接收图片不会按 7 天自动清理，需自行管理。**
- 手机队列存于应用私有目录，最多 100 条，满后淘汰旧条目；待发图片也保存在私有目录，确认发送后删除缓存。加入队列不代表电脑已经接收。

内容保存在本地，通知历史为明文，请注意设备隐私。蓝牙采用 secure RFCOMM，只允许已配对设备连接。

## v0.5.2 更新

- 新收到的图片保存到 Windows“下载/蓝牙互联”，文件夹按钮同步更新。这项改动需更新电脑 EXE；手机 v0.5.1 可继续使用。

## v0.5.1 更新

- 图片选择器支持多选，并依次加入队列。
- 修复连接成功后手机仍显示“等待电脑连接”，常驻通知也同步更新。
- 页面离开前台时停止界面刷新。
- 每分钟补发电量状态，以便检查空闲连接。

## 常见问题

- **连接失败**：确认已配对、手机服务已启动、两端蓝牙开启；检查后台限制和蓝牙驱动，必要时重新配对。
- **手机后台服务停止**：允许后台运行并检查厂商省电设置，不同手机的持续性可能不同。
- **图片不发送**：检查每张是否超过 5 MB、电脑是否为 v0.5.0 以上；大图蓝牙传输可能较慢。
- **收不到真实来电或短信**：检查对应权限；VoIP 和其他 App 通知不在支持范围。
- **关闭窗口后没有提醒**：确认程序仍在系统托盘，未点“退出程序”，并检查连接状态。

APK 使用调试签名，EXE 未签名，供个人侧载测试。构建和协议测试通过不代表已完成所有真机兼容性验证。

## 开发与构建

- `android/`：Java，JDK 17、Android SDK 35、Gradle 8.9。
- `windows-go/`：Go 蓝牙后端、托盘、图片协议和内嵌 WebView2 界面。
- `windows/`：旧 C# 客户端源码及安装脚本；当前发行 EXE 来自 Go 版本。
- `.github/workflows/build-apps.yml`：构建两端；推送 `v*` 标签自动发布测试版。

Windows 在仓库根目录执行：

```powershell
go test ./windows-go
go build -trimpath -ldflags="-H=windowsgui -s -w" -o BluetoothPhoneAlerts.exe ./windows-go
```

跨平台构建设置 `GOOS=windows GOARCH=amd64 CGO_ENABLED=0`。Android 使用 Android Studio 打开 `android/` 或执行 `gradle -p android :app:assembleDebug --no-daemon`。Actions Artifacts 下载可能要求登录，公开下载请使用 Releases 附件。

## 通讯协议

Android 为 RFCOMM 服务端，Windows 为客户端，UUID：`bac20533-10ac-4c14-b8e9-45a1b29feb17`。UTF-8 JSON 每行一条，电脑回复 `ACK|<id>\n`。

事件类型：`sms`、`call`、`test`、`status`、`text`。图片使用 `image_start`（大小、文件名、SHA-256）、`image_chunk`（顺序编号、最多 4096 字节的 Base64 数据）和 `image_end`；每帧等待回执，最大 5 MB，断线后重新发送。
