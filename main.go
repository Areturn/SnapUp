package main

import (
	"SnapUp/data/img"
	"SnapUp/window"
	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"
	"github.com/flopp/go-findfont"
	"os"
	"runtime"
	"time"
)

type platFormId int

const (
	Jd platFormId = iota + 1
	Sn
	Tb
)

var iconImg fyne.Resource

func main() {
	CheckOsSetPath()
	a := app.NewWithID("SnapUp")
	//a.Settings().SetTheme(theme.LightTheme())
	iconImg = img.Qiang
	a.SetIcon(iconImg)
	w := a.NewWindow("抢购神器-登录选择")
	//w.SetFixedSize(true)
	w.SetMaster()
	var id platFormId

	themes := fyne.NewContainerWithLayout(layout.NewGridLayout(3),
		widget.NewButton("京东", func() {
			id = Jd
			err := window.JdEntrance(a, w)
			if err != nil {
				panic(err)
			}
		}),
		widget.NewButton("苏宁", func() {
			id = Sn
			notSupport()
		}),
		widget.NewButton("淘宝", func() {
			id = Tb
			notSupport()
		}),
	)
	w.SetContent(themes)
	go func() {
		for {
			if time.Now().Unix() >= 1611331200 {
				//w.SetContent()
				//custom := dialog.NewCustom("失效提醒", "确认", widget.NewLabel("软件已过有效期，如有需要联系作者获取"), w)
				//information := dialog.NewInformation("失效提醒", "软件已过有效期，如有需要联系作者获取", w)
				//information.SetDismissText("确认")
				//information.SetOnClosed(func() {
				//	w.Close()
				//})
				dialog.NewCustomConfirm("失效提醒", "确认", "退出", widget.NewLabel("软件已过有效期，如有需要联系作者获取"), func(b bool) {
					w.Close()
				}, w).Show()
				w.Resize(fyne.NewSize(480, 240))
				return
			}

			time.Sleep(1 * time.Second)
		}
	}()
	w.CenterOnScreen()
	w.ShowAndRun()
}

func CheckOsSetPath() {
	switch runtime.GOOS {
	case "linux":
		filePath, _ := findfont.Find("DroidSansFallbackFull.ttf")
		_ = os.Setenv("FYNE_FONT", filePath)
	case "darwin":
		filePath, _ := findfont.Find("STHeiti Medium.ttc")
		_ = os.Setenv("FYNE_FONT", filePath)
	case "windows":
		filePath, _ := findfont.Find("SIMHEI.TTF")
		_ = os.Setenv("FYNE_FONT", filePath)
	}
}

func notSupport() {
	w := fyne.CurrentApp().NewWindow("提示")
	w.SetContent(fyne.NewContainerWithLayout(layout.NewCenterLayout(), widget.NewLabel("暂不支持")))
	w.Resize(fyne.NewSize(100, 100))
	w.SetFixedSize(true)
	w.Show()
}
