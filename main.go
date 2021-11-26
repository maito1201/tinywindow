package main

import (
	"log"
	"syscall"
	"unsafe"
)

const className = "testClass"

const (
	cSW_USE_DEFAULT      = 0x80000000
	cWS_VISIBLE          = 0x10000000
	cWS_OVERLAPPEDWINDOW = 0x00CF0000

	cWM_DESTROY = 0x0002
	cWM_CLOSE   = 0x0010
	cWM_PAINT   = 0x000F

	cIDC_ARROW    = 32512
	cCOLOR_WINDOW = 5
)

type tPOINT struct {
	x, y int32
}

type tMSG struct {
	hwnd    syscall.Handle
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      tPOINT
}

type tRECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type tPAINTSTRUCT struct {
	hdc         syscall.Handle
	fErace      uint32
	rcPaint     tRECT
	fRestore    uint32
	fIncUpdate  uint32
	rgbReserved byte
}

type tWNDCLASSEXW struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   syscall.Handle
	icon       syscall.Handle
	cursor     syscall.Handle
	background syscall.Handle
	menuName   *uint16
	className  *uint16
	iconSm     syscall.Handle
}

func main() {
	instance, err := getModuleHandle()
	if err != nil {
		log.Println(err)
		return
	}

	cursor, err := loadCursorResource(cIDC_ARROW)
	if err != nil {
		log.Println(err)
		return
	}

	wcx := tWNDCLASSEXW{
		wndProc:    syscall.NewCallback(wndProc),
		instance:   instance,
		cursor:     cursor,
		background: cCOLOR_WINDOW + 2,
		className:  syscall.StringToUTF16Ptr(className),
	}
	wcx.size = uint32(unsafe.Sizeof(wcx))

	if _, err = registerClassEx(&wcx); err != nil {
		log.Println(err)
		return
	}

	_, err = createWindow(
		className,
		"Test Window",
		cWS_VISIBLE|cWS_OVERLAPPEDWINDOW,
		cSW_USE_DEFAULT,
		cSW_USE_DEFAULT,
		cSW_USE_DEFAULT,
		cSW_USE_DEFAULT,
		0,
		0,
		instance,
	)
	if err != nil {
		log.Println(err)
		return
	}

	for {
		msg := tMSG{}
		gotMessage, err := getMessage(&msg, 0, 0, 0)
		if err != nil {
			log.Println(err)
			return
		}
		if gotMessage {
			translateMessage(&msg)
			dispatchMessage(&msg)
		} else {
			break
		}
	}
}

func wndProc(hwnd syscall.Handle, msg uint32, wparam, lparam uintptr) uintptr {
	switch msg {
	case cWM_CLOSE:
		destroyWindow(hwnd)
	case cWM_DESTROY:
		postQuitMessage(0)
	case cWM_PAINT:
		var ps tPAINTSTRUCT
		hdc := beginPaint(hwnd, &ps)
		_, err := textOut(hdc, "Hello Window API")
		if err != nil {
			panic(err)
		}
		endPaint(hdc, &ps)
		return 0
	default:
		ret := defWindowProc(hwnd, msg, wparam, lparam)
		return ret
	}
	return 0
}

////////////////
// user32.dll
////////////////

var (
	user32            = syscall.NewLazyDLL("user32.dll")
	pCreateWindowExW  = user32.NewProc("CreateWindowExW")
	pDefWindowProcW   = user32.NewProc("DefWindowProcW")
	pDestroyWindow    = user32.NewProc("DestroyWindow")
	pDispatchMessageW = user32.NewProc("DispatchMessageW")
	pGetMessageW      = user32.NewProc("GetMessageW")
	pLoadCursorW      = user32.NewProc("LoadCursorW")
	pPostQuitMessage  = user32.NewProc("PostQuitMessage")
	pRegisterClassExW = user32.NewProc("RegisterClassExW")
	pTranslateMessage = user32.NewProc("TranslateMessage")
	pBeginPaint       = user32.NewProc("BeginPaint")
	pEndPaint         = user32.NewProc("EndPaint")
)

func createWindow(className, windowName string, style uint32, x, y, width, height uint32, parent, menu, instance syscall.Handle) (syscall.Handle, error) {
	ret, _, err := pCreateWindowExW.Call(
		uintptr(0),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(className))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(windowName))),
		uintptr(style),
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		uintptr(parent),
		uintptr(menu),
		uintptr(instance),
		uintptr(0),
	)
	if ret == 0 {
		return 0, err
	}
	return syscall.Handle(ret), nil
}

func defWindowProc(hwnd syscall.Handle, msg uint32, wparam, lparam uintptr) uintptr {
	ret, _, _ := pDefWindowProcW.Call(
		uintptr(hwnd),
		uintptr(msg),
		uintptr(wparam),
		uintptr(lparam),
	)
	return uintptr(ret)
}

func destroyWindow(hwnd syscall.Handle) error {
	ret, _, err := pDestroyWindow.Call(uintptr(hwnd))
	if ret == 0 {
		return err
	}
	return nil
}

func beginPaint(hwnd syscall.Handle, p *tPAINTSTRUCT) syscall.Handle {

	ret, _, _ := pBeginPaint.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(p)),
	)
	return syscall.Handle(ret)
}

func endPaint(hwnd syscall.Handle, p *tPAINTSTRUCT) syscall.Handle {

	ret, _, _ := pEndPaint.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(p)),
	)
	return syscall.Handle(ret)
}

func registerClassEx(wcx *tWNDCLASSEXW) (uint16, error) {
	ret, _, err := pRegisterClassExW.Call(
		uintptr(unsafe.Pointer(wcx)),
	)
	if ret == 0 {
		return 0, err
	}
	return uint16(ret), nil
}

func translateMessage(msg *tMSG) {
	pTranslateMessage.Call(uintptr(unsafe.Pointer(msg)))
}

func dispatchMessage(msg *tMSG) {
	pDispatchMessageW.Call(uintptr(unsafe.Pointer(msg)))
}

func loadCursorResource(cursorName uint32) (syscall.Handle, error) {
	ret, _, err := pLoadCursorW.Call(
		uintptr(0),
		uintptr(uint16(cursorName)),
	)
	if ret == 0 {
		return 0, err
	}
	return syscall.Handle(ret), nil
}

func postQuitMessage(exitCode int32) {
	pPostQuitMessage.Call(uintptr(exitCode))
}

func getMessage(msg *tMSG, hwnd syscall.Handle, msgFilterMin, msgFilterMax uint32) (bool, error) {
	ret, _, err := pGetMessageW.Call(
		uintptr(unsafe.Pointer(msg)),
		uintptr(hwnd),
		uintptr(msgFilterMin),
		uintptr(msgFilterMax),
	)
	if int32(ret) == -1 {
		return false, err
	}
	return int32(ret) != 0, nil
}

//////////////
// dgi32.dll
//////////////

var (
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	pTextOut = gdi32.NewProc("TextOutW")
)

func textOut(hwnd syscall.Handle, text string) (syscall.Handle, error) {

	ret, _, _ := pTextOut.Call(
		uintptr(hwnd),
		uintptr(0),
		uintptr(0),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))),
		uintptr(len(text)),
	)
	return syscall.Handle(ret), nil
}

/////////////////
// kernel32.dll
/////////////////

var (
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
)

func getModuleHandle() (syscall.Handle, error) {
	ret, _, err := pGetModuleHandleW.Call(uintptr(0))
	if ret == 0 {
		return 0, err
	}
	return syscall.Handle(ret), nil
}
