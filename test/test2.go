package main

import (
	"SnapUp/pkg/jd_tools"
	"fmt"
	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/theme"
)

func main() {
	myApp := app.New()
	w := myApp.NewWindow("Image")
	a := fyne.CurrentApp()
	//image := canvas.NewImageFromResource(theme.FyneLogo())
	var tools jd_tools.JdInfo
	qrcode, _ := tools.GetLoginQrcode()
	image1 := canvas.NewImageFromImage(qrcode)
	// image := canvas.NewImageFromFile(fileName)
	// image := canvas.NewImageFromImage(src)
	image1.FillMode = canvas.ImageFillOriginal
	a.Settings().SetTheme(theme.LightTheme())
	w.SetContent(image1)

	w.ShowAndRun()
	err := tools.CheckLogin()
	if err != nil {
		fmt.Println(err)
	}
}
