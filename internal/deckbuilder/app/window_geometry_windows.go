//go:build windows

package deckbuilder

import (
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
	"golang.org/x/sys/windows"
)

const (
	spiGetWorkArea          = 0x0030
	monitorDefaultToNearest = 2
)

type windowsRECT struct {
	Left, Top, Right, Bottom int32
}

type windowsMonitorInfo struct {
	CbSize    uint32
	RcMonitor windowsRECT
	RcWork    windowsRECT
	DwFlags   uint32
}

var (
	user32                   = windows.NewLazySystemDLL("user32.dll")
	procSystemParametersInfo = user32.NewProc("SystemParametersInfoW")
	procMonitorFromWindow    = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfo       = user32.NewProc("GetMonitorInfoW")
	procGetWindowRect        = user32.NewProc("GetWindowRect")
)

func desktopWorkAreaDIPPlatform() fyne.Size {
	var area windowsRECT
	ret, _, _ := procSystemParametersInfo.Call(
		uintptr(spiGetWorkArea),
		0,
		uintptr(unsafe.Pointer(&area)),
		0,
	)
	if ret == 0 {
		return fyne.Size{}
	}
	return workAreaPixelsToDIP(area, 1)
}

func clampWindowToWorkAreaPlatform(window fyne.Window) {
	if window == nil {
		return
	}
	native, ok := window.(driver.NativeWindow)
	if !ok {
		return
	}

	var (
		hwnd     uintptr
		work     windowsRECT
		haveWork bool
		winRect  windowsRECT
		haveWin  bool
	)
	native.RunNative(func(context any) {
		winContext, ok := context.(driver.WindowsWindowContext)
		if !ok || winContext.HWND == 0 {
			return
		}
		hwnd = winContext.HWND
		if ret, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&winRect))); ret != 0 {
			haveWin = true
		}
		monitor, _, _ := procMonitorFromWindow.Call(hwnd, uintptr(monitorDefaultToNearest))
		if monitor == 0 {
			return
		}
		info := windowsMonitorInfo{CbSize: uint32(unsafe.Sizeof(windowsMonitorInfo{}))}
		if ret, _, _ := procGetMonitorInfo.Call(monitor, uintptr(unsafe.Pointer(&info))); ret != 0 {
			work = info.RcWork
			haveWork = true
		}
	})
	if !haveWork {
		return
	}

	scale := float32(1)
	if canvas := window.Canvas(); canvas != nil {
		if canvasScale := canvas.Scale(); canvasScale > 0 {
			scale = canvasScale
		}
	}
	maxSize := workAreaPixelsToDIP(work, scale)
	if maxSize.Width <= 0 || maxSize.Height <= 0 {
		return
	}

	current := window.Canvas().Size()
	target := clampSizeToBounds(current, maxSize)

	// If the outer window already sits inside the work area and the canvas fits,
	// leave maximized windows alone.
	if haveWin &&
		winRect.Left >= work.Left &&
		winRect.Top >= work.Top &&
		winRect.Right <= work.Right &&
		winRect.Bottom <= work.Bottom &&
		current.Width <= maxSize.Width+0.5 &&
		current.Height <= maxSize.Height+0.5 {
		return
	}

	if target != current {
		window.Resize(target)
	}
}

func workAreaPixelsToDIP(area windowsRECT, scale float32) fyne.Size {
	if scale <= 0 {
		scale = 1
	}
	widthPx := float32(area.Right-area.Left) - windowChromeWidthPx
	heightPx := float32(area.Bottom-area.Top) - windowChromeHeightPx
	if widthPx < 1 || heightPx < 1 {
		return fyne.Size{}
	}
	return fyne.NewSize(widthPx/scale, heightPx/scale)
}
