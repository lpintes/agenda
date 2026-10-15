package auth

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// entropy ties the encrypted data to this application, so other programs
// running as the same user cannot decrypt it with a plain DPAPI call.
var entropy = []byte("github.com/lpintes/agenda token")

// protect encrypts data with DPAPI for the current user.
func protect(data []byte) ([]byte, error) {
	var out windows.DataBlob
	err := windows.CryptProtectData(blob(data), nil, blob(entropy), 0, nil,
		windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	if err != nil {
		return nil, err
	}
	return takeBlob(&out), nil
}

// unprotect decrypts data encrypted by protect.
func unprotect(data []byte) ([]byte, error) {
	var out windows.DataBlob
	err := windows.CryptUnprotectData(blob(data), nil, blob(entropy), 0, nil,
		windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	if err != nil {
		return nil, err
	}
	return takeBlob(&out), nil
}

func blob(b []byte) *windows.DataBlob {
	if len(b) == 0 {
		return &windows.DataBlob{}
	}
	return &windows.DataBlob{Size: uint32(len(b)), Data: &b[0]}
}

// takeBlob copies a blob allocated by DPAPI and frees it.
func takeBlob(b *windows.DataBlob) []byte {
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(b.Data)))
	return append([]byte(nil), unsafe.Slice(b.Data, b.Size)...)
}
