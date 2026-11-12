package license

import (
	"bytes"
	"context"
	_ "embed"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed license.wasm
var licenseWasm []byte

// Jadikan variabel global agar di-cache di memori
var (
	wazeroRuntime wazero.Runtime
	compiledWasm  wazero.CompiledModule
)

// Panggil fungsi ini SATU KALI SAJA saat aplikasi baru menyala (di fungsi init atau main)
func InitWasmEngine() error {
	ctx := context.Background()
	wazeroRuntime = wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, wazeroRuntime)

	var err error
	// Compile modul Wasm ke machine code sekali saja dan simpan di cache
	compiledWasm, err = wazeroRuntime.CompileModule(ctx, licenseWasm)
	return err
}

func ValidateLicense(key string, volumesn string) (string, error) {
	ctx := context.Background()
	var stdoutBuf bytes.Buffer

	currentDate := time.Now().Format("20060102")

	config := wazero.NewModuleConfig().
		WithStdout(&stdoutBuf).
		WithArgs("license.wasm", key, volumesn, currentDate)

	// Gunakan InstantiateModule (mengeksekusi dari cache), bukan InstantiateWithConfig
	_, err := wazeroRuntime.InstantiateModule(ctx, compiledWasm, config)
	if err != nil {
		return "", err
	}

	return stdoutBuf.String(), nil
}
