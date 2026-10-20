package com.example.btphonealerts;

import android.Manifest;
import android.app.Activity;
import android.bluetooth.BluetoothAdapter;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.os.Build;
import android.os.Bundle;
import android.view.View;
import android.widget.Button;
import android.widget.LinearLayout;
import android.widget.TextView;
import android.widget.Toast;
import java.util.ArrayList;
import java.util.List;

public final class MainActivity extends Activity {
    private static final int REQUEST_PERMS = 123;
    private TextView hint;

    @Override public void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        LinearLayout root = new LinearLayout(this);
        root.setOrientation(LinearLayout.VERTICAL);
        root.setPadding(50, 65, 50, 30);
        TextView title = new TextView(this);
        title.setText("蓝牙来电 / 短信提醒");
        title.setTextSize(23);
        root.addView(title);
        hint = new TextView(this);
        hint.setText("1. 在系统蓝牙设置中先配对 Windows 电脑\n2. 手机打开蓝牙\n3. 授权后点击开始服务\n4. Windows 端选择已配对手机并连接\n\n仅发送事件类型，不读取短信内容和电话号码。");
        hint.setTextSize(16);
        hint.setPadding(0, 24, 0, 24);
        root.addView(hint);
        Button start = addButton(root, "开始蓝牙服务", v -> requestAndStart());
        Button stop = addButton(root, "停止服务", v -> {
            stopService(new Intent(this, BluetoothServerService.class));
            Toast.makeText(this, "已请求停止", Toast.LENGTH_SHORT).show();
        });
        Button testSms = addButton(root, "模拟新短信（测试）", v -> {
            EventStore.add(this, "sms");
            Toast.makeText(this, "测试短信已入队", Toast.LENGTH_SHORT).show();
        });
        Button testCall = addButton(root, "模拟来电（测试）", v -> {
            EventStore.add(this, "call");
            Toast.makeText(this, "测试来电已入队", Toast.LENGTH_SHORT).show();
        });
        setContentView(root);
    }

    private Button addButton(LinearLayout parent, String label, View.OnClickListener click) {
        Button b = new Button(this);
        b.setText(label);
        b.setOnClickListener(click);
        parent.addView(b);
        return b;
    }

    private boolean granted(String permission) {
        return Build.VERSION.SDK_INT < 23 || checkSelfPermission(permission) == PackageManager.PERMISSION_GRANTED;
    }

    private void requestAndStart() {
        List<String> missing = new ArrayList<>();
        if (Build.VERSION.SDK_INT >= 31 && !granted(Manifest.permission.BLUETOOTH_CONNECT))
            missing.add(Manifest.permission.BLUETOOTH_CONNECT);
        if (!granted(Manifest.permission.RECEIVE_SMS)) missing.add(Manifest.permission.RECEIVE_SMS);
        if (!granted(Manifest.permission.READ_PHONE_STATE)) missing.add(Manifest.permission.READ_PHONE_STATE);
        if (Build.VERSION.SDK_INT >= 33 && !granted(Manifest.permission.POST_NOTIFICATIONS))
            missing.add(Manifest.permission.POST_NOTIFICATIONS);
        if (!missing.isEmpty()) {
            requestPermissions(missing.toArray(new String[0]), REQUEST_PERMS);
        } else startNow();
    }

    @Override public void onRequestPermissionsResult(int requestCode, String[] permissions, int[] results) {
        super.onRequestPermissionsResult(requestCode, permissions, results);
        if (requestCode == REQUEST_PERMS) startNow();
    }

    private void startNow() {
        if (Build.VERSION.SDK_INT >= 31 && !granted(Manifest.permission.BLUETOOTH_CONNECT)) {
            Toast.makeText(this, "必须授权附近设备（蓝牙）", Toast.LENGTH_LONG).show(); return;
        }
        if (!granted(Manifest.permission.RECEIVE_SMS) || !granted(Manifest.permission.READ_PHONE_STATE)) {
            Toast.makeText(this, "要监听真实事件需授权短信和电话状态", Toast.LENGTH_LONG).show(); return;
        }
        BluetoothAdapter adapter = BluetoothAdapter.getDefaultAdapter();
        if (adapter == null) { Toast.makeText(this, "设备不支持蓝牙", Toast.LENGTH_LONG).show(); return; }
        if (!adapter.isEnabled()) {
            Toast.makeText(this, "请先在系统设置中打开蓝牙", Toast.LENGTH_LONG).show(); return;
        }
        try {
            startForegroundService(new Intent(this, BluetoothServerService.class));
            Toast.makeText(this, "服务已启动，请用 Windows 连接", Toast.LENGTH_LONG).show();
        } catch (Exception e) {
            Toast.makeText(this, "启动失败：" + e.getMessage(), Toast.LENGTH_LONG).show();
        }
    }
}
