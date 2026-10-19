//go:build windows

package announcements

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/updater"
)

var (
	modkernel32 = windows.NewLazySystemDLL("kernel32.dll")
	modwtsapi32 = windows.NewLazySystemDLL("wtsapi32.dll")
	moduserenv  = windows.NewLazySystemDLL("userenv.dll")

	procWTSGetActiveConsoleSessionId = modkernel32.NewProc("WTSGetActiveConsoleSessionId")
	procWTSQueryUserToken            = modwtsapi32.NewProc("WTSQueryUserToken")
	procCreateEnvironmentBlock       = moduserenv.NewProc("CreateEnvironmentBlock")
	procDestroyEnvironmentBlock      = moduserenv.NewProc("DestroyEnvironmentBlock")
)

// startPopup tries to launch the popup in the active user's session so it is
// visible when the agent is running as a service. If that fails (for example,
// when the agent is running interactively or there is no logged-on user), it
// falls back to a detached child process.
func startPopup(popupPath string, announcementFile string) error {
	args := []string{"--file", announcementFile}
	cmd := exec.Command(popupPath, args...)

	if err := startPopupInUserSession(popupPath, announcementFile); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start popup in user session: %v; falling back to detached\n", err)
		return updater.StartDetached(cmd)
	}
	return nil
}

func startPopupInUserSession(popupPath string, announcementFile string) error {
	sessionID, err := getActiveConsoleSessionID()
	if err != nil {
		return err
	}

	var userToken windows.Token
	if err := wtsQueryUserToken(sessionID, &userToken); err != nil {
		return fmt.Errorf("WTSQueryUserToken: %w", err)
	}
	defer userToken.Close()

	var envBlock *uint16
	if err := createEnvironmentBlock(&envBlock, userToken, false); err != nil {
		return fmt.Errorf("CreateEnvironmentBlock: %w", err)
	}
	defer destroyEnvironmentBlock(envBlock)

	cmdLine := fmt.Sprintf(`"%s" --file "%s"`, popupPath, announcementFile)

	var si windows.StartupInfo
	si.Cb = uint32(unsafe.Sizeof(si))
	var pi windows.ProcessInformation

	err = windows.CreateProcessAsUser(
		userToken,
		nil,
		windows.StringToUTF16Ptr(cmdLine),
		nil,
		nil,
		false,
		windows.CREATE_UNICODE_ENVIRONMENT|windows.DETACHED_PROCESS,
		envBlock,
		windows.StringToUTF16Ptr(filepath.Dir(popupPath)),
		&si,
		&pi,
	)
	if err != nil {
		return fmt.Errorf("CreateProcessAsUser: %w", err)
	}
	windows.CloseHandle(pi.Process)
	windows.CloseHandle(pi.Thread)
	return nil
}

func getActiveConsoleSessionID() (uint32, error) {
	r, _, _ := procWTSGetActiveConsoleSessionId.Call()
	if r == 0xFFFFFFFF {
		return 0, fmt.Errorf("no active console session")
	}
	return uint32(r), nil
}

func wtsQueryUserToken(sessionID uint32, token *windows.Token) error {
	r, _, e := procWTSQueryUserToken.Call(uintptr(sessionID), uintptr(unsafe.Pointer(token)))
	if r == 0 {
		return e
	}
	return nil
}

func createEnvironmentBlock(block **uint16, token windows.Token, inherit bool) error {
	var inheritInt uintptr
	if inherit {
		inheritInt = 1
	}
	r, _, e := procCreateEnvironmentBlock.Call(uintptr(unsafe.Pointer(block)), uintptr(token), inheritInt)
	if r == 0 {
		return e
	}
	return nil
}

func destroyEnvironmentBlock(block *uint16) error {
	r, _, e := procDestroyEnvironmentBlock.Call(uintptr(unsafe.Pointer(block)))
	if r == 0 {
		return e
	}
	return nil
}
