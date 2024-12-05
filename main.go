package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

var user32 = syscall.NewLazyDLL("user32.dll")
var MessageBox = user32.NewProc("MessageBoxW")

func MessageBoxW(title, message string) (int, error) {
	ret, _, err := MessageBox.Call(
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(message))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title))),
		0x00000010,
	)
	if err != nil && err.Error() != "The operation completed successfully." {
		return 0, err
	}
	return int(ret), nil
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			stackBuf := make([]byte, 2048)
			stackSize := runtime.Stack(stackBuf, true)

			message := fmt.Sprintf("Panic Occured: %v\n\n%s", r, stackBuf[:stackSize])
			title := "Whoops, something went wrong!"

			if _, err := MessageBoxW(title, message); err != nil {
				panic(err)
			}
		}
	}()
	version, err := getLatestVersion()
	if err != nil {
		panic(err)
	}
	currentVersion, err := getCurrentHash()
	if err != nil {
		panic(err)
	}
	fmt.Println("Current version:", currentVersion)
	if currentVersion == version {
		fmt.Println("You are already at latest version!")
		if _, err := os.StartProcess("./runtime/launch.exe", nil, nil); err != nil {
			panic(err)
		}
		os.Exit(0)
	}
	bytesV, err := getLatestVersionBytes(version)
	if err != nil {
		panic(err)
	}
	fileHash, err := hashBytes(bytesV)
	if err != nil {
		panic(err)
	}
	_ = os.Mkdir("./runtime", 0777)
	exePath := "./runtime/launch.exe"
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		panic(err)
	}
	if fileHash == currentVersion {
		fmt.Println("You are already at latest version!")
	} else {
		fmt.Println("Installing latest version...")
		if err := os.WriteFile("./runtime/launch.exe", bytesV, 0777); err != nil {
			panic(err)
		}
	}
	discardFile, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		panic(err)
	}
	defer discardFile.Close()
	procAttr := new(os.ProcAttr)
	procAttr.Files = []*os.File{discardFile, discardFile, discardFile}
	procAttr.Env = os.Environ()
	procAttr.Dir = "./runtime"

	process, err := os.StartProcess(exePath, []string{exePath}, procAttr)
	if err != nil {
		panic(err)
	}
	exitState, err := process.Wait()
	if err != nil {
		panic(err)
	}
	println("Exited with status:", exitState.String())
	os.Exit(0)
}

const VersionUrl = "https://github.com/TrippleAWap/SnorlaxReleases/releases/latest"

func getLatestVersion() (string, error) {
	req, _ := http.NewRequest("GET", VersionUrl, nil)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", fmt.Errorf("failed to get latest version - %s", res.Status)
	}
	breadcrumbs := strings.Split(res.Request.URL.String(), "/")
	return breadcrumbs[len(breadcrumbs)-1], nil
}

func hashBytes(bytesV []byte) (string, error) {
	hash := sha256.New()

	if _, err := io.Copy(hash, bytes.NewReader(bytesV)); err != nil {
		return "", err
	}

	hashBytes := hash.Sum(nil)

	return fmt.Sprintf("%x", hashBytes), nil
}

func getCurrentHash() (string, error) {
	file, err := os.OpenFile("./runtime/launch.exe", os.O_RDONLY, 0777)
	if err != nil {
		if os.IsNotExist(err) {
			return "none", nil
		}
		return "", err
	}
	bytesV, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	fileHash, err := hashBytes(bytesV)
	if err != nil {
		return "", err
	}
	return fileHash, nil
}

func getLatestVersionBytes(version string) ([]byte, error) {
	url := fmt.Sprintf("https://github.com/TrippleAWap/SnorlaxReleases/releases/download/%s/Snorlax.exe", version)
	req, _ := http.NewRequest("GET", url, nil)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to download latest version - %s", res.Status)
	}
	_ = os.Mkdir("./runtime", 0777)
	bytesV, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return bytesV, nil
}
