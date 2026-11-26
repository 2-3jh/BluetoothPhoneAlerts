package com.example.btphonealerts;

import android.Manifest;
import android.app.Activity;
import android.bluetooth.BluetoothAdapter;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.database.Cursor;
import android.graphics.Color;
import android.graphics.Typeface;
import android.graphics.drawable.GradientDrawable;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.provider.OpenableColumns;
import android.view.Gravity;
import android.view.View;
import android.widget.Button;
import android.widget.EditText;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;
import android.widget.Toast;
import java.util.ArrayList;
import java.util.List;
import java.io.File;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.io.OutputStream;

public final class MainActivity extends Activity {
    private static final int REQUEST_PERMS = 123;
    private static final int REQUEST_IMAGE = 124;
    private static final long MAX_IMAGE_BYTES = 5L * 1024 * 1024;
    private static final int BLUE = Color.rgb(38, 99, 235);
    private static final int NAVY = Color.rgb(20, 35, 70);
    private static final int INK = Color.rgb(30, 41, 59);
    private static final int MUTED = Color.rgb(100, 116, 139);
    private TextView statusTitle;
    private TextView statusDetail;
    private Button startButton;
    private Button stopButton;
    private EditText transferText;
    private final Runnable statusRefresh = new Runnable() {
        @Override public void run() {
            refreshStatus();
            statusTitle.postDelayed(this, 1000L);
        }
    };

    @Override public void onCreate(Bundle state) {
        super.onCreate(state);
        getWindow().setStatusBarColor(NAVY);
        getWindow().setNavigationBarColor(Color.rgb(241, 245, 249));

        ScrollView scroll = new ScrollView(this);
        scroll.setFillViewport(true);
        scroll.setBackgroundColor(Color.rgb(244, 247, 252));
        LinearLayout page = column();
        page.setPadding(dp(20), dp(24), dp(20), dp(28));
        scroll.addView(page, matchWrap());
        page.addView(hero(), matchWrap());
        page.addView(space(16));
        page.addView(statusCard(), matchWrap());
        page.addView(space(16));
        page.addView(stepsCard(), matchWrap());
        page.addView(space(18));
        page.addView(actions(), matchWrap());
        page.addView(space(14));
        page.addView(transferCard(), matchWrap());
        page.addView(space(14));
        page.addView(privacyNote(), matchWrap());
        setContentView(scroll);
        refreshStatus();
    }

    @Override protected void onResume() {
        super.onResume();
        statusTitle.removeCallbacks(statusRefresh);
        statusTitle.post(statusRefresh);
    }

    @Override protected void onPause() {
        statusTitle.removeCallbacks(statusRefresh);
        super.onPause();
    }

    private View hero() {
        LinearLayout view = column();
        view.setPadding(dp(22), dp(22), dp(22), dp(22));
        GradientDrawable background = new GradientDrawable(
                GradientDrawable.Orientation.TL_BR,
                new int[]{Color.rgb(28, 53, 112), Color.rgb(50, 113, 238)});
        background.setCornerRadius(dp(24));
        view.setBackground(background);
        view.setElevation(dp(4));
        TextView eyebrow = text("PHONE LINK", 12, Color.rgb(191, 219, 254));
        eyebrow.setTypeface(Typeface.DEFAULT_BOLD);
        eyebrow.setLetterSpacing(0.16f);
        view.addView(eyebrow);
        TextView title = text("蓝牙提醒助手", 28, Color.WHITE);
        title.setTypeface(Typeface.DEFAULT_BOLD);
        title.setPadding(0, dp(8), 0, dp(8));
        view.addView(title);
        TextView subtitle = text("让电脑及时收到手机来电与短信提醒", 15, Color.rgb(219, 234, 254));
        subtitle.setLineSpacing(0, 1.15f);
        view.addView(subtitle);
        return view;
    }

    private View statusCard() {
        LinearLayout card = card();
        TextView label = text("服务状态", 13, MUTED);
        label.setTypeface(Typeface.DEFAULT_BOLD);
        card.addView(label);
        statusTitle = text("", 22, INK);
        statusTitle.setTypeface(Typeface.DEFAULT_BOLD);
        statusTitle.setPadding(0, dp(9), 0, dp(4));
        card.addView(statusTitle);
        statusDetail = text("", 14, MUTED);
        statusDetail.setLineSpacing(0, 1.2f);
        card.addView(statusDetail);
        return card;
    }

