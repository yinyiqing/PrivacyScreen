package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unsafe"
)

const (
	className  = "ShadowDeskPrivacyScreenOverlay"
	windowName = "ShadowDesk Privacy Screen"

	wsPopup         = 0x80000000
	wsExLayered     = 0x00080000
	wsExNoActivate  = 0x08000000
	wsExTopmost     = 0x00000008
	wsExTransparent = 0x00000020
	wsExToolWindow  = 0x00000080

	lwaAlpha = 0x00000002

	swShow        = 5
	swpShowWindow = 0x0040

	hwndTopmost = ^uintptr(0)

	wmClose         = 0x0010
	wmDestroy       = 0x0002
	wmEraseBkgnd    = 0x0014
	wmPaint         = 0x000F
	wmNcHitTest     = 0x0084
	wmMouseActivate = 0x0021

	htTransparent = ^uintptr(0)
	maNoActivate  = 3

	smCxScreen       = 0
	smCyScreen       = 1
	smXVirtualScreen = 76
	smYVirtualScreen = 77
	smCxVirtual      = 78
	smCyVirtual      = 79

	wdaMonitor            = 0x00000001
	wdaExcludeFromCapture = 0x00000011
)

var (
	user32  = syscall.NewLazyDLL("user32.dll")
	gdi32   = syscall.NewLazyDLL("gdi32.dll")
	gdiplus = syscall.NewLazyDLL("gdiplus.dll")

	procRegisterClassExW      = user32.NewProc("RegisterClassExW")
	procCreateWindowExW       = user32.NewProc("CreateWindowExW")
	procDefWindowProcW        = user32.NewProc("DefWindowProcW")
	procDestroyWindow         = user32.NewProc("DestroyWindow")
	procDispatchMessageW      = user32.NewProc("DispatchMessageW")
	procFindWindowW           = user32.NewProc("FindWindowW")
	procGetMessageW           = user32.NewProc("GetMessageW")
	procGetModuleHandleW      = syscall.NewLazyDLL("kernel32.dll").NewProc("GetModuleHandleW")
	procGetSystemMetrics      = user32.NewProc("GetSystemMetrics")
	procLoadCursorW           = user32.NewProc("LoadCursorW")
	procPostMessageW          = user32.NewProc("PostMessageW")
	procPostQuitMessage       = user32.NewProc("PostQuitMessage")
	procSetWindowDisplayAff   = user32.NewProc("SetWindowDisplayAffinity")
	procSetLayeredWindowAttrs = user32.NewProc("SetLayeredWindowAttributes")
	procSetWindowPos          = user32.NewProc("SetWindowPos")
	procShowCursor            = user32.NewProc("ShowCursor")
	procShowWindow            = user32.NewProc("ShowWindow")
	procTranslateMessage      = user32.NewProc("TranslateMessage")
	procUpdateWindow          = user32.NewProc("UpdateWindow")
	procValidateRect          = user32.NewProc("ValidateRect")
	procBeginPaint            = user32.NewProc("BeginPaint")
	procEndPaint              = user32.NewProc("EndPaint")
	procGetStockObject        = gdi32.NewProc("GetStockObject")
	procFillRect              = user32.NewProc("FillRect")
	procGetClientRect         = user32.NewProc("GetClientRect")
	procGetDC                 = user32.NewProc("GetDC")
	procReleaseDC             = user32.NewProc("ReleaseDC")
	procGdiplusStartup        = gdiplus.NewProc("GdiplusStartup")
	procGdiplusShutdown       = gdiplus.NewProc("GdiplusShutdown")
	procGdipLoadImageFromFile = gdiplus.NewProc("GdipLoadImageFromFile")
	procGdipGetImageWidth     = gdiplus.NewProc("GdipGetImageWidth")
	procGdipGetImageHeight    = gdiplus.NewProc("GdipGetImageHeight")
	procGdipCreateFromHDC     = gdiplus.NewProc("GdipCreateFromHDC")
	procGdipDrawImageRectI    = gdiplus.NewProc("GdipDrawImageRectI")
	procGdipDeleteGraphics    = gdiplus.NewProc("GdipDeleteGraphics")
	procGdipDisposeImage      = gdiplus.NewProc("GdipDisposeImage")

	clickThroughHitTest bool
	overlayImage        *gdipImage
	overlayImageMode    string
	gdiplusToken        uintptr
)

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type point struct {
	X int32
	Y int32
}

