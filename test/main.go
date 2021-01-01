package main

import (
	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/container"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/widget"
	"github.com/flopp/go-findfont"
	"log"
	"os"
	"runtime"
)

func main() {
	CheckOSSetPath()
	myApp := app.New()
	win := myApp.NewWindow("JD抢购神器")
	win.Resize(fyne.Size{400, 400})

	username := widget.NewEntry()
	password := widget.NewPasswordEntry()
	content := widget.NewForm(widget.NewFormItem("用户名", username), widget.NewFormItem("密码", password))
	container.NewVBox()
	dialog.ShowCustomConfirm("请扫描二维码登录JD", "确认", "取消", content, func(b bool) {
		if !b {
			win.Close()
		}

		log.Println("Please Authenticate", username.Text, password.Text)
	}, win)

	//win.SetContent(widget.NewEntry())
	win.ShowAndRun()
}

func CheckOSSetPath() {
	switch runtime.GOOS {
	case "linux":
		filePath, _ := findfont.Find("DroidSansFallbackFull.ttf")
		os.Setenv("FYNE_FONT", filePath)
	case "darwin":
		filePath, _ := findfont.Find("STHeiti Medium.ttc")
		os.Setenv("FYNE_FONT", filePath)
	case "windows":
		filePath, _ := findfont.Find("SIMHEI.TTF")
		os.Setenv("FYNE_FONT", filePath)
	}
}
