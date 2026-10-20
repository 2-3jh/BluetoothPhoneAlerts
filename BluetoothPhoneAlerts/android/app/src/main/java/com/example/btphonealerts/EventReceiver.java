package com.example.btphonealerts;

import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.content.SharedPreferences;
import android.provider.Telephony;
import android.telephony.TelephonyManager;

/** Only records that an event happened; reads neither SMS contents nor caller ID. */
public final class EventReceiver extends BroadcastReceiver {
    @Override public void onReceive(Context context, Intent intent) {
        String action = intent.getAction();
        if (Telephony.Sms.Intents.SMS_RECEIVED_ACTION.equals(action)) {
            EventStore.add(context, "sms");
        } else if (TelephonyManager.ACTION_PHONE_STATE_CHANGED.equals(action)) {
            String state = intent.getStringExtra(TelephonyManager.EXTRA_STATE);
            if (TelephonyManager.EXTRA_STATE_RINGING.equals(state)) {
                // Some devices deliver duplicated PHONE_STATE broadcasts.
                SharedPreferences p = context.getSharedPreferences("alerts", Context.MODE_PRIVATE);
                long now = System.currentTimeMillis();
                long last = p.getLong("last_ring_at", 0L);
                if (now - last > 12_000L) {
                    p.edit().putLong("last_ring_at", now).apply();
                    EventStore.add(context, "call");
                }
            }
        }
    }
}