type msg struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type paintStruct struct {
	Hdc       uintptr
	Erase     int32
	Paint     rect
	Restore   int32
	IncUpdate int32
	Reserved  [32]byte
}

type gdiplusStartupInput struct {
	GdiplusVersion           uint32
	DebugEventCallback       uintptr
	SuppressBackgroundThread int32
	SuppressExternalCodecs   int32
}

type gdipImage struct {
	Handle uintptr
	Width  int32
	Height int32
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "on":
		runOn(os.Args[2:])
	case "off":
		if closeExisting() {
			fmt.Println("privacy screen closed")
		} else {
			fmt.Println("privacy screen is not running")
			os.Exit(1)
		}
	case "status":
		if findOverlay() != 0 {
			fmt.Println("privacy screen is running")
		} else {
			fmt.Println("privacy screen is not running")
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  privacy-screen.exe on [--timeout 60] [--image path] [--image-mode stretch|fit] [--exclude-from-capture=true] [--click-through=true] [--hide-cursor=true]")
	fmt.Println("  privacy-screen.exe off")
	fmt.Println("  privacy-screen.exe status")
}

func runOn(args []string) {
	fs := flag.NewFlagSet("on", flag.ExitOnError)
	timeout := fs.Int("timeout", 60, "seconds before the privacy screen closes automatically; 0 disables auto close")
	excludeFromCapture := fs.Bool("exclude-from-capture", true, "hide the black overlay from supported screen capture APIs")
	clickThrough := fs.Bool("click-through", true, "let mouse input pass through the overlay to the real desktop")
	hideCursor := fs.Bool("hide-cursor", true, "hide the local cursor while the privacy screen is active")
	imagePath := fs.String("image", "", "image file shown on the privacy screen; png, jpg, jpeg, and gif are supported")
	imageMode := fs.String("image-mode", "stretch", "image display mode: stretch or fit")
	_ = fs.Parse(args)

	overlayImage = nil
	overlayImageMode = *imageMode
	if *imagePath != "" {
		if err := startGDIPlus(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to start GDI+, fallback to black screen: %v\n", err)
		}
		img, err := loadGDIPlusImage(*imagePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to load privacy screen image, fallback to black screen: %v\n", err)
		} else {
			overlayImage = img
			defer disposeGDIPlusImage(overlayImage)
		}
	}
	if gdiplusToken != 0 {
		defer stopGDIPlus()
	}

	if findOverlay() != 0 {
		fmt.Fprintln(os.Stderr, "privacy screen is already running")
		os.Exit(1)
	}

	hwnd, err := createOverlayWindow(*clickThrough)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *excludeFromCapture {
		if err := setCaptureExclusion(hwnd); err != nil {
			destroyWindow(hwnd)
			fmt.Fprintf(os.Stderr, "failed to exclude overlay from capture: %v\n", err)
			os.Exit(1)
		}
	}

	if *hideCursor {
		hideSystemCursor()
		defer showSystemCursor()
	}

	showOverlay(hwnd)
	if overlayImage != nil {
		fmt.Println("privacy screen is on with image")
	} else {
		fmt.Println("privacy screen is on with black screen")
	}

	if *timeout > 0 {
		go func() {
			time.Sleep(time.Duration(*timeout) * time.Second)
			postClose(hwnd)
		}()
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		postClose(hwnd)
	}()

	messageLoop()
	fmt.Println("privacy screen is off")
}

func createOverlayWindow(clickThrough bool) (uintptr, error) {
	clickThroughHitTest = clickThrough
	instance, _, _ := procGetModuleHandleW.Call(0)
	classNamePtr := syscall.StringToUTF16Ptr(className)
	cursor, _, _ := procLoadCursorW.Call(0, uintptr(32512))
	blackBrush, _, _ := procGetStockObject.Call(4)

	wc := wndClassEx{
		Size:       uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:    syscall.NewCallback(windowProc),
		Instance:   instance,
		Cursor:     cursor,
		Background: blackBrush,
		ClassName:  classNamePtr,
	}

	if atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); atom == 0 {
		if errno, ok := err.(syscall.Errno); !ok || errno != 1410 {
			return 0, fmt.Errorf("failed to register window class: %v", err)
		}
	}

	x, y, width, height := virtualScreenBounds()
	exStyle := uintptr(wsExTopmost | wsExToolWindow)
	if clickThrough {
		exStyle |= uintptr(wsExLayered | wsExTransparent | wsExNoActivate)
	}

	hwnd, _, err := procCreateWindowExW.Call(
		exStyle,
		uintptr(unsafe.Pointer(classNamePtr)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(windowName))),
		uintptr(wsPopup),
		uintptr(int32ToIntArg(x)),
		uintptr(int32ToIntArg(y)),
		uintptr(int32ToIntArg(width)),
		uintptr(int32ToIntArg(height)),
		0,
		0,
		instance,
		0,
	)
	if hwnd == 0 {
		return 0, fmt.Errorf("failed to create overlay window: %v", err)
	}
	if clickThrough {
		if ok, _, err := procSetLayeredWindowAttrs.Call(hwnd, 0, 255, uintptr(lwaAlpha)); ok == 0 {
			destroyWindow(hwnd)
			return 0, fmt.Errorf("failed to configure click-through overlay: %v", err)
		}
	}
	return hwnd, nil
}

