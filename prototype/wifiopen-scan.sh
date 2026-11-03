#!/data/data/com.termux/files/usr/bin/sh
# No compiler and no downloaded native binaries. Android 14 has system awk.
set -eu
HERE=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
RISH_APPLICATION_ID=com.termux
export RISH_APPLICATION_ID
if command -v rish >/dev/null 2>&1; then
  RISH=rish
elif [ -x "$HOME/opt/rish" ]; then
  RISH="$HOME/opt/rish"
elif [ -x "$HOME/rish" ]; then
  RISH="$HOME/rish"
else
  echo 'Missing rish. Copy (do not symlink) rish and rish_shizuku.dex into ~/opt, then chmod 700 ~/opt/rish and chmod 400 ~/opt/rish_shizuku.dex.' >&2
  exit 1
fi
if [ -x /system/bin/awk ]; then
  AWK=/system/bin/awk
elif command -v awk >/dev/null 2>&1; then
  AWK=awk
else
  echo 'Missing awk; this Android build lacks the AOSP system awk.' >&2
  exit 1
fi
min_rssi=-80
max_age=30
limit=8
details=0
picker=0
no_prompt=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    --min-rssi|--max-age|--limit)
      [ "$#" -ge 2 ] || { echo "Missing value: $1" >&2; exit 2; }
      case "$1" in
        --min-rssi) min_rssi=$2;;
        --max-age) max_age=$2;;
        --limit) limit=$2;;
      esac
      shift 2;;
    --details) details=1; shift;;
    --picker) picker=1; no_prompt=1; shift;;
    --no-prompt) no_prompt=1; shift;;
    --help)
      echo 'Usage: wifiopen-scan [--min-rssi -80] [--max-age 30] [--limit 8] [--details] [--no-prompt] [--picker]'
      exit 0;;
    *) echo "Unknown option: $1" >&2; exit 2;;
  esac
done
# One Shizuku invocation: scan is asynchronous; result age is checked below.
OUTPUT=$("$RISH" -c 'cmd wifi start-scan; sleep 2; cmd wifi list-scan-results' 2>&1) || {
  printf 'Android Wi-Fi scan failed:\n%s\n' "$OUTPUT" >&2
  exit 1
}
case "$OUTPUT" in
  *BSSID*Frequency*RSSI*'Age(sec)'*) ;;
  *) printf 'Unexpected Android scan output:\n%s\n' "$OUTPUT" >&2; exit 1;;
esac
printf '%s\n' "$OUTPUT" | "$AWK" -v "min_rssi=$min_rssi" -v "max_age=$max_age" -v "limit=$limit" -v "details=$details" -f "$HERE/wifiopen.awk"
if [ "$picker" -eq 1 ]; then
  "$RISH" -c 'am start -a android.net.wifi.PICK_WIFI_NETWORK'
elif [ "$no_prompt" -eq 0 ] && [ -t 0 ]; then
  printf '\nOpen Android Wi-Fi picker now? [y/N] '
  IFS= read -r answer || exit 0
  case "$answer" in
    y|Y) "$RISH" -c 'am start -a android.net.wifi.PICK_WIFI_NETWORK';;
  esac
fi
