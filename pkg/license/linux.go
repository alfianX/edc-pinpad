//go:build linux
// +build linux

package license

import (
	"fmt"
	"os"
	"strings"
)

// func GetVolume() string {
// 	var stat unix.Statfs_t
// 	err := unix.Statfs("/mnt", &stat)
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 		return ""
// 	}

// 	var volumeID uint64
// 	// Debug and adapt based on your platform's Fsid structure
// 	if len(stat.Fsid.Val) >= 2 {
// 		volumeID = uint64(stat.Fsid.Val[0])<<32 | uint64(stat.Fsid.Val[1])
// 	} else {
// 		fmt.Println("Fsid structure incompatible.")
// 		return ""
// 	}

// 	// Convert the volume ID to a hexadecimal string
// 	dongleKey := make([]byte, 8)
// 	binary.LittleEndian.PutUint64(dongleKey, volumeID)

// 	return hex.EncodeToString(dongleKey)[:8]
// }

func GetVolume() string {
	// 1. Pintu masuk khusus untuk Docker on Windows
	if dockerWinVol := os.Getenv("WINDOWS_HOST_VOLUME"); dockerWinVol != "" {
		return dockerWinVol
	}

	// 2. Jika bukan Docker on Windows (Native Linux / Docker on Linux VM)

	targetPath := "/etc/machine-id"

	// 3. Baca isi file machine-id dari path tersebut
	data, err := os.ReadFile(targetPath)
	if err != nil {
		fmt.Println("Error reading machine-id:", err)
		return ""
	}

	// 4. Bersihkan teks dari karakter spasi tambahan atau enter (newline)
	machineID := strings.TrimSpace(string(data))
	machineID = strings.ReplaceAll(machineID, "-", "") // Jaga-jaga jika ada tanda strip

	// 5. Pastikan panjang id memenuhi syarat sebelum dipotong (mencegah panic/error)
	if len(machineID) < 8 {
		fmt.Println("Error: Format machine-id tidak valid atau terlalu pendek")
		return ""
	}

	// 6. Kembalikan 8 karakter pertama sebagai license key
	return machineID[:8]
}
