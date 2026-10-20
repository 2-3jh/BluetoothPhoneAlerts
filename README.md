# BluetoothPhoneAlerts — Windows + Android 纯蓝牙来电 / 短信提醒

> **简化安装版（v0.2 源码）**。只需在手机安装 APK、电脑运行 EXE，两端蓝牙配对即可。**用户运行时不必安装 Python、Node.js、Java、.NET 或 Android Studio**。本源码包没有预先编译好的 APK / EXE；可以通过下文的 GitHub Actions 自动生成，不需要在自己电脑安装开发工具。
>
> **项目未在真实手机/Windows 设备上端到端验证**。GitHub Actions 是否可一次构建成功取决于实际构建环境；这不是经测试的正式发行版。

## 2026-10-19：已编译 Windows x64 客户端（Go 原生程序）

本次新增 `windows-go/main_windows.go`，在有 Go 编译器的 Linux 环境中成功交叉编译为真正的 Windows x64 **PE 可执行文件**，可直接双击运行；不需要 .NET/Node/Java/Python。使用同一 RFCOMM UUID / JSON-ACK 协议，与安卓 Java 客户端兼容（协议层面）。

可单独下载编译后的 Windows 可执行文件包（文件名 `BluetoothPhoneAlerts.exe`）；测试方法：直接双击 → 点击 **「测试弹窗」**；手机 APK 准备好后，再配对手机并点击「连接 / 自动重连」。`install-windows.bat` 可选地设置开机启动。关闭窗口隐藏到托盘，点「退出程序」真正退出。原 `windows/BluetoothPhoneAlerts.Win` **C# 源码仍保留**，但此次编译出来的 EXE 是 **Go 版本，不是 C# 版本**；GitHub Actions 已改为用 Go 构建 EXE。

**构建状态：Windows Go 客户端成功交叉编译，但尚未经过 Windows 11 真机运行和蓝牙连接测试；Android APK 尚未编译。** 当前机器没有 Android SDK 或 Gradle，且不允许下载 SDK；Android APK 仍需使用源码内的 GitHub Actions 或本地 Android Studio 构建。请勿将目前状态理解为双端已通过联调。

---

## 普通用户：拿到 APK 和 EXE 后怎么用（最简单）

1. Windows 蓝牙设置 ↔ 安卓手机蓝牙设置，先完成配对。
2. 安卓安装 `app-debug.apk`，打开并允许 **附近设备、短信、电话状态、通知** 权限，按 **开始蓝牙服务**。安卓可能提示“允许安装未知来源应用”。
3. Windows 解压 `Windows-EXE.zip`，直接双击 `BluetoothPhoneAlerts.exe`（绿色免安装版）；**或者**双击 `install-windows.bat` 安装到当前用户目录并设置开机后台启动（可选）。
4. 第一次在 Windows 程序中选择已配对手机 → **开始连接**。下一次启动程序将自动选中上一次手机并尝试重新连接。
5. 手机上分别点击 **模拟新短信** / **模拟来电**，确认 Windows 出现气泡提醒；实际收到短信、蜂窝来电时也会尝试通知。

无需 Wi-Fi、同一个局域网、外网、登录帐号、云服务。安卓端必须保持前台服务运行；不同厂商省电策略会影响后台持续性。

## 没有现成安装包？用 GitHub 免费云编译

不要求在个人电脑安装 Android Studio / .NET SDK / Java / Gradle，但需要 GitHub 账号，并上传此源码到自己建立的仓库。

1. GitHub 新建一个仓库，把压缩包内的 **`BluetoothPhoneAlerts/` 文件夹里的所有内容** 放到仓库根目录，包含隐藏的 `.github/workflows/build-apps.yml` 文件（关键）。不要把外层文件夹直接套到仓库根目录，否则路径不对。
2. 在仓库 **Actions → Build Android APK and Windows EXE → Run workflow** 手动触发；也可以通过推送代码到 `main` / `master` 自动触发。
3. 等两个构建任务结束且显示绿色成功标记后，进入该次运行的 **Artifacts** 下载 **`Android-APK`** 和 **`Windows-EXE`**，分别解压。
4. 安卓安装 APK；Windows 直接打开 EXE。编译产物默认在 Actions 保留 **14 天**；以后可重新执行构建。

⚠️ APK 为 Android 调试签名构建，适合**自己安装测试**，不适合上架发布。Windows EXE 也是没有付费代码签名的个人构建，Windows SmartScreen 可能提示未知发布者；仅在确认是自己信任的源代码构建时运行。

## 技术怎么选？

- 安卓端：**Java**，负责接收系统短信/来电事件并暴露蓝牙 Classic RFCOMM 服务。
- Windows 端：**优先使用已交叉编译的 Go 原生 Windows EXE**，无需依赖运行时。旧 C#/.NET 8 WinForms + 32feet.NET 客户端源码作为备选保留；GH Actions 默认编译 Go 版本。
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
- **通知没有弹窗**：确认 Windows 设置 → 系统 → 通知 已开启，且未被免打扰模式/专注模式阻止；程序日志中可以确认事件是否抵达。WinForms `NotifyIcon` 气泡提示的显示方式由 Windows 决定。
- **后台服务停止**：需重新打开 Android App 点击开始。首次版本没有开机自动启动，也不保证厂家节电策略杀进程后的自动恢复。
- **Google Play 发布限制**：`RECEIVE_SMS` 属于受严格限制的权限，Google Play 上架需满足适用政策并审核；这里用于个人侧载验证，不代表可直接上架。
- **电话内容/电话号码**：为了简单和最小权限，不请求 `READ_CALL_LOG`，不显示电话号码，也不存储短信内容。
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
