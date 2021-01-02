package jd_tools

import (
	"SnapUp/pkg/url_tools"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/chromedp/chromedp"
	cookiejar "github.com/orirawlings/persistent-cookiejar"
	"github.com/toqueteos/webbrowser"
	"image"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	//"text/template/parse"

	//cookiejar "github.com/juju/persistent-cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

type JdTools interface {
	GetLoginQrcode() (qrcode image.Image, err error) //获取登录二维码
	CheckLogin() (code int, err error)               //检查是否扫描登录成功
	GetUserInfo() (err error)                        //获取用户信息
	CheckCookies() (code int, err error)             //检查cookies是否过期
	AutoObtainEidFp() (err error)                    //自动获取eid和fp
	ManualObtainEidFp() (err error)                  //手动获取eid和fp
	Reservation() (err error)                        //预约
	//SnapUp() ()											//抢购
}

type UserInfo struct {
	HouseholdAppliance int    `json:"householdAppliance"`
	ImgUrl             string `json:"imgUrl"`
	LastLoginTime      string `json:"lastLoginTime"`
	NickName           string `json:"nickName"`
	PlusStatus         string `json:"plusStatus"`
	RealName           string `json:"realName"`
	UserLevel          int    `json:"userLevel"`
	UserScoreVO        struct {
		AccountScore     int    `json:"accountScore"`
		ActivityScore    int    `json:"activityScore"`
		ConsumptionScore int    `json:"consumptionScore"`
		Default          bool   `json:"default"`
		FinanceScore     int    `json:"financeScore"`
		Pin              string `json:"pin"`
		RiskScore        int    `json:"riskScore"`
		TotalScore       int    `json:"total_score"`
	} `json:"userScoreVO"`
}

type JdInfo struct {
	UserInfo          UserInfo
	LoginStatus       bool
	ReservationStatus bool
	GoodsInfo         string
	CookiesJar        *cookiejar.Jar
	Eid               string
	Fp                string
}

var Headers map[string]string = map[string]string{"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36",
	"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
	"Connection":                "keep-alive",
	"Cache-Control":             "max-age=0",
	"Upgrade-Insecure-Requests": "1",
	"Accept-Language":           "zh-CN,zh;q=0.9"}

var GetEidFpHtml string = `<!DOCTYPE html>
<html lang="en">
<head></head>
<body>
<div >eid:</div>
<div id="eid"></div>
<div >fp:</div>
<div id="fp"></div>
<div id="end"></div>
</body>
</html>

<script src="https://gias.jd.com/js/td.js"></script>

<script>
    setTimeout(function () {
        try {
            getJdEid(function (eid, fp, udfp) {
                document.getElementById('eid').innerText = eid;
                document.getElementById('fp').innerText = fp;
                document.getElementById('end').innerHTML = '<form id="endok"></form>';
            });
        } catch (e) {
            document.getElementById('info').innerText = e;
        }
    }, 1000);
</script>`

var CookiesFile string = "jd.cookies"

func (Tools *JdInfo) GetLoginQrcode() (qrcode image.Image, err error) {
	var request *http.Request
	var response *http.Response

	//Tools.CookiesJar,_ = cookiejar.New(nil)
	Tools.CookiesJar, err = url_tools.InitCookieJar(Tools.CookiesJar, CookiesFile)
	if err != nil {
		return
	}
	//fmt.Println(Tools.CookiesJar)

	client := &http.Client{CheckRedirect: nil, Jar: Tools.CookiesJar}
	// 打开登录页
	LoginUrl := "https://passport.jd.com/new/login.aspx"
	request, _ = http.NewRequest("GET", LoginUrl, nil)
	request = url_tools.AddHeader(request, Headers)
	response, err = client.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	//fmt.Println(response.Cookies())
	//Tools.Client.Jar.SetCookies(response.Request.URL,response.Cookies())

	QrcodeUrl := "https://qr.m.jd.com/show"
	var QueryValues url.Values = map[string][]string{"appid": []string{"133"}, "size": []string{"147"}, "t": []string{strconv.FormatInt(time.Now().UnixNano(), 10)[:13]}}
	parse, _ := url.Parse(QrcodeUrl)
	parse.RawQuery = QueryValues.Encode()
	//fmt.Println(parse.String())
	request, _ = http.NewRequest("GET", parse.String(), nil)
	request = url_tools.AddHeader(request, Headers)
	request.Header.Add("Referer", "https://passport.jd.com/")
	//url_tools.AddCookies(request,response.Cookies())
	//fmt.Println(request)
	response, err = client.Do(request)
	//response2,err = http.Get("https://qr.m.jd.com/show?appid=133&size=147&t=1609429653561")
	if err != nil {
		return
	}
	defer response.Body.Close()
	//Tools.Client.Jar.SetCookies(response.Request.URL,response.Cookies())
	if response.StatusCode == 200 {
		var n int
		buf := make([]byte, 1024)
		var body []byte
		//var f *os.File
		//fmt.Println(response.Request)
		for {
			n, err = response.Body.Read(buf)
			if err != nil && err != io.EOF {
				return
			}
			if n == 0 {
				break
			}
			body = append(body, buf...)
		}
		//f, err = os.OpenFile("test.png", os.O_RDWR|os.O_CREATE, 0644)
		//if err != nil {
		//	return
		//}
		//_, err = f.Write(body)

		//fmt.Println(response.Cookies())
		//f, _ := os.OpenFile("test.cookies", os.O_RDWR|os.O_CREATE, 0600)
		//defer f.Close()
		//enc := gob.NewEncoder(f)
		//enc.Encode(response.Cookies())
		//i := client.Jar.Cookies(response.Request.URL)

		qrcode, _, err = image.Decode(bytes.NewReader(body))
		//cookies = response.Cookies()
		//myApp := app.New()
		//w := myApp.NewWindow("Image")
		//image1 := canvas.NewImageFromImage(qrcode)
		//image1.FillMode = canvas.ImageFillOriginal
		//w.SetContent(image1)
		//
		//w.ShowAndRun()

	} else {
		err = fmt.Errorf("状态码：%s,获取登录二维码失败.", response.Status)
	}

	return
}

func (Tools *JdInfo) CheckLogin() (code int, err error) {
	type QrCheckInfo struct {
		Code   int    `json:"code"`
		Msg    string `json:"msg"`
		Ticket string `json:"ticket"`
	}
	type qrCodeTicketValidation struct {
		ReturnCode int `json:"returnCode"`
	}
	var request *http.Request
	var response *http.Response
	var token string
	var readAll []byte
	var qcinfo QrCheckInfo
	var qctv qrCodeTicketValidation

	client := &http.Client{CheckRedirect: nil, Jar: Tools.CookiesJar}
	Url := "https://qr.m.jd.com/check"
	//var n int = 85
	for {
		u, _ := url.Parse(Url)
		cookies := Tools.CookiesJar.Cookies(u)
		for _, v := range cookies {
			if v.Name == "wlfstk_smdl" {
				token = v.Value
				break
			}
		}
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		var callback = "jQuery" + strconv.Itoa(r.Int())[:9]

		var QueryValues url.Values = map[string][]string{"appid": []string{"133"}, "callback": []string{callback}, "token": []string{token}, "_": []string{strconv.FormatInt(time.Now().UnixNano(), 10)[:13]}}
		parse, _ := url.Parse(Url)
		parse.RawQuery = QueryValues.Encode()
		//fmt.Println(parse.String())
		request, _ = http.NewRequest("GET", parse.String(), nil)
		request = url_tools.AddHeader(request, Headers)
		request.Header.Add("Referer", "https://passport.jd.com/")
		//url_tools.AddCookies(request,QrcodeCookies)
		//fmt.Println(request)
		response, err = client.Do(request)
		if err != nil {
			return
		}
		defer response.Body.Close()
		//Tools.Client.Jar.SetCookies(response.Request.URL,response.Cookies())

		readAll, err = ioutil.ReadAll(response.Body)
		if err != nil {
			return
		}
		//fmt.Println(string(readAll))
		//fmt.Println(Tools.CookiesJar)
		compile := regexp.MustCompile(`(?s:\{.*\})`)
		findString := compile.FindString(string(readAll))

		err = json.Unmarshal([]byte(findString), &qcinfo)
		if err != nil {
			return
		}
		code = qcinfo.Code
		// 201:二维码未扫描，请扫描二维码	202:请手机客户端确认登录	203:二维码过期，请重新扫描	205:二维码已取消授权
		if qcinfo.Code == 201 || qcinfo.Code == 202 {
			//err = fmt.Errorf("err_msg: '%s'",qcinfo.Msg)
			continue
		} else if qcinfo.Code == 200 {
			//fmt.Println(Tools.CookiesJar)
			Url := "https://passport.jd.com/uc/qrCodeTicketValidation"
			var QueryValues url.Values = map[string][]string{"t": []string{qcinfo.Ticket}}
			parse, _ := url.Parse(Url)
			parse.RawQuery = QueryValues.Encode()
			//fmt.Println(parse.String())
			request, _ = http.NewRequest("GET", parse.String(), nil)
			request = url_tools.AddHeader(request, Headers)
			request.Header.Add("Referer", "https://passport.jd.com/uc/login?ltype=logout")
			//url_tools.AddCookies(request,response.Cookies())
			//fmt.Println(request)
			response, err = client.Do(request)
			//response2,err = http.Get("https://qr.m.jd.com/show?appid=133&size=147&t=1609429653561")
			if err != nil {
				return
			}
			defer response.Body.Close()
			readAll, err = ioutil.ReadAll(response.Body)
			if err != nil {
				return
			}
			err = json.Unmarshal([]byte(readAll), &qctv)
			if err != nil {
				return
			}
			if qctv.ReturnCode == 0 {
				// 获取用户信息后保存cookice
				//err = Tools.GetUserInfo()
				//if err != nil {
				//	return
				//}
				Tools.LoginStatus = true
				err = Tools.CookiesJar.Save()
				//fmt.Println(Tools.CookiesJar)
				//fmt.Println("---------------------------")
				//err = url_tools.SaveCookie("Jd", Tools.UserInfo.NickName, Tools.CookiesJar)
				return
			} else {
				err = fmt.Errorf("二维码信息校验失败.")
				return
			}
		} else if qcinfo.Code == 203 || qcinfo.Code == 205 {
			err = fmt.Errorf("code: %d ,err_msg: '%s'", qcinfo.Code, qcinfo.Msg)
			return
		} else {
			err = fmt.Errorf("code: %d ,err_msg: '%s'", qcinfo.Code, qcinfo.Msg)
			return
		}
		time.Sleep(2 * time.Second)
	}
	return
}

func (Tools *JdInfo) GetUserInfo() (err error) {
	var request *http.Request
	var response *http.Response
	var readAll []byte

	client := &http.Client{CheckRedirect: nil, Jar: Tools.CookiesJar}

	Url := "https://passport.jd.com/user/petName/getUserInfoForMiniJd.action"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var QueryValues url.Values = map[string][]string{"callback": []string{"jQuery" + strconv.Itoa(r.Int())[:7]}, "_": []string{strconv.FormatInt(time.Now().UnixNano(), 10)[:13]}}
	parse, _ := url.Parse(Url)
	parse.RawQuery = QueryValues.Encode()
	request, _ = http.NewRequest("GET", parse.String(), nil)
	request = url_tools.AddHeader(request, Headers)
	request.Header.Add("Referer", "https://order.jd.com/center/list.action")
	response, err = client.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	readAll, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}

	compile := regexp.MustCompile(`(?s:\{.*\})`)
	allString := compile.FindString(string(readAll))
	err = json.Unmarshal([]byte(allString), &Tools.UserInfo)
	if err != nil {
		return
	}
	return
}