    private View stepsCard() {
        LinearLayout card = card();
        TextView heading = text("连接电脑", 18, INK);
        heading.setTypeface(Typeface.DEFAULT_BOLD);
        card.addView(heading);
        card.addView(step("1", "在系统蓝牙设置中与 Windows 电脑配对"));
        card.addView(step("2", "打开手机蓝牙并授予所需权限"));
        card.addView(step("3", "点击开始服务，再从电脑端发起连接"));
        return card;
    }

    private View step(String number, String copy) {
        LinearLayout row = new LinearLayout(this);
        row.setGravity(Gravity.CENTER_VERTICAL);
        row.setPadding(0, dp(15), 0, 0);
        TextView badge = text(number, 13, BLUE);
        badge.setTypeface(Typeface.DEFAULT_BOLD);
        badge.setGravity(Gravity.CENTER);
        badge.setBackground(rounded(Color.rgb(239, 246, 255), 12));
        row.addView(badge, new LinearLayout.LayoutParams(dp(30), dp(30)));
        TextView description = text(copy, 14, INK);
        description.setPadding(dp(12), 0, 0, 0);
        row.addView(description, new LinearLayout.LayoutParams(0, -2, 1));
        return row;
    }

    private View actions() {
        LinearLayout group = column();
        startButton = button("开始蓝牙服务", BLUE, Color.WHITE, v -> requestAndStart());
        group.addView(startButton, new LinearLayout.LayoutParams(-1, dp(54)));
        group.addView(space(10));
        stopButton = button("停止服务", Color.rgb(226, 232, 240), INK, v -> stopNow());
        group.addView(stopButton, new LinearLayout.LayoutParams(-1, dp(50)));
        group.addView(space(14));
        LinearLayout tests = new LinearLayout(this);
        Button sms = button("测试短信", Color.WHITE, BLUE, v -> enqueueTest("sms", "测试短信已发送"));
        Button call = button("测试来电", Color.WHITE, BLUE, v -> enqueueTest("call", "测试来电已发送"));
        tests.addView(sms, new LinearLayout.LayoutParams(0, dp(48), 1));
        LinearLayout.LayoutParams second = new LinearLayout.LayoutParams(0, dp(48), 1);
        second.setMarginStart(dp(10));
        tests.addView(call, second);
        group.addView(tests, matchWrap());
        return group;
    }

    private View privacyNote() {
        TextView note = text("隐私保护：短信、来电和主动发送的文字或图片只通过已配对蓝牙传到电脑，不联网。", 13, MUTED);
        note.setLineSpacing(0, 1.2f);
        note.setPadding(dp(14), dp(12), dp(14), dp(12));
        note.setBackground(rounded(Color.rgb(232, 245, 239), 14));
        return note;
    }

    private View transferCard() {
        LinearLayout card = card();
        TextView heading = text("发送到电脑", 18, INK);
        heading.setTypeface(Typeface.DEFAULT_BOLD);
        card.addView(heading);
        TextView hint = text("可发送短文字或多张图片（一次最多 20 张，每张不超过 5 MB）；未连接时排队等待。", 13, MUTED);
        hint.setPadding(0, dp(6), 0, dp(12));
        card.addView(hint);
        transferText = new EditText(this);
        transferText.setHint("输入要发送的文字");
        transferText.setMinLines(2);
        transferText.setMaxLines(5);
        transferText.setTextSize(15);
        card.addView(transferText, matchWrap());
        card.addView(space(10));
        card.addView(button("发送文字", BLUE, Color.WHITE, v -> sendText()),
                new LinearLayout.LayoutParams(-1, dp(48)));
        card.addView(space(10));
        card.addView(button("选择图片发送", Color.rgb(239, 246, 255), BLUE, v -> pickImage()),
                new LinearLayout.LayoutParams(-1, dp(48)));
        return card;
    }