func virtualScreenBounds() (int32, int32, int32, int32) {
	x := getMetric(smXVirtualScreen)
	y := getMetric(smYVirtualScreen)
	width := getMetric(smCxVirtual)
	height := getMetric(smCyVirtual)
	if width <= 0 || height <= 0 {
		x = 0
		y = 0
		width = getMetric(smCxScreen)
		height = getMetric(smCyScreen)
	}
	if width <= 0 || height <= 0 {
		width = 1
		height = 1
	}
	return x, y, width, height
}

func getMetric(index int32) int32 {
	ret, _, _ := procGetSystemMetrics.Call(uintptr(index))
	return int32(ret)
}

func setCaptureExclusion(hwnd uintptr) error {
	if ok, _, err := procSetWindowDisplayAff.Call(hwnd, uintptr(wdaExcludeFromCapture)); ok != 0 {
		return nil
	} else if err != syscall.Errno(0) {
		return err
	}

	if ok, _, err := procSetWindowDisplayAff.Call(hwnd, uintptr(wdaMonitor)); ok != 0 {
		return fmt.Errorf("WDA_EXCLUDEFROMCAPTURE unsupported; WDA_MONITOR fallback was set")
	} else if err != syscall.Errno(0) {
		return err
	}

	return fmt.Errorf("SetWindowDisplayAffinity failed")
}

func showOverlay(hwnd uintptr) {
	x, y, width, height := virtualScreenBounds()
	procSetWindowPos.Call(
		hwnd,
		hwndTopmost,
		uintptr(int32ToIntArg(x)),
		uintptr(int32ToIntArg(y)),
		uintptr(int32ToIntArg(width)),
		uintptr(int32ToIntArg(height)),
		uintptr(swpShowWindow),
	)
	procShowWindow.Call(hwnd, swShow)
	paintBlack(hwnd)
	procUpdateWindow.Call(hwnd)
}

func messageLoop() {
	var m msg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func windowProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case wmNcHitTest:
		if clickThroughHitTest {
			return htTransparent
		}
		ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
		return ret
	case wmMouseActivate:
		if clickThroughHitTest {
			return maNoActivate
		}
		ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
		return ret
	case wmEraseBkgnd:
		paintOverlayDC(hwnd, wParam)
		return 1
	case wmPaint:
		var ps paintStruct
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		if hdc != 0 {
			paintOverlayDC(hwnd, hdc)
			procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		}
		procValidateRect.Call(hwnd, 0)
		return 0
	case wmClose:
		destroyWindow(hwnd)
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	default:
		ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
		return ret
	}
}

func paintBlack(hwnd uintptr) {
	dc, _, _ := procGetDC.Call(hwnd)
	if dc == 0 {
		return
	}
	defer procReleaseDC.Call(hwnd, dc)
	paintOverlayDC(hwnd, dc)
}

func paintOverlayDC(hwnd uintptr, dc uintptr) {
	var r rect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	blackBrush, _, _ := procGetStockObject.Call(4)
	procFillRect.Call(dc, uintptr(unsafe.Pointer(&r)), blackBrush)

	if overlayImage == nil || overlayImage.Handle == 0 {
		return
	}

	dstX, dstY, dstW, dstH := imageDestination(r, overlayImage.Width, overlayImage.Height, overlayImageMode)
	drawGDIPlusImage(dc, overlayImage, dstX, dstY, dstW, dstH)
}