func (Tools *JdInfo) CheckCookies() (code int, err error) {
	var request *http.Request
	var response *http.Response
	Tools.CookiesJar, err = url_tools.InitCookieJar(Tools.CookiesJar, CookiesFile)
	if err != nil {
		return
	}
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}, Jar: Tools.CookiesJar}

	Url := "https://order.jd.com/center/list.action"
	var QueryValues url.Values = map[string][]string{"rid": []string{strconv.FormatInt(time.Now().UnixNano(), 10)[:13]}}
	parse, _ := url.Parse(Url)
	parse.RawQuery = QueryValues.Encode()
	request, _ = http.NewRequest("GET", parse.String(), nil)
	request = url_tools.AddHeader(request, Headers)
	response, err = client.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	code = response.StatusCode
	//fmt.Println(Tools.CookiesJar)
	//fmt.Println(response.Request.URL,response.Status)
	if code == 200 {
		Tools.LoginStatus = true
	} else if code == 302 {
		err = fmt.Errorf("cookices 已过期，请重新登录.")
	} else {
		err = fmt.Errorf("检查cookices时发生异常.")
	}
	return
}

func (Tools *JdInfo) AutoObtainEidFp() (err error) {
	var allocCtx = context.Background()
	// 关闭无头模式
	//opts := append(chromedp.DefaultExecAllocatorOptions[:],
	//	chromedp.Flag("headless", false),
	//)
	//allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	//defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ts := httptest.NewServer(url_tools.WriteHTML(GetEidFpHtml))
	defer ts.Close()
	if err = chromedp.Run(ctx,
		chromedp.Navigate(ts.URL),
		chromedp.WaitVisible(`#endok`, chromedp.ByID),
		chromedp.Text(`#eid`, &Tools.Eid),
		chromedp.Text(`#fp`, &Tools.Fp),
	); err != nil {
		return
	}
	if Tools.Eid == "" || Tools.Fp == "" {
		err = Tools.AutoObtainEidFp()
	}
	return
}

