# BluetoothPhoneAlerts — Windows + Android 纯蓝牙来电 / 短信提醒

> **v0.4.1 测试版**。只需在手机安装 APK、电脑运行 EXE，两端蓝牙配对即可。**用户运行时不必安装 Python、Node.js、Java、.NET 或 Android Studio**。公开下载请到 [Releases](https://github.com/2-3jh/BluetoothPhoneAlerts/releases)；无需登录 GitHub。
>
> **项目尚未在真实手机/Windows 设备上完成端到端验证**。APK 为调试签名，EXE 未做代码签名；这是供自行测试的版本，不是正式商店发行版。

## 2026-10-19：已编译 Windows x64 客户端（Go 原生程序）

本次新增 `windows-go/main_windows.go`，在有 Go 编译器的 Linux 环境中成功交叉编译为真正的 Windows x64 **PE 可执行文件**，可直接双击运行；不需要 .NET/Node/Java/Python。使用同一 RFCOMM UUID / JSON-ACK 协议，与安卓 Java 客户端兼容（协议层面）。

可单独下载编译后的 Windows 可执行文件包（文件名 `BluetoothPhoneAlerts.exe`）；测试方法：直接双击 → 点击 **「测试弹窗」**；手机 APK 准备好后，再配对手机并点击「连接 / 自动重连」。`install-windows.bat` 可选地设置开机启动。关闭窗口隐藏到托盘，点「退出程序」真正退出。原 `windows/BluetoothPhoneAlerts.Win` **C# 源码仍保留**，但此次编译出来的 EXE 是 **Go 版本，不是 C# 版本**；GitHub Actions 已改为用 Go 构建 EXE。

**构建状态：Android APK 与 Windows EXE 已通过 GitHub Actions 构建，但尚未完成真实设备的双端联调。**

## 2026-10-22：Windows 界面改为 Web 前端

Windows 主界面现由内嵌的 **HTML / CSS / JavaScript** 渲染，使用 WebView2 呈现液态玻璃风格、设备卡片、状态栏、时间线记录和前端提醒卡片；Go 继续负责蓝牙、托盘、自动重连和本地历史。页面资源直接打包在 EXE 内，不访问远程网页。Windows 11 自带 WebView2 Runtime；精简版 Windows 10 若缺少该组件，需要安装微软的 WebView2 Runtime。

---

## 普通用户：拿到 APK 和 EXE 后怎么用（最简单）

1. Windows 蓝牙设置 ↔ 安卓手机蓝牙设置，先完成配对。
2. 安卓安装 `app-debug.apk`，打开并允许 **附近设备、短信、电话状态、通知** 权限，按 **开始蓝牙服务**。安卓可能提示“允许安装未知来源应用”。
3. Windows 解压 `Windows-EXE.zip`，直接双击 `BluetoothPhoneAlerts.exe`（绿色免安装版）；**或者**双击 `install-windows.bat` 安装到当前用户目录并设置开机后台启动（可选）。
4. 第一次在 Windows 程序中选择已配对手机 → **开始连接**。下一次启动程序将自动选中上一次手机并尝试重新连接。
5. 手机上分别点击 **模拟新短信** / **模拟来电**，确认 Windows 出现气泡提醒；实际收到短信、蜂窝来电时也会尝试通知。

无需 Wi-Fi、同一个局域网、外网、登录帐号、云服务。安卓端必须保持前台服务运行；不同厂商省电策略会影响后台持续性。

## 没有现成安装包？用 GitHub 免费云编译

直接下载已发布的安装包请访问 [Releases](https://github.com/2-3jh/BluetoothPhoneAlerts/releases)，无需 GitHub 账号。以下步骤仅供想自行重新编译源码的人使用。

1. GitHub 新建一个仓库，把压缩包内的 **`BluetoothPhoneAlerts/` 文件夹里的所有内容** 放到仓库根目录，包含隐藏的 `.github/workflows/build-apps.yml` 文件（关键）。不要把外层文件夹直接套到仓库根目录，否则路径不对。
2. 在仓库 **Actions → Build Android APK and Windows EXE → Run workflow** 手动触发；也可以通过推送代码到 `main` / `master` 自动触发。
3. 等两个构建任务结束且显示绿色成功标记后，进入该次运行的 **Artifacts** 下载 **`Android-APK`** 和 **`Windows-EXE`**，分别解压。
4. 安卓安装 APK；Windows 直接打开 EXE。编译产物默认在 Actions 保留 **14 天**；以后可重新执行构建。

⚠️ APK 为 Android 调试签名构建，适合**自己安装测试**，不适合上架发布。Windows EXE 也是没有付费代码签名的个人构建，Windows SmartScreen 可能提示未知发布者；仅在确认是自己信任的源代码构建时运行。

## 技术怎么选？

- 安卓端：**Java**，负责接收系统短信/来电事件并暴露蓝牙 Classic RFCOMM 服务。
- Windows 端：**Go 蓝牙后端 + 内嵌 HTML/CSS/JavaScript 前端**，通过系统 WebView2 显示；旧 C#/.NET 8 WinForms 客户端源码作为备选保留。
- Windows 端可以改成 **Python**，但蓝牙 RFCOMM 服务发现通常需要额外库（如 PyBluez2），配置和打包比现有版本繁琐。
- Windows 端可以改成 **Node.js**，通过 npm 安装 RFCOMM 库（如 `bluetooth-serial-port`），但有原生二进制依赖，可能触发 C++ 编译；而且需适配本项目自定义 UUID 的服务发现。
- **npm 是 Node.js 的包管理器，不是一种服务端编程语言**。
- Java 桌面端同样可以实现，但 Windows 蓝牙访问往往需要第三方桥接库，并不一定比 C# 简单。

## 项目结构

- `android/` — Android Java 项目。
- `windows-go/main_windows.go` — Windows 原生 Go GUI 源码（可直接交叉编译，不需要第三方库）。
- `windows/BluetoothPhoneAlerts.Win/` — 原 Windows C# GUI 源码（备选）。
- `windows/build-windows.bat` — Windows 开发者：本地发布单文件 EXE（需 .NET SDK）。
- `windows/install-windows.bat` — 普通用户：把已构建的 EXE 复制到 `%LOCALAPPDATA%`，创建用户开机启动快捷方式。
- `.github/workflows/build-apps.yml` — 无需自己安装编译器的 GitHub 自动构建。

---

## 通讯协议

Android 是 RFCOMM **服务端**，Windows 是 RFCOMM **客户端**。手机接受来自已经配对的设备连接。每个 JSON 一行，UTF-8，以 `\n` 结尾：

```json
{"id":"f4715768-92f4-4689-9081-e564cd4f2dca","type":"call"}
```

`type` 为 `sms` / `call` / `test`；Windows 回发 `ACK|<id>\n`。Android 在 ACK 后从本地短队列移除；断线会重传。Windows 本次运行保留最近 120 个消息 id 用于去重。

## 常见问题

- **Windows 端一直连接失败**：确认两端已配对、Android 已点击“开始蓝牙服务”、蓝牙已开启；可在系统设置里删除配对再重新配对；某些蓝牙驱动不兼容自定义 RFCOMM 服务。
- **模拟提醒正常，真实短信/来电不触发**：检查短信、电话权限以及手机系统对广播或后台的限制；双卡、VoIP 通话（微信/QQ 等）不属于普通蜂窝电话 `PHONE_STATE`，不监听这些 App 通知。
- **通知没有弹窗**：先点电脑端“测试提醒”，确认右下角卡片能出现；再检查蓝牙连接状态与程序日志。系统托盘气泡可能受 Windows 通知设置影响，但卡片不依赖该气泡。
- **后台服务停止**：需重新打开 Android App 点击开始。首次版本没有开机自动启动，也不保证厂家节电策略杀进程后的自动恢复。
- **Google Play 发布限制**：`RECEIVE_SMS` 属于受严格限制的权限，Google Play 上架需满足适用政策并审核；这里用于个人侧载验证，不代表可直接上架。
- **电话内容/电话号码**：授权通话记录与联系人后可显示来电号码和联系人；授权短信后可显示短信内容。电脑端通知历史保留最多 30 条、最长 7 天，可在主窗口清空；记录以明文保存在本机，请注意设备隐私。
- **隐私与安全**：数据只发给当前已配对的蓝牙设备，蓝牙通道使用 Android secure RFCOMM；手机队列最多保留 100 条简单事件，不记录内容。请仅安装到自己的设备上。

## 待改进

- 添加开机启动和断线状态 UI（手机端）
- 记录 Windows 最近使用的蓝牙设备，并允许自动连接
- 如需兼容更严格后台限制，迁移为 Companion Device Manager 关联设备方案
- 真机兼容性测试、签名发布 APK / EXE、改进 Windows 11 Toast 样式

## 开发者本地构建（可选）

- Android Studio 打开 `android/`；首次运行 `android/setup-wrapper.bat` 补齐官方 Gradle wrapper，然后 Build → Build APK(s)。Android SDK API 35、JDK 17。
- Windows Go 版：安装 Go 1.23，执行 `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-H=windowsgui -s -w" -o BluetoothPhoneAlerts.exe windows-go/main_windows.go`；已在当前 Linux 环境实际交叉编译。
- 旧 Windows C# 版：安装 .NET 8 SDK，双击 `windows/build-windows.bat`（仅供备选）。

> 注意：`install-windows.bat` 只设置 **Windows 端** 开机自启。**手机端不会在重启后无条件自动启动**，首次版本请手动启动安卓服务。手机收不到通知时，检查短信/电话权限、蓝牙范围、系统后台耗电限制和 Windows 免打扰模式。
