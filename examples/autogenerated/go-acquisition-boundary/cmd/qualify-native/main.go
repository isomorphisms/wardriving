// Build-host-only executable qualifier; never install this on the C67.
package main

import (
	"crypto/sha256"
	"debug/elf"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func run(path string, args []string, changes map[string]string) {
	cmd := exec.Command(path, args...)
	for _, v := range os.Environ() {
		key, _, _ := strings.Cut(v, "=")
		if _, replace := changes[key]; !replace {
			cmd.Env = append(cmd.Env, v)
		}
	}
	for k, v := range changes {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func main() {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" || runtime.Version() != "go1.27.3" {
		panic("exact pinned Linux amd64 Go build host required")
	}
	sdk := os.Getenv("ANDROID_HOME")
	if !filepath.IsAbs(sdk) {
		panic("absolute ANDROID_HOME required")
	}
	if len(os.Args) == 2 && os.Args[1] == "prepare-ndk" {
		manager := filepath.Join(sdk, "cmdline-tools", "latest", "bin", "sdkmanager")
		if info, err := os.Stat(manager); err != nil || info.Mode()&0111 == 0 {
			panic("verified SDK manager unavailable")
		}
		run(manager, []string{"--sdk_root=" + sdk, "ndk;27.2.12479018"}, nil)
		return
	}
	if len(os.Args) != 1 {
		panic("usage: qualify-native [prepare-ndk]")
	}
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if _, err = os.Stat(filepath.Join(root, "go.mod")); err != nil {
		panic(err)
	}
	output := filepath.Join(os.Getenv("RUNNER_TEMP"), "wifi-go-native")
	if !filepath.IsAbs(output) {
		panic("absolute RUNNER_TEMP required")
	}
	if err = os.Mkdir(output, 0700); err != nil {
		panic(err)
	}
	ndk := filepath.Join(sdk, "ndk", "27.2.12479018")
	properties, err := os.ReadFile(filepath.Join(ndk, "source.properties"))
	if err != nil {
		panic(err)
	}
	if !strings.Contains(string(properties), "Pkg.Revision = 27.2.12479018") {
		panic("NDK identity mismatch")
	}
	clang, err := exec.LookPath("clang")
	if err != nil {
		panic(err)
	}
	version, err := exec.Command(clang, "--version").Output()
	if err != nil || !strings.Contains(string(version), "clang version") {
		panic("host Clang required; no GCC fallback")
	}
	goTool := filepath.Join(runtime.GOROOT(), "bin", "go")
	hostlib := filepath.Join(output, "libwifi_host.so")
	run(goTool, []string{"build", "-trimpath", "-buildmode=c-shared", "-o", hostlib, "./cmd/shared"}, map[string]string{"CGO_ENABLED": "1", "GOOS": "linux", "GOARCH": "amd64", "CC": clang, "GOTOOLCHAIN": "local"})
	observer := filepath.Join(output, "abi-observer")
	run(clang, []string{"-std=c11", "-Wall", "-Wextra", "-Werror", "abi/observer.c", "-ldl", "-o", observer}, nil)
	run(observer, []string{hostlib}, nil)
	run(goTool, []string{"test", "-race", "./..."}, map[string]string{"CGO_ENABLED": "1", "CC": clang, "GOTOOLCHAIN": "local"})
	androidlib := filepath.Join(output, "libwifi_android_arm64.so")
	androidCC := filepath.Join(ndk, "toolchains", "llvm", "prebuilt", "linux-x86_64", "bin", "aarch64-linux-android34-clang")
	run(goTool, []string{"build", "-trimpath", "-buildmode=c-shared", "-o", androidlib, "./cmd/shared"}, map[string]string{"CGO_ENABLED": "1", "GOOS": "android", "GOARCH": "arm64", "CC": androidCC, "GOTOOLCHAIN": "local"})
	f, err := elf.Open(androidlib)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if f.Machine != elf.EM_AARCH64 || f.Type != elf.ET_DYN {
		panic("Android shared ELF identity mismatch")
	}
	symbols, err := f.DynamicSymbols()
	if err != nil {
		panic(err)
	}
	for _, name := range []string{"WD_ABIVersion", "WD_ObserveNetwork", "WD_CurrentGeneration", "WD_CurrentHandle"} {
		found := false
		for _, symbol := range symbols {
			if symbol.Name == name && symbol.Section != elf.SHN_UNDEF {
				found = true
			}
		}
		if !found {
			panic("missing native export " + name)
		}
	}
	bytes, err := os.ReadFile(androidlib)
	if err != nil {
		panic(err)
	}
	fmt.Printf("PASS Android ARM64 API34 Go c-shared link/export checks\nsha256\t%x\nAndroid_runtime\tNOT_RUN\nphysical_C67\tNOT_RUN\n", sha256.Sum256(bytes))
}
