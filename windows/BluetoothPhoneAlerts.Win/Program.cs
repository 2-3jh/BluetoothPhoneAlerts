using System;
using System.Collections.Generic;
using System.Drawing;
using System.IO;
using System.Linq;
using System.Text;
using System.Text.Json;
using System.Threading;
using System.Threading.Tasks;
using System.Windows.Forms;
using InTheHand.Net.Bluetooth;
using InTheHand.Net.Sockets;

namespace BluetoothPhoneAlerts.Win;

internal static class Program
{
    [STAThread]
    private static void Main(string[] args)
    {
        ApplicationConfiguration.Initialize();
        Application.Run(new MainWindow(args.Contains("--background", StringComparer.OrdinalIgnoreCase)));
    }
}

internal sealed class MainWindow : Form
{
    // Same service UUID as Android Protocol.java
    private static readonly Guid ServiceId = new("bac20533-10ac-4c14-b8e9-45a1b29feb17");
    private readonly ComboBox devices = new() { DropDownStyle = ComboBoxStyle.DropDownList, Width = 360 };
    private readonly Button refresh = new() { Text = "刷新已配对手机", Width = 145 };
    private readonly Button start = new() { Text = "开始连接", Width = 105 };
    private readonly Button stop = new() { Text = "断开连接", Width = 105, Enabled = false };
    private readonly Label status = new() { Text = "尚未连接", AutoSize = true, Margin = new Padding(5, 14, 5, 10) };
    private readonly TextBox log = new() {
        Multiline = true, ReadOnly = true, Dock = DockStyle.Fill, ScrollBars = ScrollBars.Vertical
    };
    private readonly NotifyIcon tray = new() {
        Icon = SystemIcons.Information, Text = "手机蓝牙来电短信提醒", Visible = true
    };
    private readonly object clientLock = new();
    private BluetoothClient? currentClient;
    private CancellationTokenSource? connectionCancel;
    private readonly HashSet<string> recentIds = new();
    private readonly Queue<string> recentOrder = new();
    private bool quitting = false;
    private readonly bool startHidden;
    private static readonly string SavedDevicePath = Path.Combine(
        Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),
        "BluetoothPhoneAlerts", "paired-device.txt");

    public MainWindow(bool startHidden = false)
    {
        this.startHidden = startHidden;
        Text = "蓝牙手机通知（纯蓝牙）";
        Width = 585;
        Height = 405;
        StartPosition = FormStartPosition.CenterScreen;
        MinimumSize = new Size(530, 300);

        var top = new FlowLayoutPanel { Dock = DockStyle.Top, Height = 120, WrapContents = true,
            FlowDirection = FlowDirection.LeftToRight, Padding = new Padding(12) };
        top.Controls.Add(new Label { Text = "已配对设备", AutoSize = true, Margin = new Padding(5, 10, 5, 8) });
        top.Controls.Add(devices);
        top.SetFlowBreak(devices, true);
        top.Controls.Add(refresh);
        top.Controls.Add(start);
        top.Controls.Add(stop);
        top.SetFlowBreak(stop, true);
        top.Controls.Add(status);
        Controls.Add(log);
        Controls.Add(top);

        refresh.Click += (_, _) => RefreshPairedDevices();
        start.Click += (_, _) => StartConnection();
        stop.Click += (_, _) => StopConnection();
        tray.DoubleClick += (_, _) => ShowWindow();
        var menu = new ContextMenuStrip();
        menu.Items.Add("打开", null, (_, _) => ShowWindow());
        menu.Items.Add("退出", null, (_, _) => Quit());
        tray.ContextMenuStrip = menu;
        FormClosing += (_, e) => {
            if (!quitting) { e.Cancel = true; Hide(); }
        };
        Shown += (_, _) => { RefreshPairedDevices(); TryConnectSavedDevice(); if (this.startHidden) Hide(); };
        Log("先在 Windows 蓝牙设置中与安卓手机配对，并在手机端开始蓝牙服务。");
        Log("关闭窗口只会最小化到托盘；右键托盘图标可退出。");
    }

    private void RefreshPairedDevices()
    {
        try
        {
            string? previous = (devices.SelectedItem as PhoneDevice)?.Info.DeviceAddress.ToString();
            using var bluetooth = new BluetoothClient();
            var paired = bluetooth.PairedDevices.OrderBy(d => d.DeviceName).ToArray();
            devices.Items.Clear();
            foreach (var device in paired) devices.Items.Add(new PhoneDevice(device));
            if (devices.Items.Count == 0) Log("没有检测到已配对设备。请先在系统蓝牙设置中配对。");
            int index = Array.FindIndex(paired, d => d.DeviceAddress.ToString() == previous);
            if (devices.Items.Count > 0) devices.SelectedIndex = index >= 0 ? index : 0;
            Log($"已读取 {paired.Length} 个已配对设备。");
        }
        catch (Exception ex) { Log("蓝牙设备读取失败：" + ex.Message); }
    }

    private void TryConnectSavedDevice()
    {
        try {
            if (!File.Exists(SavedDevicePath)) return;
            var address = File.ReadAllText(SavedDevicePath).Trim();
            for (int i = 0; i < devices.Items.Count; i++) {
                if (devices.Items[i] is PhoneDevice phone &&
                    string.Equals(phone.Info.DeviceAddress.ToString(), address,
                        StringComparison.OrdinalIgnoreCase)) {
                    devices.SelectedIndex = i;
                    Log("发现上次连接的手机，正在自动重连…");
                    StartConnection();
                    return;
                }
            }
            Log("上次连接的手机没有出现在已配对列表中，请重新配对。");
        } catch (Exception ex) { Log("自动连接未启动：" + ex.Message); }
    }

    private void StartConnection()
    {
        if (connectionCancel != null) return;
        if (devices.SelectedItem is not PhoneDevice phone) {
            MessageBox.Show("请先与手机配对，并选择设备。", "提示"); return;
        }
        // Remember only the Bluetooth device address (no SMS text, numbers, or contacts).
        try {
            Directory.CreateDirectory(Path.GetDirectoryName(SavedDevicePath)!);
            File.WriteAllText(SavedDevicePath, phone.Info.DeviceAddress.ToString());
        } catch (Exception ex) { Log("无法记住设备，下次需手动选择：" + ex.Message); }
        connectionCancel = new CancellationTokenSource();
        start.Enabled = false;
        devices.Enabled = false;
        refresh.Enabled = false;
        stop.Enabled = true;
        var token = connectionCancel.Token;
        Log("正在连接：" + phone.Info.DeviceName);
        _ = Task.Run(() => ConnectionLoop(phone.Info, token));
    }

    private void ConnectionLoop(BluetoothDeviceInfo phone, CancellationToken token)
    {
        while (!token.IsCancellationRequested)
        {
            try
            {
                SafeUi(() => status.Text = "蓝牙连接中…");
                using var client = new BluetoothClient();
                lock (clientLock) currentClient = client;
                client.Connect(phone.DeviceAddress, ServiceId);
                SafeUi(() => { status.Text = "已连接：" + phone.DeviceName; Log("蓝牙连接成功"); });
                using var stream = client.GetStream();
                using var reader = new StreamReader(stream, Encoding.UTF8, false, 1024, leaveOpen: true);
                using var writer = new StreamWriter(stream, new UTF8Encoding(false), 1024, leaveOpen: true) {
                    AutoFlush = true
                };
                string? line;
                while (!token.IsCancellationRequested && (line = reader.ReadLine()) != null)
                {
                    try {
                        using var json = JsonDocument.Parse(line);
                        var root = json.RootElement;
                        if (!root.TryGetProperty("id", out var idElement) ||
                            !root.TryGetProperty("type", out var typeElement)) continue;
                        string? id = idElement.GetString();
                        string? kind = typeElement.GetString();
                        if (string.IsNullOrWhiteSpace(id) || id.Length > 100 ||
                            (kind != "sms" && kind != "call" && kind != "test")) continue;
                        // ACK before displaying so a duplicate on reconnect won't alarm twice.
                        writer.WriteLine("ACK|" + id);
                        if (RememberId(id)) SafeUi(() => DisplayAlert(kind!));
                    } catch (JsonException) { SafeUi(() => Log("忽略无法解析的消息")); }
                }
            }
            catch (Exception ex) {
                if (!token.IsCancellationRequested)
                    SafeUi(() => Log("连接中断：" + ex.Message));
            }
            finally
            {
                lock (clientLock) currentClient = null;
                SafeUi(() => status.Text = "未连接，将自动重试");
            }
            if (token.WaitHandle.WaitOne(3000)) break;
        }
        SafeUi(() => {
            status.Text = "已停止连接";
            Log("监听已停止");
            connectionCancel?.Dispose();
            connectionCancel = null;
            devices.Enabled = true;
            refresh.Enabled = true;
            start.Enabled = true;
            stop.Enabled = false;
        });
    }

    private bool RememberId(string id)
    {
        lock (recentIds) {
            if (!recentIds.Add(id)) return false;
            recentOrder.Enqueue(id);
            if (recentOrder.Count > 120) recentIds.Remove(recentOrder.Dequeue());
            return true;
        }
    }

    private void DisplayAlert(string kind)
    {
        string title = kind switch {
            "sms" => "手机收到新短信",
            "call" => "手机有来电",
            _ => "蓝牙测试通知"
        };
        string message = kind == "call" ? "请查看手机上的来电" :
            kind == "sms" ? "请打开手机查看短信" : "手机与电脑蓝牙连接正常";
        Log(title);
        tray.ShowBalloonTip(4000, title, message, ToolTipIcon.Info);
    }

    private void StopConnection()
    {
        connectionCancel?.Cancel();
        lock (clientLock) {
            try { currentClient?.Close(); } catch { /* cancellation */ }
        }
        status.Text = "正在断开…";
    }

    private void SafeUi(Action action)
    {
        if (!IsDisposed && IsHandleCreated) {
            try { BeginInvoke(action); } catch (InvalidOperationException) { }
        }
    }

    private void Log(string message)
    {
        if (log.IsDisposed) return;
        log.AppendText($"[{DateTime.Now:HH:mm:ss}] {message}{Environment.NewLine}");
    }

    private void ShowWindow() { Show(); WindowState = FormWindowState.Normal; Activate(); }
    private void Quit()
    {
        quitting = true;
        StopConnection();
        tray.Visible = false;
        tray.Dispose();
        Close();
    }

    private sealed class PhoneDevice
    {
        public BluetoothDeviceInfo Info { get; }
        public PhoneDevice(BluetoothDeviceInfo info) => Info = info;
        public override string ToString() => $"{Info.DeviceName} ({Info.DeviceAddress})";
    }
}