func (Tools *JdInfo) ManualObtainEidFp() (err error) {
	var filepath string
	filepath, err = url_tools.SaveHtml("geteidfp", GetEidFpHtml)
	if err != nil {
		return
	}
	err = webbrowser.Open("file://" + filepath)
	return
}

func (Tools *JdInfo) Reservation(JdUrl string) (err error) {
	var goodsId int
	var request *http.Request
	var response *http.Response
	var readAll []byte

	compile := regexp.MustCompile(`^(http://|https://)?item.jd.com/([0-9]{6,})\.html$`)
	urlfromat := compile.MatchString(JdUrl)
	if !urlfromat {
		err = fmt.Errorf("京东url格式有误.")
		return
	} else {
		goodsId, _ = strconv.Atoi(compile.FindAllStringSubmatch(JdUrl, 1)[0][2])
	}

	client := &http.Client{CheckRedirect: nil, Jar: Tools.CookiesJar}

	Url := "https://yushou.jd.com/youshouinfo.action"
	var QueryValues url.Values = map[string][]string{"suk": []string{strconv.Itoa(goodsId)}}
	parse, _ := url.Parse(Url)
	parse.RawQuery = QueryValues.Encode()
	request, _ = http.NewRequest("GET", parse.String(), nil)
	request = url_tools.AddHeader(request, Headers)
	request.Header.Add("Referer", fmt.Sprintf("https://item.jd.com/%d.html", goodsId))
	response, err = client.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	//fmt.Println(goodsId)
	readAll, err = ioutil.ReadAll(response.Body)

	err = json.Unmarshal([]byte(readAll), &Tools.UserInfo)
	if err != nil {
		return
	}

	return
}
