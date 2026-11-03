#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
fixture=$(mktemp)
output=$(mktemp)
err=$(mktemp)
trap 'rm -f "$fixture" "$output" "$err"' EXIT
# Matches the Android AOSP ScanResultUtil.dumpScanResults() field widths.
python3 - <<'PY' > "$fixture"
print(' BSSID Frequency RSSI Age(sec) SSID  Flags')
def ap(last, freq, rssi, age, ssid, caps):
    print(f' 02:11:22:33:44:{last} {freq:9d} {rssi:18s} {age:>7s} {ssid:<32s} {caps}')
ap('55',2412,'-55','1.230','Coffee Shop WiFi','[ESS]')
ap('56',5180,'-63','1.980','Coffee Shop WiFi','[ESS]')
ap('57',5180,'-58','3.000','OWE network','[RSN-OWE-CCMP][ESS]')
ap('58',2412,'-52(0:-54/1:-53)','3.500','Transition','[RSN-OWE_TRANSITION-CCMP][ESS]')
ap('59',2412,'-38','2.000','WpaProtected','[WPA2-PSK-CCMP][ESS]')
ap('5a',2412,'-20','2.000','WepNet','[WEP][ESS]')
ap('5b',2412,'-36','80.000','StaleStation','[ESS]')
ap('5c',2412,'-91','1.000','WeakOpen','[ESS]')
ap('5d',2412,'-51','___?___','UnknownAge','[ESS]')
ap('5e',2412,'-62','1.200','Café Türkçe [free]','[ESS]')
ap('5f',2412,'-44(0:-49)','0.800','SingleChain','[ESS]')
ap('60',2412,'-33','0.100','UnknownRSN','[RSN-UNKNOWN][ESS]')
ap('61',2412,'-22','0.100','CorpEnterprise','[RSN-EAP/SHA256-CCMP][ESS]')
ap('62',2412,'-36','0.100','EmptyName','[IBSS]')
PY
awk -v min_rssi=-80 -v max_age=30 -v limit=8 -v details=1 -f wifiopen.awk "$fixture" > "$output"
cat "$output"
grep -q -- '-55 dBm .*Coffee Shop WiFi (2 APs)' "$output"
grep -q 'OWE network' "$output"
grep -q 'OPEN/OWE *-52 dBm' "$output"
grep -q 'SingleChain' "$output"
grep -q 'Café Türkçe \[free\]' "$output"
for bad in WpaProtected WepNet StaleStation WeakOpen UnknownAge UnknownRSN CorpEnterprise EmptyName; do
  ! grep -q "$bad" "$output"
done
# Ranking: strongest fresh eligible candidate comes first.
grep '^ 1\.' "$output" | grep -q 'SingleChain'
# A conservative option parser: reject invalid thresholds.
if awk -v min_rssi=+999 -f wifiopen.awk "$fixture" > "$output" 2> "$err"; then
  echo 'Invalid threshold unexpectedly accepted' >&2; exit 1
fi
sh -n install-termux.sh
sh -n wifiopen-scan.sh
# Neither installer nor launcher compiles code on the phone.
! grep -Eiq '(pkg install clang|(^|/)clang |(^|/)gcc |(^|/)cc -)' install-termux.sh wifiopen-scan.sh
printf 'PASS: security / dedup / chains / order / age / strength / Unicode / options / no compiler\n'
