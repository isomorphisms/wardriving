# WiFi-Open: passwordless Wi-Fi candidates from AOSP Android 14 scan cache.
# POSIX awk, intended for Android's stock /system/bin/awk.
# Input is the output of cmd wifi list-scan-results.
# No network association and no portal login are performed.
BEGIN {
    if (min_rssi == "") min_rssi=-80
    if (max_age == "") max_age=30
    if (limit == "") limit=8
    if (details == "") details=0
    if (min_rssi !~ /^-?[0-9]+$/ || min_rssi < -127 || min_rssi > 0 ||
        max_age !~ /^[0-9]+([.][0-9]+)?$/ || max_age > 1000 ||
        limit !~ /^[0-9]+$/ || limit < 1 || limit > 100) {
        print "Invalid wifiopen-scan option" > "/dev/stderr"
        exit 2
    }
    count=0
}
function clean(s) {
    gsub(/^[ \t]+/,"",s)
    gsub(/[ \t]+$/,"",s)
    return s
}
function token_capabilities(s, result,  pos,part,cap,sep) {
    # Remove flags from the right without consuming SSID text such as "[free]".
    result=""
    while (s ~ /\]$/) {
        if (!match(s,/\[[^][]+\]$/)) return ""
        cap=substr(s,RSTART,RLENGTH)
        result=cap result
        s=substr(s,1,RSTART-1)
        # Capabilities have no intervening space. AOSP separates them
        # from the SSID with one or more spaces.
        if (s ~ /[ \t]$/) break
    }
    if (result == "" || s !~ /[ \t]$/) return ""
    return result
}
function emit(i, x,  sanitized) {
    sanitized=ssid[x]
    gsub(/[\001-\037\177]/,"?",sanitized)
    printf "%2d. %-8s %4d dBm %2.0fs %s", i, security[x],rssi[x],age[x],sanitized
    if (radios[x]>1) printf " (%d APs)", radios[x]
    printf "\n"
    if (details) printf "     %s  %d MHz  %s\n",bssid[x],freq[x],caps[x]
}
# Fixed ASCII header and fixed-width fields; SSID may contain spaces and UTF-8.
# Field positions, 1-based: BSSID 2-18, MHz 20-28, RSSI 30-47,
# age 49-55, SSID from 57. The capability suffix is peeled from the right.
/^ / && length($0) >= 90 {
    mac=clean(substr($0,2,17))
    mhz=clean(substr($0,20,9))
    power=clean(substr($0,30,18))
    seconds=clean(substr($0,49,7))
    if (mac !~ /^([[:xdigit:]][[:xdigit:]]:){5}[[:xdigit:]][[:xdigit:]]$/) next
    if (mhz !~ /^[0-9]+$/ || mhz < 2300 || mhz > 7200) next
    # AOSP includes per-radio-chain levels in parentheses, e.g.
    # -55(0:-63/1:-57). The aggregate reading is the leading integer.
    if (power !~ /^-[0-9]+(\([0-9]+:-?[0-9]+(\/[0-9]+:-?[0-9]+)?\))?$/) next
    sub(/\(.*/, "", power)
    if (power < -127 || power > 0 || (power+0) < (min_rssi+0)) next
    if (seconds !~ /^[0-9]+([.][0-9]+)?$/ || (seconds+0) > (max_age+0)) next
    line=$0
    sub(/[ \t\r]+$/, "", line)
    flags=token_capabilities(line)
    if (flags == "" || index(flags,"[ESS]")==0) next
    if (flags ~ /WEP|PSK|SAE|EAP|WAPI|DPP|WPA|\?/) next
    if (flags ~ /OWE_TRANSITION/) sec="OPEN/OWE"
    else if (flags ~ /OWE/) sec="OWE"
    else if (flags ~ /RSN/) next
    else sec="OPEN"
    # Remove exactly the capability suffix and separating space, not content
    # inside a displayed SSID such as "Café [free]".
    if (substr(line,length(line)-length(flags)+1)!=flags) next
    ss=substr(line,57,length(line)-length(flags)-56)
    ss=clean(ss)
    if (ss == "") next
    # Reject SSIDs containing control characters.
    if (ss ~ /[\001-\037\177]/) next
    key=sec SUBSEP ss
    if (key in where) {
        x=where[key];radios[x]++
        if ((power+0) > rssi[x]) {
            rssi[x]=power+0; age[x]=seconds+0; bssid[x]=mac;freq[x]=mhz+0;caps[x]=flags
        }
    } else if (count < 1024) {
        x=++count;where[key]=x
        ssid[x]=ss; security[x]=sec;rssi[x]=power+0;age[x]=seconds+0
        bssid[x]=mac;freq[x]=mhz+0;caps[x]=flags;radios[x]=1
    }
}
END {
    if (count==0) {
        print "No fresh, sufficiently strong passwordless Wi-Fi in scan cache."
        print "(Scan may be throttled/stale; no portal check was performed.)"
        exit
    }
    # Choose best visible result each iteration; at most 100 rows, with
    # ordinary scan result sets much smaller than 1024 candidates.
    print "Passwordless candidates (not verified for Internet):"
    n=(count<limit?count:limit)
    for (p=1;p<=n;p++) {
        best=0
        for (j=1;j<=count;j++) if (!shown[j]) {
            if (!best || rssi[j]>rssi[best] ||
               (rssi[j]==rssi[best] && age[j]<age[best]) ||
               (rssi[j]==rssi[best] && age[j]==age[best] && ssid[j]<ssid[best])) best=j
        }
        if (!best) break
        shown[best]=1
        emit(p,best)
    }
    if (count>limit) printf "... %d more candidates hidden by limit.\n",count-limit
    print "Selection/connection is handled separately by Android."
}
