package window

import (
	"SnapUp/pkg/jdTools"
	"SnapUp/pkg/urlTools"
	"fmt"
	"fyne.io/fyne"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/container"
	"fyne.io/fyne/data/validation"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
	"image"
	"image/color"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var userName string

func JdEntrance(a fyne.App, w fyne.Window) (err error) {
	w.SetTitle("抢购神器-京东模式")
	w.SetContent(container.NewVBox(widget.NewLabel("正在打开登录窗口...")))
	tools := jdTools.Init()
	// 检查登录cookies
	n := 3
	for {
		code, err1 := tools.CheckCookies()
		if code == 0 && err1 != nil {
			if n == 0 {
				break
			}
			n--
			continue
		} else if code == 200 {
			parse, _ := url.Parse("https://passport.jd.com")
			cookies := tools.CookiesJar.Cookies(parse)
			//fmt.Println(cookies)
			for _, v := range cookies {
				if v.Name == "unick" {
					userName = v.Value
				}
			}
			break
		} else {
			break
		}
	}
	if tools.LoginStatus {
		confirm := dialog.NewConfirm("检测到已登录缓存", fmt.Sprintf("用户名: %s", userName), func(b bool) {
			if !b {
				err = jdLogin(tools, a, w)
			} else {
				err = jdPage(tools, a, w)
			}
			//fmt.Println(reLogin)
		}, w)
		confirm.SetConfirmText("直接登录")
		confirm.SetDismissText("重新登录")
		w.Resize(fyne.Size{400, 400})
		w.SetContent(container.NewVBox(widget.NewLabel("")))
		confirm.Show()
	} else {
		//reLogin = true
		err = jdLogin(tools, a, w)
	}
	return
}

func jdLogin(tools *jdTools.JdInfo, a fyne.App, w fyne.Window) (err error) {
	var qrcode image.Image
	w.Resize(fyne.Size{147, 147})
	center := container.NewVBox(widget.NewLabel("二维码加载中..."))
	w.SetContent(center)
	var getQrcodeStatus bool
	getQrcode := func() {
		var err1 error
		var n = 3
		for {
			if n == 0 {
				errPrompt(w, err1, "获取二维码异常")
				break
			}
			qrcode, err1 = tools.GetLoginQrcode()
			n--
			if err1 == nil {
				getQrcodeStatus = true
				break
			}
		}
	}
	getQrcode()
	if getQrcodeStatus {
		login := func(s string) {
			image1 := canvas.NewImageFromImage(qrcode)
			image1.FillMode = canvas.ImageFillOriginal
			hSplit := container.NewVBox(image1, widget.NewLabel(s))
			center = container.NewCenter(hSplit)
			a.Settings().SetTheme(theme.LightTheme())
			w.SetContent(center)
		}
		var loginCheck bool
		go func() {
			login("手机app扫描二维码登录")
			for {
				code, err2 := tools.CheckLogin()
				if code == 203 {
					getQrcode()
					login("二维码过期，请重新扫描")
				} else if code == 205 {
					getQrcode()
					login("二维码已取消授权,请重新扫描")
				} else if code == 300 {
					getQrcode()
					login(fmt.Sprintf("二维码校验异常，请重新扫描\n%s", err2))
				} else if code == 200 {
					loginCheck = true
					break
				} else {
					errPrompt(w, err2, "扫描二维码时出现未知异常")
					break
				}
			}
		}()
		go func() {
			for {
				if loginCheck {
					parse, _ := url.Parse("https://passport.jd.com")
					cookies := tools.CookiesJar.Cookies(parse)
					//fmt.Println(cookies)
					for _, v := range cookies {
						if v.Name == "unick" {
							userName = v.Value
						}
					}
					err = jdPage(tools, a, w)
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}
	return
}

func jdPage(tools *jdTools.JdInfo, a fyne.App, w fyne.Window) (err error) {
	a.Settings().SetTheme(theme.DarkTheme())
	w.Resize(fyne.NewSize(640, 460))
	var log = make(chan string, 10)
	var goodsId int
	name := widget.NewEntry()
	name.Disable()
	name.SetText(userName)
	goods := widget.NewEntry()
	goods.Disable()
	//goods.SetText("fasdfasdfasdfasdfsdfasfasdfasdfasdfasdf")

	//button := widget.NewButton("退出", func() {
	//	w.Close()
	//})

	jdUrl := widget.NewEntry()
	jdUrl.SetPlaceHolder("https://item.jd.com/100012043978.html")
	jdUrl.Validator = validation.NewRegexp(`^(http://|https://)?item.jd.com/([0-9]{6,})\.html$`, "不是一个有效的京东商品链接")
	jdUrl.SetText("https://item.jd.com/100012043978.html")
	//jdUrl.Wrapping = fyne.TextWrapWord

	num := widget.NewEntry()
	num.SetPlaceHolder("1")
	num.Validator = validation.NewRegexp(`^[1-9][0-9]*$`, "不是一个有效数量")
	num.SetText("2")

	form := &widget.Form{
		Items: []*widget.FormItem{
			//{Text: "京东模式",Widget: widget.NewTextGridFromString("-----------------------------------")},
			{Text: "商品链接:", Widget: container.NewVScroll(jdUrl)},
			{Text: "抢购数量:", Widget: num},
		},
		OnSubmit: func() {
			var err2 error
			goodsId, err2 = tools.Reservation(jdUrl.Text)
			if err2 != nil {
				log <- err2.Error()
				return
			}
			goods.SetText(tools.GoodsInfo[goodsId].Name)
			if tools.Eid == "" || tools.Fp == "" {
				err2 := getEipAndFp(w, tools)
				if err2 != nil {
					log <- err2.Error()
					return
				}
			}

			tools.GoodsInfo[goodsId].SnapUpStop = false
			tools.GoodsInfo[goodsId].SnapUpStatus = false
			tools.GoodsInfo[goodsId].SnapUpEndStatus = false

			tools.GoodsInfo[goodsId].BuyNum, _ = strconv.Atoi(num.Text)
			var concurrency = make(chan string, 2)
			go func() {
				for {
					concurrency <- "1"
					if tools.Eid == "" || tools.Fp == "" {
						_ = <-concurrency
						time.Sleep(2 * time.Second)
						log <- "等待Eid,Fp获取中..."
						continue
					} else {
						log <- fmt.Sprintf("已获取Eid: %s ,Fp: %s", tools.Eid, tools.Fp)
					}
					if tools.GoodsInfo[goodsId].SnapUpStatus == true || tools.GoodsInfo[goodsId].SnapUpEndStatus == true || tools.GoodsInfo[goodsId].SnapUpStop == true {
						close(concurrency)
						//fmt.Println("test")
						break
					}
					log <- fmt.Sprintf("等待抢购开始")
					for {
						if tools.GoodsInfo[goodsId].SnapUpStop == true {
							close(concurrency)
							return
						}
						surplusTime, err2 := tools.SnapUpStartSurplusTime(goodsId)
						if err2 != nil {
							log <- err2.Error()
							//_ = <-concurrency
							continue
						}
						if surplusTime.Seconds() <= 12 {
							log <- fmt.Sprintf("抢购开始了!")
							break
						} else {
							log <- fmt.Sprintf("距离抢购开始还有: %s", surplusTime.String())
							time.Sleep(10 * time.Second)
						}
					}
					go func() {
						err2 = tools.SnapUp(goodsId, log)
						if err2 != nil {
							log <- err2.Error()
						}
						_ = <-concurrency
					}()
				}
			}()
		},
		SubmitText: "开始",
		OnCancel: func() {
			tools.GoodsInfo[goodsId].SnapUpStop = true
		},
		CancelText: "停止",
	}
	info := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "用户名:", Widget: name},
			{Text: "正在抢购商品:", Widget: goods},
		},
		OnCancel: func() {
			w.Close()
		},
		CancelText: "退出",
	}

	//withLayout := fyne.NewContainerWithLayout(layout.NewBorderLayout(name,nil,nil,button), name, button)
	//split := container.NewHBox(name,makeCell(), button)
	//logInfo := widget.NewEntry()
	//logInfo.Disable()
	logInfo := widget.NewLabel("")
	logInfo.Wrapping = fyne.TextWrapWord
	go func() {
		n := 0
		compile := regexp.MustCompile(`^(?sU).* [0-9]{3}\.([0-9]{2}:){2}[0-9]{2} [0-9]{2}-[0-9]{2}-[0-9]{4}`)
		var newlog string
		for {
			newlog = <-log
			newlog = strings.ReplaceAll(newlog, "\n", "\\n")
			if n >= 10 {
				logInfo.SetText(fmt.Sprintf("%s  %s\n", time.Now().Format("2006-01-02 15:04:05.000"), newlog) + reverseString(compile.ReplaceAllString(reverseString(logInfo.Text), "")))
				continue
			} else {
				logInfo.SetText(fmt.Sprintf("%s  %s\n", time.Now().Format("2006-01-02 15:04:05.000"), newlog) + logInfo.Text)
			}
			n++
		}
	}()
	//split2 := container.NewHBox(form,form)
	//vBox := container.NewVBox(split2, logInfo)
	border1 := container.NewBorder(info, nil, nil, nil)
	border2 := container.NewBorder(form, nil, nil, nil)
	hSplit := container.NewHSplit(border1, border2)
	vSplit := container.NewVSplit(hSplit, logInfo)
	hSplit.Offset = 0.2
	vSplit.Offset = 0.2
	w.SetContent(vSplit)
	return
}

func errPrompt(w fyne.Window, err error, title string) {
	w.Resize(fyne.NewSize(480, 240))
	//w.SetFixedSize(false)
	compile := regexp.MustCompile(`(?s:.{30})`)
	info := compile.ReplaceAllStringFunc(err.Error(), func(s string) string {
		return s + "\n"
	})
	information := dialog.NewInformation(title, info, w)
	information.SetOnClosed(func() {
		w.Close()
	})
	information.Show()
}

func getEipAndFp(w fyne.Window, tools *jdTools.JdInfo) (err error) {
	path := urlTools.FindChromePath()
	if path == "" {
		err = tools.ManualObtainEidFp()
		if err != nil {
			return
		}
		//eip := widget.NewEntry()
		//fp := widget.NewEntry()
		//content := widget.NewForm(widget.NewFormItem("Eip", eip), widget.NewFormItem("Fp", fp))
		//
		//dialog.ShowCustomConfirm("填入浏览器页面中的eip和fp", "确认", "取消", content, func(b bool) {
		//	if b {
		//		tools.Eid = eip.Text
		//		tools.Fp = fp.Text
		//	} else {
		//		w.Close()
		//	}
		//}, w)
		getEipFpDialog("填入浏览器页面中的eip和fp", w, tools)
	} else {
		err = tools.AutoObtainEidFp()
		if err != nil {
			return
		}
	}
	return
}

func getEipFpDialog(title string, w fyne.Window, tools *jdTools.JdInfo) {
	eip := widget.NewEntry()
	fp := widget.NewEntry()
	content := widget.NewForm(widget.NewFormItem("Eip", eip), widget.NewFormItem("Fp", fp))
	dialog.ShowCustomConfirm(title, "确认", "取消", content, func(b bool) {
		if b {
			tools.Eid = eip.Text
			tools.Fp = fp.Text
			if tools.Eid == "" || tools.Fp == "" {
				getEipFpDialog("eip和fp不能为空！", w, tools)
			}
		} else {
			w.Close()
		}
	}, w)
}

func makeCell() fyne.CanvasObject {
	rect := canvas.NewRectangle(&color.NRGBA{128, 128, 128, 255})
	rect.SetMinSize(fyne.NewSize(2, 2))
	return rect
}

func reverseString(s string) string {
	runes := []rune(s)
	for from, to := 0, len(runes)-1; from < to; from, to = from+1, to-1 {
		runes[from], runes[to] = runes[to], runes[from]
	}
	return string(runes)
}
