# Wardriving

**Purpose:** fast discovery of nearby passwordless Wi-Fi on a stock Android 14 device, emphasizing public access points and deliberately hiding protected residential networks. This repository is currently an experimental scanner, **not** an automatic-connection app.

## Working prototype

`prototype/` contains an AOSP scan-output filter in POSIX `awk`, a Shizuku `rish` launcher, and a Termux installer. It is executable as source: **no Clang, GCC, ICK, Gradle, or other compiler is installed or invoked on the phone.** The scripts remain in the repository as a diagnostic fallback after the native application exists.

On the MIRO C67, the intended setup is Termux + Shizuku's authorized `rish` shell. Android's shared `/storage/emulated/0` is not an executable location: **copy**, do not symlink, the Shizuku exports to Termux private storage. The following commands match a Shizuku export in `~/Opt`:

```sh
mkdir -p ~/opt
rm -f ~/opt/rish ~/opt/rish_shizuku.dex
cp -L ~/Opt/rish ~/Opt/rish_shizuku.dex ~/opt/
sed -i 's/PKG/com.termux/g' ~/opt/rish
chmod 700 ~/opt/rish
chmod 400 ~/opt/rish_shizuku.dex
RISH_APPLICATION_ID=com.termux ~/opt/rish -c 'id'
```

With Shizuku running and Termux authorized, the expected output includes `uid=2000(shell)`. The `rish_shizuku.dex` file must be non-writable for Android 14. If the exported `rish` script has a placeholder `PKG`, it must be set to `com.termux` as above. Download/extract the repository, then run from inside the `prototype` directory:

```sh
sh install-termux.sh
wifiopen-scan --min-rssi -80 --max-age 30
```

Add `--details` for BSSID/frequency/capabilities, `--no-prompt` for a noninteractive scan, or `--picker` to open Android's standard Wi-Fi picker after scanning. The installer copies only scripts into the Termux home and adds a Termux:Widget shortcut. Shizuku setup is separate.

Status: the filter is host-tested against synthetic rows modeled on AOSP's exact scan-output layout. **It has not yet been physically tested on the C67.** There is no tested APK or real one-tap association yet.

## Constraints

- Show Open and Enhanced Open (OWE) networks; exclude WPA/WPA2/WPA3, WEP, EAP, and unresolved protection types.
- Filter stale and weak observations, group multiple BSSIDs by SSID/security, sort by RSSI, and expose age and frequency for diagnosis.
- Treat passwordless as *no advertised Wi-Fi password*, not permission, Internet connectivity, or absence of a captive portal.
- Require an explicit user choice to join. Do not bypass captive-portal terms or connect to untrusted networks automatically.
- No telemetry, location history, packet capture, or network probing in the initial slice.
- No builds on the phone. For the native app, compile **off-device with ICK**, and use the existing Idriç/Android-NDK integration where appropriate. Do not claim ICK Android executable/APK qualification before the relevant runtime and package gates pass.
- No first-party Java/Kotlin code. Android's framework APIs can be accessed through JNI in a native APK.

## Tests

On a development host with `python3` and `awk`:

```sh
sh prototype/tests/test.sh
```

Source details and the next-stage acceptance criteria are in [docs/architecture.md](docs/architecture.md).

## Safety

Do not interact with the UI while driving. Identical SSIDs can be spoofed; a strong signal is not authentication. Use only networks whose operators permit access.

License: MIT for the files introduced here.