    private void sendText() {
        String value = transferText.getText().toString().trim();
        if (value.isEmpty()) {
            Toast.makeText(this, "先输入文字", Toast.LENGTH_SHORT).show();
            return;
        }
        if (value.length() > 1200) {
            Toast.makeText(this, "文字最多 1200 字", Toast.LENGTH_SHORT).show();
            return;
        }
        EventStore.add(this, "text", "手机传来文字", value);
        transferText.setText("");
        Toast.makeText(this, "文字已加入蓝牙发送队列", Toast.LENGTH_SHORT).show();
    }

    private void pickImage() {
        Intent intent = new Intent(Intent.ACTION_OPEN_DOCUMENT);
        intent.addCategory(Intent.CATEGORY_OPENABLE);
        intent.setType("image/*");
        intent.putExtra(Intent.EXTRA_ALLOW_MULTIPLE, true);
        startActivityForResult(intent, REQUEST_IMAGE);
    }

    @Override protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        super.onActivityResult(requestCode, resultCode, data);
        if (requestCode != REQUEST_IMAGE || resultCode != RESULT_OK || data == null)
            return;
        List<Uri> images = new ArrayList<>();
        if (data.getClipData() != null) {
            if (data.getClipData().getItemCount() > 20) {
                Toast.makeText(this, "一次最多选择 20 张图片", Toast.LENGTH_LONG).show();
                return;
            }
            for (int i = 0; i < data.getClipData().getItemCount(); i++) {
                Uri uri = data.getClipData().getItemAt(i).getUri();
                if (uri != null && !images.contains(uri)) images.add(uri);
            }
        } else if (data.getData() != null) images.add(data.getData());
        new Thread(() -> {
            for (Uri uri : images) queueSelectedImage(uri);
        }, "queue-images").start();
    }

    private void queueSelectedImage(Uri uri) {
        String name = "手机图片.jpg";
        try (Cursor cursor = getContentResolver().query(uri, null, null, null, null)) {
            if (cursor != null && cursor.moveToFirst()) {
                int column = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME);
                if (column >= 0 && cursor.getString(column) != null) name = cursor.getString(column);
            }
        } catch (Exception ignored) {}
        queueImage(uri, name);
    }

    private void queueImage(Uri uri, String name) {
        File image = null;
        try {
            File directory = new File(getFilesDir(), "outgoing-images");
            if (!directory.isDirectory() && !directory.mkdirs())
                throw new IllegalStateException("无法建立图片缓存目录");
            image = File.createTempFile("image-", ".bin", directory);
            long copied = 0;
            try (InputStream input = getContentResolver().openInputStream(uri);
                 OutputStream output = new FileOutputStream(image)) {
                if (input == null) throw new IllegalStateException("无法读取所选图片");
                byte[] buffer = new byte[8192];
                int count;
                while ((count = input.read(buffer)) != -1) {
                    copied += count;
                    if (copied > MAX_IMAGE_BYTES) throw new IllegalArgumentException("图片不能超过 5 MB");
                    output.write(buffer, 0, count);
                }
            }
            if (copied == 0) throw new IllegalArgumentException("图片为空");
            EventStore.addImage(getApplicationContext(), name, image);
            runOnUiThread(() -> Toast.makeText(this, "图片已加入蓝牙发送队列", Toast.LENGTH_SHORT).show());
        } catch (Exception e) {
            if (image != null) image.delete();
            String message = e.getMessage() == null ? "图片发送准备失败" : e.getMessage();
            runOnUiThread(() -> Toast.makeText(this, message, Toast.LENGTH_LONG).show());
        }
    }

    private void refreshStatus() {
        if (statusTitle == null) return;
        boolean running = BluetoothServerService.isRunning();
        boolean connected = BluetoothServerService.isConnected();
        statusTitle.setText(running ? (connected ? "已连接电脑" : "正在等待电脑连接") : "服务尚未启动");
        statusTitle.setTextColor(running ? Color.rgb(5, 150, 105) : INK);
        statusDetail.setText(running
                ? (connected ? "蓝牙连接正常，可同步提醒、电量以及发送文字和图片。" : "服务正在后台运行。保持蓝牙开启，然后在 Windows 端点击连接。")
                : "点击下方按钮启动安全的 RFCOMM 蓝牙服务。");
        startButton.setEnabled(!running);
        startButton.setAlpha(running ? 0.55f : 1f);
        stopButton.setEnabled(running);
        stopButton.setAlpha(running ? 1f : 0.55f);
    }

    private void stopNow() {
        stopService(new Intent(this, BluetoothServerService.class));
        statusTitle.postDelayed(this::refreshStatus, 200L);
        Toast.makeText(this, "蓝牙服务已停止", Toast.LENGTH_SHORT).show();
    }

    private void enqueueTest(String kind, String message) {
        EventStore.add(this, kind);
        Toast.makeText(this, message, Toast.LENGTH_SHORT).show();
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
        if (!granted(Manifest.permission.READ_CALL_LOG)) missing.add(Manifest.permission.READ_CALL_LOG);
        if (!granted(Manifest.permission.READ_CONTACTS)) missing.add(Manifest.permission.READ_CONTACTS);
        if (Build.VERSION.SDK_INT >= 33 && !granted(Manifest.permission.POST_NOTIFICATIONS))
            missing.add(Manifest.permission.POST_NOTIFICATIONS);
        if (!missing.isEmpty()) requestPermissions(missing.toArray(new String[0]), REQUEST_PERMS);
        else startNow();
    }

    @Override public void onRequestPermissionsResult(int requestCode, String[] permissions, int[] results) {
        super.onRequestPermissionsResult(requestCode, permissions, results);
        if (requestCode == REQUEST_PERMS) startNow();
    }

    private void startNow() {
        if (Build.VERSION.SDK_INT >= 31 && !granted(Manifest.permission.BLUETOOTH_CONNECT)) {
            Toast.makeText(this, "必须授权附近设备（蓝牙）", Toast.LENGTH_LONG).show(); return;
        }
        if (!granted(Manifest.permission.RECEIVE_SMS) || !granted(Manifest.permission.READ_PHONE_STATE))
            Toast.makeText(this, "短信或电话权限未授权，对应提醒暂不可用", Toast.LENGTH_LONG).show();
        BluetoothAdapter adapter = BluetoothAdapter.getDefaultAdapter();
        if (adapter == null) { Toast.makeText(this, "设备不支持蓝牙", Toast.LENGTH_LONG).show(); return; }
        if (!adapter.isEnabled()) {
            Toast.makeText(this, "请先在系统设置中打开蓝牙", Toast.LENGTH_LONG).show(); return;
        }
        try {
            startForegroundService(new Intent(this, BluetoothServerService.class));
            statusTitle.postDelayed(this::refreshStatus, 200L);
            Toast.makeText(this, "服务已启动，请用 Windows 连接", Toast.LENGTH_LONG).show();
        } catch (Exception e) {
            Toast.makeText(this, "启动失败：" + e.getMessage(), Toast.LENGTH_LONG).show();
        }
    }

    private LinearLayout column() {
        LinearLayout view = new LinearLayout(this);
        view.setOrientation(LinearLayout.VERTICAL);
        return view;
    }

    private LinearLayout card() {
        LinearLayout view = column();
        view.setPadding(dp(18), dp(17), dp(18), dp(17));
        view.setBackground(rounded(Color.WHITE, 20));
        view.setElevation(dp(2));
        return view;
    }

    private TextView text(String value, int sp, int color) {
        TextView view = new TextView(this);
        view.setText(value);
        view.setTextSize(sp);
        view.setTextColor(color);
        return view;
    }

    private Button button(String label, int background, int foreground, View.OnClickListener action) {
        Button view = new Button(this);
        view.setText(label);
        view.setTextSize(15);
        view.setTextColor(foreground);
        view.setTypeface(Typeface.DEFAULT_BOLD);
        view.setAllCaps(false);
        view.setBackground(rounded(background, 15));
        view.setOnClickListener(action);
        view.setStateListAnimator(null);
        return view;
    }

    private GradientDrawable rounded(int color, int radiusDp) {
        GradientDrawable drawable = new GradientDrawable();
        drawable.setColor(color);
        drawable.setCornerRadius(dp(radiusDp));
        return drawable;
    }

    private View space(int heightDp) {
        View view = new View(this);
        view.setLayoutParams(new LinearLayout.LayoutParams(1, dp(heightDp)));
        return view;
    }

    private LinearLayout.LayoutParams matchWrap() {
        return new LinearLayout.LayoutParams(-1, -2);
    }

    private int dp(int value) {
        return Math.round(value * getResources().getDisplayMetrics().density);
    }
}
