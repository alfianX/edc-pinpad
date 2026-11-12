//go:build windows
// +build windows

package license

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

func GetCurrentDir() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}

	// Print the executable file path
	// fmt.Println("Executable path:", ex)

	// Get the directory of the executable
	exPath := filepath.Dir(ex)
	// fmt.Println("Executable directory:", exPath)
	return exPath
}

func GetVolume() string {
	path := GetCurrentDir()

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procGetVolumeInformationA := kernel32.NewProc("GetVolumeInformationA")

	var (
		volumeSerialNumber uint32
		maxComponentLength uint32
		fileSystemFlags    uint32
	)

	lpRootPathNamePtr, err := syscall.BytePtrFromString(path[:3])
	if err != nil {
		fmt.Println("Error:", err)
	}

	ret, _, err := procGetVolumeInformationA.Call(
		uintptr(unsafe.Pointer(lpRootPathNamePtr)),
		0,
		0,
		uintptr(unsafe.Pointer(&volumeSerialNumber)),
		uintptr(unsafe.Pointer(&maxComponentLength)),
		uintptr(unsafe.Pointer(&fileSystemFlags)),
		0,
		0,
	)

	if ret == 0 {
		fmt.Println("Error:", err)
	}

	dongleKey := make([]byte, 4)
	binary.LittleEndian.PutUint32(dongleKey, volumeSerialNumber)
	// fmt.Println(dongleKey)

	result := hex.EncodeToString(dongleKey)

	// fmt.Println("Serial Number Hex:", result)
	return result
}