func imageDestination(r rect, imageWidth, imageHeight int32, mode string) (int32, int32, int32, int32) {
	screenW := r.Right - r.Left
	screenH := r.Bottom - r.Top
	if screenW <= 0 || screenH <= 0 || imageWidth <= 0 || imageHeight <= 0 {
		return 0, 0, screenW, screenH
	}
	if mode != "fit" {
		return 0, 0, screenW, screenH
	}

	sw := float64(screenW)
	sh := float64(screenH)
	iw := float64(imageWidth)
	ih := float64(imageHeight)
	scale := sw / iw
	if ih*scale > sh {
		scale = sh / ih
	}
	dstW := int32(iw * scale)
	dstH := int32(ih * scale)
	dstX := (screenW - dstW) / 2
	dstY := (screenH - dstH) / 2
	if dstW <= 0 {
		dstW = 1
	}
	if dstH <= 0 {
		dstH = 1
	}
	return dstX, dstY, dstW, dstH
}

func startGDIPlus() error {
	if gdiplusToken != 0 {
		return nil
	}
	input := gdiplusStartupInput{GdiplusVersion: 1}
	status, _, err := procGdiplusStartup.Call(
		uintptr(unsafe.Pointer(&gdiplusToken)),
		uintptr(unsafe.Pointer(&input)),
		0,
	)
	if status != 0 {
		return fmt.Errorf("GdiplusStartup failed: status=%d err=%v", status, err)
	}
	return nil
}

func stopGDIPlus() {
	if gdiplusToken == 0 {
		return
	}
	procGdiplusShutdown.Call(gdiplusToken)
	gdiplusToken = 0
}

func loadGDIPlusImage(path string) (*gdipImage, error) {
	var handle uintptr
	status, _, err := procGdipLoadImageFromFile.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path))),
		uintptr(unsafe.Pointer(&handle)),
	)
	if status != 0 || handle == 0 {
		return nil, fmt.Errorf("GdipLoadImageFromFile failed: status=%d err=%v", status, err)
	}

	var width uint32
	status, _, err = procGdipGetImageWidth.Call(handle, uintptr(unsafe.Pointer(&width)))
	if status != 0 || width == 0 {
		procGdipDisposeImage.Call(handle)
		return nil, fmt.Errorf("GdipGetImageWidth failed: status=%d err=%v", status, err)
	}

	var height uint32
	status, _, err = procGdipGetImageHeight.Call(handle, uintptr(unsafe.Pointer(&height)))
	if status != 0 || height == 0 {
		procGdipDisposeImage.Call(handle)
		return nil, fmt.Errorf("GdipGetImageHeight failed: status=%d err=%v", status, err)
	}

	return &gdipImage{
		Handle: handle,
		Width:  int32(width),
		Height: int32(height),
	}, nil
}

func disposeGDIPlusImage(img *gdipImage) {
	if img == nil || img.Handle == 0 {
		return
	}
	procGdipDisposeImage.Call(img.Handle)
	img.Handle = 0
}

func drawGDIPlusImage(dc uintptr, img *gdipImage, x, y, width, height int32) {
	var graphics uintptr
	status, _, _ := procGdipCreateFromHDC.Call(dc, uintptr(unsafe.Pointer(&graphics)))
	if status != 0 || graphics == 0 {
		return
	}
	defer procGdipDeleteGraphics.Call(graphics)

	procGdipDrawImageRectI.Call(
		graphics,
		img.Handle,
		uintptr(int32ToIntArg(x)),
		uintptr(int32ToIntArg(y)),
		uintptr(int32ToIntArg(width)),
		uintptr(int32ToIntArg(height)),
	)
}

func findOverlay() uintptr {
	hwnd, _, _ := procFindWindowW.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(className))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(windowName))),
	)
	return hwnd
}

func closeExisting() bool {
	hwnd := findOverlay()
	if hwnd == 0 {
		return false
	}
	postClose(hwnd)
	return true
}

func postClose(hwnd uintptr) {
	procPostMessageW.Call(hwnd, wmClose, 0, 0)
}

func destroyWindow(hwnd uintptr) {
	procDestroyWindow.Call(hwnd)
}

func hideSystemCursor() {
	for i := 0; i < 8; i++ {
		ret, _, _ := procShowCursor.Call(0)
		if int32(ret) < 0 {
			return
		}
	}
}

func showSystemCursor() {
	for i := 0; i < 8; i++ {
		ret, _, _ := procShowCursor.Call(1)
		if int32(ret) >= 0 {
			return
		}
	}
}

func int32ToIntArg(v int32) int64 {
	return int64(v)
}
