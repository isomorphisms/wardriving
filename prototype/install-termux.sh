#!/data/data/com.termux/files/usr/bin/sh
# Only copies scripts. No compilers, downloads or package installs.
set -eu
HERE=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
: "${PREFIX:=/data/data/com.termux/files/usr}"
mkdir -p "$HOME/wardriving" "$PREFIX/bin" "$HOME/.shortcuts"
cp "$HERE/wifiopen.awk" "$HOME/wardriving/wifiopen.awk"
cp "$HERE/wifiopen-scan.sh" "$HOME/wardriving/wifiopen-scan.sh"
chmod 700 "$HOME/wardriving/wifiopen-scan.sh"
cat > "$PREFIX/bin/wifiopen-scan" <<'END'
#!/data/data/com.termux/files/usr/bin/sh
exec "$HOME/wardriving/wifiopen-scan.sh" "$@"
END
chmod 700 "$PREFIX/bin/wifiopen-scan"
cat > "$HOME/.shortcuts/WiFi-Open" <<'END'
#!/data/data/com.termux/files/usr/bin/sh
exec "$HOME/wardriving/wifiopen-scan.sh"
END
chmod 700 "$HOME/.shortcuts/WiFi-Open"
printf '%s\n' 'WiFi-Open installed as scripts; no compilation performed.' 'Run wifiopen-scan after configuring Shizuku rish.'
