// wifi-boundary is a read-only capability diagnostic, not a connection app.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	boundary "isomorphisms/wardriving/go-acquisition-boundary"
)

func main() {
	rish := flag.String("rish", "", "absolute path to the already-authorized rish launcher")
	flag.Parse()
	if !filepath.IsAbs(*rish) || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "supply --rish with its verified absolute executable path")
		os.Exit(2)
	}
	fmt.Printf("schema\twardriving-readonly-boundary-v1\nbuild_go\t%s\nbuild_target\t%s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Println("scope\tread-only; no scan request, association, login, credential access or internet probe")
	fmt.Println("privacy\toutput contains local network identifiers; keep private")
	queries := []struct{ name, command string }{
		{"identity", "id"},
		{"model", "getprop ro.product.model"},
		{"android", "getprop ro.build.version.release"},
		{"sdk", "getprop ro.build.version.sdk"},
		{"abi", "getprop ro.product.cpu.abilist"},
		{"wifi_help", "cmd wifi help"},
		{"wifi_status", "cmd wifi status"},
		{"saved_metadata", "cmd wifi list-networks"},
		{"scan_cache", "cmd wifi list-scan-results"},
	}
	failed := false
	for _, q := range queries {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		r, err := boundary.Capture(ctx, *rish, []string{"-c", q.command}, 32768)
		cancel()
		fmt.Printf("%s.elapsed_us\t%d\n%s.stdout\t%q\n%s.stderr\t%q\n", q.name, r.Elapsed.Microseconds(), q.name, r.Stdout, q.name, r.Stderr)
		if err != nil {
			fmt.Printf("%s.result\tFAIL %q\n", q.name, err.Error())
			failed = true
		} else {
			fmt.Printf("%s.result\tCOMMAND_COMPLETED_NOT_SEMANTIC_ACCEPTANCE\n", q.name)
		}
		if q.name == "identity" && (err != nil || !shellUID(r.Stdout)) {
			fmt.Println("authority\tREFUSED expected shell UID 2000")
			os.Exit(1)
		}
	}
	fmt.Println("remote_cancellation\tNOT_PROVEN\nphysical_wifi_binding\tNOT_RUN\nsensor_inventory\tNOT_RUN")
	if failed {
		os.Exit(1)
	}
}

func shellUID(output string) bool {
	// id emits uid=2000(shell) on the target. Exact token boundaries avoid
	// interpreting another UID such as 20000 as shell authority.
	for i := 0; i+9 <= len(output); i++ {
		if (i == 0 || output[i-1] == ' ' || output[i-1] == '\n') && output[i:i+9] == "uid=2000(" {
			return true
		}
	}
	return false
}
