package com.example.btphonealerts;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.Service;
import android.bluetooth.BluetoothAdapter;
import android.bluetooth.BluetoothServerSocket;
import android.bluetooth.BluetoothSocket;
import android.content.Intent;
import android.content.pm.ServiceInfo;
import android.os.Build;
import android.os.IBinder;
import android.os.SystemClock;
import android.os.BatteryManager;
import android.content.IntentFilter;
import android.util.Log;
import org.json.JSONObject;
import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.io.IOException;
import java.io.OutputStreamWriter;
import java.io.BufferedWriter;
import java.nio.charset.StandardCharsets;

public final class BluetoothServerService extends Service {
    private static final String TAG = "BtPhoneAlerts";
    private static final String CHANNEL = "btphone_channel";
    private volatile boolean running = false;
    private Thread worker;
    private volatile BluetoothServerSocket serverSocket;
    private volatile BluetoothSocket clientSocket;
    private static volatile boolean active = false;

    public static boolean isRunning() { return active; }

    @Override public void onCreate() {
        super.onCreate();
        NotificationManager nm = getSystemService(NotificationManager.class);
        nm.createNotificationChannel(new NotificationChannel(
                CHANNEL, "蓝牙连接服务", NotificationManager.IMPORTANCE_LOW));
    }

    @Override public int onStartCommand(Intent intent, int flags, int startId) {
        Notification notification = new Notification.Builder(this, CHANNEL)
                .setSmallIcon(android.R.drawable.stat_sys_data_bluetooth)
                .setContentTitle("来电短信蓝牙提醒")
                .setContentText("等待 Windows 电脑连接蓝牙")
                .setOngoing(true).build();
        if (Build.VERSION.SDK_INT >= 29) {
            startForeground(1, notification, ServiceInfo.FOREGROUND_SERVICE_TYPE_CONNECTED_DEVICE);
        } else {
            startForeground(1, notification);
        }
        if (!running) {
            running = true;
            active = true;
            worker = new Thread(this::listenLoop, "rfcomm-listener");
            worker.start();
        }
        return START_STICKY;
    }

    private void listenLoop() {
        while (running) {
            try {
                BluetoothAdapter adapter = BluetoothAdapter.getDefaultAdapter();
                if (adapter == null || !adapter.isEnabled()) {
                    SystemClock.sleep(2000);
                    continue;
                }
                serverSocket = adapter.listenUsingRfcommWithServiceRecord(
                        Protocol.SERVICE_NAME, Protocol.SERVICE_UUID);
                while (running) {
                    BluetoothSocket socket = serverSocket.accept();
                    if (!running) { socket.close(); break; }
                    clientSocket = socket;
                    queuePhoneStatus();
                    try { handleClient(socket); }
                    finally {
                        try { socket.close(); } catch (Exception ignored) {}
                        clientSocket = null;
                    }
                }
            } catch (Exception e) {
                if (running) Log.w(TAG, "Bluetooth listener restarting", e);
                SystemClock.sleep(2000);
            } finally {
                closeServer();
            }
        }
    }

    private void queuePhoneStatus() {
        Intent battery = registerReceiver(null, new IntentFilter(Intent.ACTION_BATTERY_CHANGED));
        if (battery == null) return;
        int level = battery.getIntExtra(BatteryManager.EXTRA_LEVEL, -1);
        int scale = battery.getIntExtra(BatteryManager.EXTRA_SCALE, 100);
        int status = battery.getIntExtra(BatteryManager.EXTRA_STATUS, -1);
        int percent = level < 0 ? -1 : Math.round(level * 100f / Math.max(1, scale));
        boolean charging = status == BatteryManager.BATTERY_STATUS_CHARGING ||
                status == BatteryManager.BATTERY_STATUS_FULL;
        String body = (percent < 0 ? "电量未知" : "电量 " + percent + "%") +
                (charging ? " · 正在充电" : " · 未充电");
        EventStore.add(this, "status", "手机状态", body);
    }

    private void handleClient(BluetoothSocket socket) throws Exception {
        // Secure RFCOMM requests pairing. Reject unpaired clients as defense in depth.
        if (socket.getRemoteDevice().getBondState() != android.bluetooth.BluetoothDevice.BOND_BONDED) {
            return;
        }
        BufferedWriter out = new BufferedWriter(new OutputStreamWriter(
                socket.getOutputStream(), StandardCharsets.UTF_8));
        InputStream in = socket.getInputStream();
        while (running) {
            JSONObject next = EventStore.peek(this);
            if (next == null) {
                SystemClock.sleep(400);
                continue;
            }
            String id = next.optString("id");
            out.write(next.toString());
            out.newLine();
            out.flush();
            String ack = readAck(in);
            if (!("ACK|" + id).equals(ack)) throw new java.io.IOException("Invalid ACK");
            EventStore.removeIfFirstId(this, id);
        }
    }

    /** Poll a short ACK line with timeout; BluetoothSocket has no setSoTimeout API. */
    private String readAck(InputStream in) throws IOException {
        ByteArrayOutputStream bytes = new ByteArrayOutputStream();
        long deadline = SystemClock.elapsedRealtime() + 5000L;
        while (running && SystemClock.elapsedRealtime() < deadline) {
            while (in.available() > 0) {
                int value = in.read();
                if (value < 0) throw new IOException("Socket closed");
                if (value == '\n') return bytes.toString("UTF-8").trim();
                if (bytes.size() >= 140) throw new IOException("ACK too long");
                bytes.write(value);
            }
            SystemClock.sleep(40L);
        }
        throw new IOException("Timed out waiting for ACK");
    }

    private void closeServer() {
        try { if (serverSocket != null) serverSocket.close(); } catch (Exception ignored) {}
        serverSocket = null;
    }

    @Override public void onDestroy() {
        running = false;
        active = false;
        closeServer();
        try { if (clientSocket != null) clientSocket.close(); } catch (Exception ignored) {}
        Thread oldWorker = worker;
        worker = null;
        if (oldWorker != null) {
            oldWorker.interrupt();
            try { oldWorker.join(800L); }
            catch (InterruptedException e) { Thread.currentThread().interrupt(); }
        }
        if (Build.VERSION.SDK_INT >= 24) stopForeground(STOP_FOREGROUND_REMOVE);
        else stopForeground(true);
        super.onDestroy();
    }

    @Override public IBinder onBind(Intent intent) { return null; }
}
