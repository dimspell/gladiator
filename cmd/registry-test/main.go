//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const path = `SOFTWARE\WOW6432Node\AbalonStudio\Dispel\Multi`
const key = "Server"

func main() {
	//
}

func usingPowershell() {
	cmd := "reg.exe"
	newValue := time.Now().Format(time.TimeOnly)
	args := strings.Join([]string{
		"ADD",
		`HKEY_LOCAL_MACHINE\SOFTWARE\WOW6432Node\AbalonStudio\Dispel\Multi`,
		"/v", "Server",
		"/t", "REG_SZ",
		"/f",
		"/d", newValue,
	}, " ")

	r := exec.Command("powershell.exe", "Start-Process", cmd, "-Verb", "runAs", "-ArgumentList", `"`+args+`"`)
	r.Stdin = os.Stdin
	r.Stdout = os.Stdout
	r.Stderr = os.Stderr

	log.Fatal(r.Run())
}

func workingVersion() {
	if !windows.GetCurrentProcessToken().IsElevated() {
		runSelf()
		time.Sleep(1 * time.Second)
		readKey()
		return
	}

	saveKey()
}

func readKey() {
	openKey, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err != nil {
		log.Fatal("Could not open: ", err)
	}
	defer openKey.Close()

	s, _, err := openKey.GetStringValue(key)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Value in the registry is %q\n", s)
}

func saveKey() {
	saveKey, exist, err := registry.CreateKey(registry.LOCAL_MACHINE, path, registry.SET_VALUE)
	if err != nil {
		log.Fatal("Could not create: ", err)
	}
	defer saveKey.Close()

	fmt.Println(saveKey, exist)

	if err := saveKey.SetStringValue("Server", time.Now().Format(time.TimeOnly)); err != nil {
		log.Fatal("Could not change: ", err)
	}
}

func runSelf() {
	verb := "runas"
	exe, _ := os.Executable()
	cwd, _ := os.Getwd()
	args := strings.Join(os.Args[1:], " ")

	verbPtr, err := syscall.UTF16PtrFromString(verb)
	if err != nil {
		log.Fatal(err)
	}
	exePtr, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		log.Fatal(err)
	}
	cwdPtr, err := syscall.UTF16PtrFromString(cwd)
	if err != nil {
		log.Fatal(err)
	}
	argPtr, err := syscall.UTF16PtrFromString(args)
	if err != nil {
		log.Fatal(err)
	}

	var showCmd int32 = 0

	if err := windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd); err != nil {
		fmt.Println(err)
	}

	fmt.Println("Done")
}
