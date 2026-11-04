# Android Wi-Fi discovery and connection plan

## Stage A — read-only, compiler-free scout

The current scanner asks Shizuku rish (running as the Android shell user) for 'cmd wifi start-scan' and then 'cmd wifi list-scan-results'. The scan command may finish before a new scan does, and the system can throttle scans: show observation age rather than reporting cached results as new. Defaults: RSSI at least -80 dBm, age at most 30 seconds.

The AOSP output is fixed-width but permits per-radio-chain RSSI suffixes, UTF-8 SSIDs, spaces in SSIDs, and appended security flags. Parse conservatively, reject unknown formats rather than misidentify protected networks, and accept only Open or OWE networks. Validate the exact output on the physical C67.

## Stage B — native UI

The user has selected a Go acquisition engine compiled off-device with a pinned Go toolchain. Use the existing typed Android-NDK framework/JNI boundary for narrow adapters; ICK applies to owned C where qualified, not to Go compilation. Do not install any compiler on the phone or introduce first-party Java/Kotlin. The [Go acquisition decision](go-acquisition/decision.md) records the source investigation, executable experiments, evidence limits and [bounded implementation jobs](go-acquisition/implementation-jobs.md). This branch does not yet provide the finished app or an APK.

Show ranked SSIDs, security, BSSID, frequency, signal and age in a short list with large touch targets. Android network suggestions and request APIs have user-approval and routing restrictions; do not claim that Shizuku alone can silently force system-wide network association.

## Stage C — timing

Record scan request, scan result, explicit selection, association, DHCP/IPv6, DNS and portal/Internet validation timestamps. Use Android's built-in captive-portal UI. Do not capture passwords or portal credentials. Connecting to passwordless radio is not equivalent to having usable Internet.

## Shizuku rish on MIRO C67

Export 'rish' and 'rish_shizuku.dex' from Shizuku. Copy both, not symlink, from Android shared storage into Termux private storage (for example ~/opt). Replace the 'PKG' placeholder with 'com.termux' where present and apply 'chmod 400' to the dex on Android 14. The launcher recognizes PATH rish, ~/opt/rish, and ~/rish. Success means 'rish -c id' reports uid=2000(shell).

## Required qualification

1. ICK-produced AArch64 Android/Bionic object, link and executable/ART smoke checks.
2. Independently tested Android API 34 runtime and permission-denial behavior.
3. Real C67 scan-output and permission acceptance.
4. A signed arm64-v8a APK with reproducible provenance and downloadable build artifact.
5. Timing measurements on an authorized public AP, including any captive portal.

## References

- https://android.googlesource.com/platform/packages/modules/Wifi/+/refs/heads/android14-release/service/java/com/android/server/wifi/WifiShellCommand.java
- https://android.googlesource.com/platform/packages/modules/Wifi/+/refs/heads/main/framework/java/android/net/wifi/util/ScanResultUtil.java
- https://developer.android.com/develop/connectivity/wifi/wifi-scan
- https://github.com/isomorphisms/ick/blob/main/docs/android-release-gate.md
