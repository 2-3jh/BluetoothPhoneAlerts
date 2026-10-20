package com.example.btphonealerts;

import android.content.Context;
import android.content.SharedPreferences;
import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;
import java.util.UUID;

/** Small on-device queue so SMS/call events survive brief Bluetooth disconnections. */
public final class EventStore {
    private static final String FILE = "alerts";
    private static final String QUEUE = "queue";

    private EventStore() {}

    private static SharedPreferences pref(Context c) {
        return c.getApplicationContext().getSharedPreferences(FILE, Context.MODE_PRIVATE);
    }

    private static JSONArray load(Context c) {
        try { return new JSONArray(pref(c).getString(QUEUE, "[]")); }
        catch (JSONException e) { return new JSONArray(); }
    }

    public static synchronized void add(Context c, String kind) {
        if (!kind.equals("sms") && !kind.equals("call") && !kind.equals("test")) return;
        try {
            JSONArray old = load(c);
            JSONArray next = new JSONArray();
            // Keep the last 99 events, plus the new one.
            for (int i = Math.max(0, old.length() - 99); i < old.length(); i++) {
                next.put(old.getJSONObject(i));
            }
            JSONObject alert = new JSONObject();
            alert.put("id", UUID.randomUUID().toString());
            alert.put("type", kind);
            next.put(alert);
            pref(c).edit().putString(QUEUE, next.toString()).commit();
        } catch (JSONException ignored) {}
    }

    public static synchronized JSONObject peek(Context c) {
        JSONArray a = load(c);
        return a.optJSONObject(0);
    }

    public static synchronized void removeIfFirstId(Context c, String id) {
        JSONArray old = load(c);
        JSONObject first = old.optJSONObject(0);
        if (first == null || !id.equals(first.optString("id"))) return;
        JSONArray next = new JSONArray();
        for (int i = 1; i < old.length(); i++) next.put(old.opt(i));
        pref(c).edit().putString(QUEUE, next.toString()).commit();
    }
}
