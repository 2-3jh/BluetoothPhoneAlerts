package com.example.btphonealerts;

import java.util.UUID;

/** Must match Program.cs on Windows. */
public final class Protocol {
    private Protocol() {}
    public static final UUID SERVICE_UUID =
            UUID.fromString("bac20533-10ac-4c14-b8e9-45a1b29feb17");
    public static final String SERVICE_NAME = "BluetoothPhoneAlerts";
}
