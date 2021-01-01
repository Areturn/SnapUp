package jd_tools

import (
	"SnapUp/pkg/url_tools"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

type JdTools interface {
	GetLoginQrcode() (qrcode image.Image, err error) //获取登录二维码
	CheckLogin() (err error)                         //检查是否扫描登录成功
	CheckCookies() (status bool)                     //检查cookies是否过期
}

type JdInfo struct {
	UserNmae    string
	LoginStatus bool
	CookiesJar  *cookiejar.Jar
}

func (Tools *JdInfo) GetLoginQrcode() (qrcode image.Image, err error) {
	var request *http.Request
	var response *http.Response

	//Tools.CookiesJar,_ = cookiejar.New(nil)
	Tools.CookiesJar = url_tools.InitCookieJar(Tools.CookiesJar)
	//fmt.Println(Tools.CookiesJar)

	client := &http.Client{CheckRedirect: nil, Jar: Tools.CookiesJar}
	// 打开登录页
	LoginUrl := "https://passport.jd.com/new/login.aspx"
	request, _ = http.NewRequest("GET", LoginUrl, nil)
	request = url_tools.AddHeader(request)
	response, err = client.Do(request)
	defer response.Body.Close()
	if err != nil {
		return
	}
	//fmt.Println(response.Cookies())
	//Tools.Client.Jar.SetCookies(response.Request.URL,response.Cookies())

	QrcodeUrl := "https://qr.m.jd.com/show"
	var QueryValues url.Values = map[string][]string{"appid": []string{"133"}, "size": []string{"147"}, "t": []string{strconv.FormatInt(time.Now().UnixNano(), 10)[:13]}}
	parse, _ := url.Parse(QrcodeUrl)
	parse.RawQuery = QueryValues.Encode()
	//fmt.Println(parse.String())
	request, _ = http.NewRequest("GET", parse.String(), nil)
	request = url_tools.AddHeader(request)
	request.Header.Add("Referer", "https://passport.jd.com/")
	//url_tools.AddCookies(request,response.Cookies())
	//fmt.Println(request)
	response, err = client.Do(request)
	//response2,err = http.Get("https://qr.m.jd.com/show?appid=133&size=147&t=1609429653561")
	defer response.Body.Close()
	if err != nil {
		return
	}
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

func (Tools *JdInfo) CheckLogin() (err error) {
	type QrCheckInfo struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	var request *http.Request
	var response *http.Response
	var token string
	var readAll []byte
	var qcinfo QrCheckInfo

	client := &http.Client{CheckRedirect: nil, Jar: Tools.CookiesJar}
	QrcodeUrl := "https://qr.m.jd.com/check"
	//var n int = 85
	for {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		u, _ := url.Parse(QrcodeUrl)
		cookies := Tools.CookiesJar.Cookies(u)
		for _, v := range cookies {
			if v.Name == "wlfstk_smdl" {
				token = v.Value
				break
			}
		}
		var QueryValues url.Values = map[string][]string{"appid": []string{"133"}, "callback": []string{"jQuery" + strconv.Itoa(r.Int())[:9]}, "token": []string{token}, "_": []string{strconv.FormatInt(time.Now().UnixNano(), 10)[:13]}}
		parse, _ := url.Parse(QrcodeUrl)
		parse.RawQuery = QueryValues.Encode()
		//fmt.Println(parse.String())
		request, _ = http.NewRequest("GET", parse.String(), nil)
		request = url_tools.AddHeader(request)
		request.Header.Add("Referer", "https://passport.jd.com/")
		//url_tools.AddCookies(request,QrcodeCookies)
		//fmt.Println(request)
		response, err = client.Do(request)
		defer response.Body.Close()
		if err != nil {
			return
		}
		//Tools.Client.Jar.SetCookies(response.Request.URL,response.Cookies())

		readAll, err = ioutil.ReadAll(response.Body)
		if err != nil {
			return
		}
		//fmt.Println(string(readAll))
		compile := regexp.MustCompile(`(?s:\{.*\})`)
		findString := compile.FindString(string(readAll))

		err = json.Unmarshal([]byte(findString), &qcinfo)
		if err != nil {
			return
		}
		// 201:二维码未扫描，请扫描二维码	202:请手机客户端确认登录	203:二维码过期，请重新扫描	205:二维码已取消授权
		if qcinfo.Code == 201 || qcinfo.Code == 202 {
			//err = fmt.Errorf("err_msg: '%s'",qcinfo.Msg)
			continue
		} else if qcinfo.Code == 200 {
			//fmt.Println(Tools.CookiesJar)
			return
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

func (Tools *JdInfo) CheckCookies() (status bool) {

	return
}
