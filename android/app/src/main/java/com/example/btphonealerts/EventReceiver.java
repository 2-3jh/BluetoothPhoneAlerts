package com.example.btphonealerts;

import android.Manifest;
import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.content.SharedPreferences;
import android.content.pm.PackageManager;
import android.database.Cursor;
import android.net.Uri;
import android.provider.ContactsContract;
import android.provider.Telephony;
import android.telephony.SmsMessage;
import android.telephony.TelephonyManager;

public final class EventReceiver extends BroadcastReceiver {
    @Override public void onReceive(Context context, Intent intent) {
        String action = intent.getAction();
        if (Telephony.Sms.Intents.SMS_RECEIVED_ACTION.equals(action)) {
            SmsMessage[] parts = Telephony.Sms.Intents.getMessagesFromIntent(intent);
            if (parts == null || parts.length == 0) return;
            String number = null;
            StringBuilder body = new StringBuilder();
            for (SmsMessage part : parts) {
                if (part == null) continue;
                if (number == null) number = part.getOriginatingAddress();
                if (part.getMessageBody() != null) body.append(part.getMessageBody());
            }
            String name = contactName(context, number);
            String sender = name.isEmpty() ? safe(number, "未知号码") : name + "（" + safe(number, "未知号码") + "）";
            EventStore.add(context, "sms", "短信 · " + sender, body.toString());
        } else if (TelephonyManager.ACTION_PHONE_STATE_CHANGED.equals(action)) {
            if (!TelephonyManager.EXTRA_STATE_RINGING.equals(intent.getStringExtra(TelephonyManager.EXTRA_STATE))) return;
            // With call-log access Android may send a second broadcast containing the number.
            if (!intent.hasExtra(TelephonyManager.EXTRA_INCOMING_NUMBER) &&
                    context.checkSelfPermission(Manifest.permission.READ_CALL_LOG) == PackageManager.PERMISSION_GRANTED) return;
            String number = intent.getStringExtra(TelephonyManager.EXTRA_INCOMING_NUMBER);
            SharedPreferences p = context.getSharedPreferences("alerts", Context.MODE_PRIVATE);
            long now = System.currentTimeMillis();
            if (now - p.getLong("last_ring_at", 0L) <= 12_000L) return;
            p.edit().putLong("last_ring_at", now).apply();
            String name = contactName(context, number);
            String caller = name.isEmpty() ? safe(number, "未知号码") : name + "（" + safe(number, "未知号码") + "）";
            EventStore.add(context, "call", "手机来电", caller);
        }
    }

    private static String contactName(Context context, String number) {
        if (number == null || number.isEmpty() || context.checkSelfPermission(Manifest.permission.READ_CONTACTS) != PackageManager.PERMISSION_GRANTED) return "";
        Uri uri = Uri.withAppendedPath(ContactsContract.PhoneLookup.CONTENT_FILTER_URI, Uri.encode(number));
        try (Cursor cursor = context.getContentResolver().query(uri,
                new String[]{ContactsContract.PhoneLookup.DISPLAY_NAME}, null, null, null)) {
            return cursor != null && cursor.moveToFirst() ? safe(cursor.getString(0), "") : "";
        } catch (Exception ignored) { return ""; }
    }

    private static String safe(String value, String fallback) {
        return value == null || value.trim().isEmpty() ? fallback : value.trim();
    }
}
