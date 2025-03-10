package jdTools

import (
	"SnapUp/pkg/logger"
	"SnapUp/pkg/urlTools"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	cookiejar "github.com/orirawlings/persistent-cookiejar"
	"github.com/toqueteos/webbrowser"
	"image"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	//"text/template/parse"

	//cookiejar "github.com/juju/persistent-cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

type JdTools interface {
	GetLoginQrcode() (qrcode image.Image, err error)                           //获取登录二维码
	CheckLogin() (code int, err error)                                         //检查是否扫描登录成功
	GetUserInfo() (err error)                                                  //获取用户信息
	CheckCookies() (code int, err error)                                       //检查cookies是否过期
	AutoObtainEidFp() (err error)                                              //自动获取eid和fp
	ManualObtainEidFp() (err error)                                            //手动获取eid和fp
	Reservation(JdUrl string) (goodsId int, err error)                         //预约
	SnapUpStartSurplusTime(goodsId int) (SurplusTime time.Duration, err error) //等待抢购开始还有多长时间
	SnapUp(goodsId int, logChan chan<- string) (err error)                     //抢购
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

type YuGouInfo struct {
	Info       string `json:"info"`
	Url        string `json:"url"`
	QiangStime string `json:"qiangStime"`
	QiangEtime string `json:"qiangEtime"`
	YueStime   string `json:"yueStime"`
	YueEtime   string `json:"yueEtime"`
	Error      string `json:"error"`
}

type GoodsInfo struct {
	ReservationStatus bool
	SnapUpStatus      bool
	Name              string
	YuGouInfo         YuGouInfo
	//Sku			int
	BuyNum          int
	SnapUpEndStatus bool
	SnapUpStop      bool
	SnapUpStart     bool
}

//var GoodsInfos = map[int]GoodsInfo{}

type JdInfo struct {
	UserInfo    UserInfo
	LoginStatus bool
	GoodsInfo   map[int]*GoodsInfo
	CookiesJar  *cookiejar.Jar
	Eid         string
	Fp          string
}

var Headers = map[string]string{"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36",
	"Accept":                    "text/html,application/xhtml+xml,application/xml,application/json;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
	"Connection":                "keep-alive",
	"Cache-Control":             "max-age=0",
	"Upgrade-Insecure-Requests": "1",
	"Accept-Language":           "zh-CN,zh;q=0.9"}

var GetEidFpHtml = `<!DOCTYPE html>
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

var CookiesFile = "jd.cookies"
var LocalZone = time.FixedZone("CST", int((8 * time.Hour).Seconds()))
var logApi = logger.Newlogger(logger.ERROR, os.Stdout, logger.Ldate|logger.Lmicroseconds|logger.Llongfile)

func Init() *JdInfo {
	info := &JdInfo{GoodsInfo: make(map[int]*GoodsInfo)}
	return info
}

func (Tools *JdInfo) GetLoginQrcode() (qrcode image.Image, err error) {
	var request *http.Request
	var response *http.Response

	//Tools.CookiesJar,_ = cookiejar.New(nil)
	Tools.CookiesJar, err = urlTools.InitCookieJar(Tools.CookiesJar, CookiesFile)
	if err != nil {
		return
	}
	//fmt.Println(Tools.CookiesJar)

	client := &http.Client{CheckRedirect: nil, Jar: Tools.CookiesJar}
	// 打开登录页
	LoginUrl := "https://passport.jd.com/new/login.aspx"
	request, _ = http.NewRequest("GET", LoginUrl, nil)
	request = urlTools.AddHeader(request, Headers)
	response, err = client.Do(request)
	if err != nil {
		return
	}
	//defer response.Body.Close()
	_ = response.Body.Close()
	//fmt.Println(response.Cookies())
	//Tools.Client.Jar.SetCookies(response.Request.URL,response.Cookies())

	QrcodeUrl := "https://qr.m.jd.com/show"
	var QueryValues url.Values = map[string][]string{"appid": {"133"}, "size": {"147"}, "t": {strconv.FormatInt(time.Now().UnixNano(), 10)[:13]}}
	parse, _ := url.Parse(QrcodeUrl)
	parse.RawQuery = QueryValues.Encode()
	//fmt.Println(parse.String())
	request, _ = http.NewRequest("GET", parse.String(), nil)
	request = urlTools.AddHeader(request, Headers)
	request.Header.Add("Referer", "https://passport.jd.com/")
	//url_tools.AddCookies(request,response.Cookies())
	//fmt.Println(request)
	response, err = client.Do(request)
	//response2,err = http.Get("https://qr.m.jd.com/show?appid=133&size=147&t=1609429653561")
	if err != nil {
		return
	}
	//defer response.Body.Close()
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
		_ = response.Body.Close()

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
		err = fmt.Errorf("状态码：%s,获取登录二维码失败", response.Status)
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

		var QueryValues url.Values = map[string][]string{"appid": {"133"}, "callback": {callback}, "token": {token}, "_": {strconv.FormatInt(time.Now().UnixNano(), 10)[:13]}}
		parse, _ := url.Parse(Url)
		parse.RawQuery = QueryValues.Encode()
		//fmt.Println(parse.String())
		request, _ = http.NewRequest("GET", parse.String(), nil)
		request = urlTools.AddHeader(request, Headers)
		request.Header.Add("Referer", "https://passport.jd.com/")
		//url_tools.AddCookies(request,QrcodeCookies)
		//fmt.Println(request)
		response, err = client.Do(request)
		if err != nil {
			return
		}
		//defer response.Body.Close()
		//Tools.Client.Jar.SetCookies(response.Request.URL,response.Cookies())

		readAll, err = ioutil.ReadAll(response.Body)
		if err != nil {
			return
		}
		_ = response.Body.Close()
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
			time.Sleep(2 * time.Second)
			continue
		} else if qcinfo.Code == 200 {
			Url := "https://passport.jd.com/uc/qrCodeTicketValidation"
			var QueryValues url.Values = map[string][]string{"t": {qcinfo.Ticket}}
			parse, _ := url.Parse(Url)
			parse.RawQuery = QueryValues.Encode()
			request, _ = http.NewRequest("GET", parse.String(), nil)
			request = urlTools.AddHeader(request, Headers)
			request.Header.Add("Referer", "https://passport.jd.com/uc/login?ltype=logout")
			response, err = client.Do(request)
			if err != nil {
				code = 300
				return
			}
			//defer response.Body.Close()
			readAll, err = ioutil.ReadAll(response.Body)
			if err != nil {
				code = 300
				return
			}
			_ = response.Body.Close()
			err = json.Unmarshal(readAll, &qctv)
			if err != nil {
				code = 300
				return
			}
			if qctv.ReturnCode == 0 {
				Tools.LoginStatus = true
				err = Tools.CookiesJar.Save()
				//ioutil.WriteFile("/tmp/test.log",[]byte(err.Error()),0644)
				if err != nil {
					code = 300
				}
				return
			} else {
				err = fmt.Errorf("二维码信息校验失败")
				if err != nil {
					code = 300
				}
				return
			}
		} else if qcinfo.Code == 203 || qcinfo.Code == 205 {
			err = fmt.Errorf("code: %d ,err_msg: '%s'", qcinfo.Code, qcinfo.Msg)
			return
		} else {
			err = fmt.Errorf("code: %d ,err_msg: '%s'", qcinfo.Code, qcinfo.Msg)
			return
		}
	}
	//return
}

func (Tools *JdInfo) GetUserInfo() (err error) {
	var request *http.Request
	var response *http.Response
	var readAll []byte

	client := &http.Client{CheckRedirect: nil, Jar: Tools.CookiesJar}

	Url := "https://passport.jd.com/user/petName/getUserInfoForMiniJd.action"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var QueryValues url.Values = map[string][]string{"callback": {"jQuery" + strconv.Itoa(r.Int())[:7]}, "_": {strconv.FormatInt(time.Now().UnixNano(), 10)[:13]}}
	parse, _ := url.Parse(Url)
	parse.RawQuery = QueryValues.Encode()
	request, _ = http.NewRequest("GET", parse.String(), nil)
	request = urlTools.AddHeader(request, Headers)
	request.Header.Add("Referer", "https://order.jd.com/center/list.action")
	response, err = client.Do(request)
	if err != nil {
		return
	}
	//defer response.Body.Close()
	readAll, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}
	_ = response.Body.Close()
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
	Tools.CookiesJar, err = urlTools.InitCookieJar(Tools.CookiesJar, CookiesFile)
	if err != nil {
		return
	}
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}, Jar: Tools.CookiesJar}

	Url := "https://order.jd.com/center/list.action"
	var QueryValues url.Values = map[string][]string{"rid": {strconv.FormatInt(time.Now().UnixNano(), 10)[:13]}}
	parse, _ := url.Parse(Url)
	parse.RawQuery = QueryValues.Encode()
	request, _ = http.NewRequest("GET", parse.String(), nil)
	request = urlTools.AddHeader(request, Headers)
	response, err = client.Do(request)
	if err != nil {
		return
	}
	//defer response.Body.Close()
	_ = response.Body.Close()
	code = response.StatusCode
	//fmt.Println(Tools.CookiesJar)
	//fmt.Println(response.Request.URL,response.Status)
	if code == 200 {
		Tools.LoginStatus = true
	} else if code == 302 {
		err = fmt.Errorf("cookices 已过期，请重新登录")
	} else {
		err = fmt.Errorf("检查cookices时发生异常")
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
	ts := httptest.NewServer(urlTools.WriteHTML(GetEidFpHtml))
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
	var filepath, url string
	filepath, err = urlTools.SaveHtml("geteidfp", GetEidFpHtml)
	if err != nil {
		return
	}
	url = strings.ReplaceAll("file://"+filepath, `\`, `/`)
	err = webbrowser.Open(url)
	//fmt.Println("file://" + filepath)
	return
}

func (Tools *JdInfo) Reservation(JdUrl string) (goodsId int, err error) {
	//var goodsId int
	var request *http.Request
	var response *http.Response
	var readAll []byte
	var Url string
	var QueryValues url.Values
	var doc *goquery.Document

	compile := regexp.MustCompile(`^(http://|https://)?item.jd.com/([0-9]{6,})\.html$`)
	urlfromat := compile.MatchString(JdUrl)
	if !urlfromat {
		err = fmt.Errorf("京东url格式有误")
		return
	} else {
		//fmt.Println(compile.FindAllStringSubmatch(JdUrl, 1)[0][2])
		goodsId, _ = strconv.Atoi(compile.FindAllStringSubmatch(JdUrl, 1)[0][2])
	}

	client := &http.Client{CheckRedirect: nil, Jar: Tools.CookiesJar}

	Url = "https://yushou.jd.com/youshouinfo.action"
	QueryValues = map[string][]string{"sku": {strconv.Itoa(goodsId)}}
	parse, _ := url.Parse(Url)
	parse.RawQuery = QueryValues.Encode()
	request, _ = http.NewRequest("GET", parse.String(), nil)
	request = urlTools.AddHeader(request, Headers)
	request.Header.Add("Referer", fmt.Sprintf("https://item.jd.com/%d.html", goodsId))
	response, err = client.Do(request)
	if err != nil {
		return
	}
	//defer response.Body.Close()
	//fmt.Println(goodsId)
	readAll, err = ioutil.ReadAll(response.Body)
	_ = response.Body.Close()
	//fmt.Println(string(readAll))
	var yinfo YuGouInfo
	//var goodsinfo GoodsInfo
	err = json.Unmarshal(readAll, &yinfo)
	if err != nil {
		return
	}
	//fmt.Println(yugouinfo)
	//Tools.GoodsInfo = map[int]GoodsInfo{}
	//append(Tools.GoodsInfo[],map[int]*GoodsInfo{goodsId: {YuGouInfo: yugouinfo}})
	Tools.GoodsInfo[goodsId] = &GoodsInfo{YuGouInfo: yinfo}
	//Tools.GoodsInfo[goodsId].YuGouInfo = yugouinfo
	if Tools.GoodsInfo[goodsId].YuGouInfo.Error != "" {
		err = fmt.Errorf("该商品非预约抢购商品,%s", Tools.GoodsInfo[goodsId].YuGouInfo.Error)
		return
	}
	//fmt.Println(Tools.GoodsInfo,Tools.GoodsInfo[goodsId])

	Url = "https:" + Tools.GoodsInfo[goodsId].YuGouInfo.Url
	request, _ = http.NewRequest("GET", Url, nil)
	request = urlTools.AddHeader(request, Headers)
	request.Header.Add("Referer", fmt.Sprintf("https://item.jd.com/%d.html", goodsId))
	response, err = client.Do(request)
	if err != nil {
		return
	}
	//defer response.Body.Close()
	//fmt.Println(goodsId)
	//readAll, err = ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	doc, err = goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		//fmt.Println(11111)
		return
	}
	var info string
	if info = doc.Find(".failed .bd-right-result").Text(); info != "" {
		err = fmt.Errorf(strings.TrimSpace(info))
		return
	} else if info = doc.Find(".success .bd-right-result").Text(); info != "" {
		var n = 3
		var name string
		for n != 0 {
			name, err = getGoodsName(goodsId)
			if err == nil {
				break
			}
			n--
		}
		if name == "" {
			name = "未获取成功，请稍后再试"
		}
		Tools.GoodsInfo[goodsId].Name = name
		Tools.GoodsInfo[goodsId].ReservationStatus = true
		return
	} else {
		err = fmt.Errorf("未知错误")
		return
	}
	//return
}

func (Tools *JdInfo) SnapUpStartSurplusTime(goodsId int) (SurplusTime time.Duration, err error) {
	var response *http.Response
	var readAll []byte
	type sTime struct {
		ServerTime int64 `json:"serverTime"`
	}
	var jdTime sTime
	//loc, _ := time.LoadLocation("Asia/Shanghai")
	t, _ := time.ParseInLocation("2006-01-02 15:04:05", Tools.GoodsInfo[goodsId].YuGouInfo.QiangStime, LocalZone)
	qt := (t.Unix() - 10) * 1000

	response, err = http.Get("https://a.jd.com//ajax/queryServerData.html")
	if err != nil {
		return
	}
	readAll, err = ioutil.ReadAll(response.Body)
	_ = response.Body.Close()
	err = json.Unmarshal(readAll, &jdTime)
	if err != nil {
		return
	}
	SurplusTime = time.Millisecond * time.Duration(qt-jdTime.ServerTime)
	return
}

func (Tools *JdInfo) SnapUp(goodsId int, logChan chan<- string) (err error) {
	//err = Tools.SnapUpStartSurplusTime(goodId)
	var request *http.Request
	var response *http.Response
	var read []byte
	var skuId = strconv.Itoa(goodsId)
	var Url string
	var postData url.Values
	type qgUrl struct {
		Url string `json:"url"`
	}
	type seckillInfo struct {
		Code        string `json:"code"`
		Msg         string `json:"msg"`
		AddressList []struct {
			AddressDetail  string `json:"addressDetail"`
			AddressName    string `json:"addressName"`
			AreaCode       string `json:"areaCode"`
			CityId         int    `json:"cityId"`
			CityName       string `json:"cityName"`
			CountyId       int    `json:"countyId"`
			CountyName     string `json:"countyName"`
			DefaultAddress bool   `json:"defaultAddress"`
			Email          string `json:"email"`
			Id             int    `json:"id"`
			Mobile         string `json:"mobile"`
			MobileKey      string `json:"mobileKey"`
			Name           string `json:"name"`
			Overseas       int    `json:"overseas"`
			Phone          string `json:"phone"`
			PostCode       string `json:"postCode"`
			ProvinceId     int    `json:"provinceId"`
			ProvinceName   string `json:"province_name"`
			TownId         int    `json:"townId"`
			TownName       string `json:"townName"`
			YuyueAddress   bool   `json:"yuyueAddress"`
		} `json:"addressList"`
		InvoiceInfo struct {
			InvoiceCode        string `json:"invoiceCode"`
			InvoiceCompany     string `json:"invoiceCompany"`
			InvoiceContentType int    `json:"invoiceContentType"`
			InvoiceEmail       string `json:"invoiceEmail"`
			InvoicePhone       string `json:"invoicePhone"`
			InvoicePhoneKey    string `json:"invoicePhoneKey"`
			InvoiceTitle       int    `json:"invoiceTitle"`
			InvoiceType        int    `json:"invoiceType"`
		} `json:"invoiceInfo"`
		BuyNum int    `json:"buyNum"`
		Token  string `json:"token"`
	}
	type qgInfo struct {
		Success      bool   `json:"success"`
		OrderId      int    `json:"orderId"`
		ErrorMessage string `json:"errorMessage"`
		ResultCode   int    `json:"resultCode"`
		TotalMoney   string `json:"totalMoney"`
		PcUrl        string `json:"pcUrl"`
	}
	var qgurl qgUrl
	var sinfo seckillInfo
	var qginfo qgInfo
	client := &http.Client{CheckRedirect: nil, Jar: Tools.CookiesJar}

	// 获取抢购链接
	Url = "https://itemko.jd.com/itemShowBtn"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var QueryValues url.Values = map[string][]string{"callback": {"jQuery" + strconv.Itoa(r.Int())[:7]}, "skuId": {skuId}, "from": {"pc"}, "_": {strconv.FormatInt(time.Now().UnixNano(), 10)[:13]}}
	parse, _ := url.Parse(Url)
	parse.RawQuery = QueryValues.Encode()
	request, _ = http.NewRequest("GET", parse.String(), nil)
	request = urlTools.AddHeader(request, Headers)
	request.Header.Add("Referer", fmt.Sprintf("https://item.jd.com/%d.html", goodsId))
	response, err = client.Do(request)
	if err != nil {
		logApi.DEBUG("获取抢购链接请求错误:%s ", err.Error())
		return
	}
	//defer response.Body.Close()
	//readAll, err = ioutil.ReadAll(response.Body)
	read, err = urlTools.ReadBody(response.Body, 1024)
	if err != nil {
		logApi.DEBUG("获取抢购链接请求时,读取response错误:%s", err.Error())
		return
	}
	_ = response.Body.Close()
	compile := regexp.MustCompile(`(?s:\{.*\})`)
	allString := compile.FindString(string(read))
	err = json.Unmarshal([]byte(allString), &qgurl)
	if err != nil {
		logApi.DEBUG("获取抢购链接时,json解析错误:%s,text: '%s' ", err.Error(), allString)
		return fmt.Errorf("获取抢购链接时,json解析错误:%s", err.Error())
	}
	if qgurl.Url == "" {
		err = fmt.Errorf("获取抢购链接失败")
		logApi.DEBUG("获取抢购链接失败")
		return
	}
	// 测试
	//qgurl.Url = "//divide.jd.com/user_routing?skuId=100012043978&sn=eb5489dcbe793c5725e661416bdd21c1&from=pc"

	// 访问抢购链接
	Url = "https:" + strings.ReplaceAll(strings.ReplaceAll(qgurl.Url, "divide", "marathon"), "user_routing", "captcha.html")
	client = &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}, Jar: Tools.CookiesJar}
	request, _ = http.NewRequest("GET", Url, nil)
	request = urlTools.AddHeader(request, Headers)
	request.Header.Add("Referer", fmt.Sprintf("https://item.jd.com/%d.html", goodsId))
	response, err = client.Do(request)
	if err != nil {
		logApi.DEBUG("访问抢购链接页面请求错误:%s ", err.Error())
		return
	}
	_ = response.Body.Close()

	//loc, _ := time.LoadLocation("Asia/Shanghai")
	endTime, _ := time.ParseInLocation("2006-01-02 15:04:05", Tools.GoodsInfo[goodsId].YuGouInfo.QiangEtime, LocalZone)

	//访问抢购订单结算页面
	Url = "https://marathon.jd.com/seckill/seckill.action"
	client = &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}, Jar: Tools.CookiesJar}
	rid := strconv.FormatInt(time.Now().Unix(), 10)
	QueryValues = map[string][]string{"skuId": {skuId}, "num": {strconv.Itoa(Tools.GoodsInfo[goodsId].BuyNum)}, "rid": {rid}}
	parse, _ = url.Parse(Url)
	parse.RawQuery = QueryValues.Encode()
	request, _ = http.NewRequest("GET", parse.String(), nil)
	request = urlTools.AddHeader(request, Headers)
	request.Header.Add("Referer", fmt.Sprintf("https://item.jd.com/%d.html", goodsId))
	response, err = client.Do(request)
	if err != nil {
		//logChan <- err.Error()  + " " + request.URL.String()
		logApi.DEBUG("订单结算页面请求错误:%s ", err.Error())
		return
	}
	//defer response.Body.Close()
	//readAll, err = ioutil.ReadAll(response.Body)
	_ = response.Body.Close()

	// 获取秒杀初始化信息
	postData = url.Values{
		"sku":             {skuId},
		"num":             {strconv.Itoa(Tools.GoodsInfo[goodsId].BuyNum)},
		"isModifyAddress": {"false"},
	}
	Url = "https://marathon.jd.com/seckillnew/orderService/pc/init.action"
	client = &http.Client{CheckRedirect: nil, Jar: Tools.CookiesJar}
	//fmt.Printf(postData.Encode())
	request, _ = http.NewRequest("POST", Url, strings.NewReader(postData.Encode()))
	//request, _ = http.NewRequest("POST", Url, strings.NewReader("isModifyAddress=false&num=0&sku=100012043978"))
	request = urlTools.AddHeader(request, Headers)
	request.Header.Add("Referer", parse.String())
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	response, err = client.Do(request)
	if err != nil {
		//logChan <- err.Error()  + " " + request.URL.String()
		//continue
		logApi.DEBUG("秒杀初始化信息请求错误:%s ", err.Error())
		return
	}
	//defer response.Body.Close()
	//readAll, err = ioutil.ReadAll(response.Body)
	read, err = urlTools.ReadBody(response.Body, 4096)
	_ = response.Body.Close()
	err = json.Unmarshal(read, &sinfo)
	if err != nil {
		logApi.DEBUG("获取秒杀初始化信息时，json解析错误:%s,text: '%s' ", err.Error(), string(read[:1024]))
		return fmt.Errorf("获取秒杀初始化信息时，json解析错误:%s", err.Error())
	}

	if sinfo.Code != "200" {
		logApi.DEBUG("获取秒杀初始化信息失败：%s", string(read))
		return fmt.Errorf("获取秒杀初始化信息失败")
	}
	//fmt.Println(string(readAll))
	var invoice string
	if sinfo.InvoiceInfo.InvoiceContentType == 0 {
		invoice = "false"
	} else {
		invoice = "true"
	}
	postData = url.Values{
		"addressDetail":      {sinfo.AddressList[0].AddressDetail},
		"addressId":          {strconv.Itoa(sinfo.AddressList[0].Id)},
		"areaCode":           {sinfo.AddressList[0].AreaCode},
		"cityId":             {strconv.Itoa(sinfo.AddressList[0].CityId)},
		"cityName":           {sinfo.AddressList[0].CityName},
		"codTimeType":        {"3"},
		"countyId":           {strconv.Itoa(sinfo.AddressList[0].CountyId)},
		"countyName":         {sinfo.AddressList[0].CountyName},
		"eid":                {Tools.Eid},
		"email":              {sinfo.AddressList[0].Email},
		"fp":                 {Tools.Fp},
		"invoice":            {invoice},
		"invoiceCompanyName": {sinfo.InvoiceInfo.InvoiceCompany},
		"invoiceContent":     {strconv.Itoa(sinfo.InvoiceInfo.InvoiceContentType)},
		"invoiceEmail":       {sinfo.InvoiceInfo.InvoiceEmail},
		"invoicePhone":       {sinfo.InvoiceInfo.InvoicePhone},
		"invoicePhoneKey":    {sinfo.InvoiceInfo.InvoicePhoneKey},
		"invoiceTaxpayerNO":  {""},
		"invoiceTitle":       {strconv.Itoa(sinfo.InvoiceInfo.InvoiceTitle)},
		"isModifyAddress":    {"false"},
		"mobile":             {sinfo.AddressList[0].Mobile},
		"mobileKey":          {sinfo.AddressList[0].MobileKey},
		"name":               {sinfo.AddressList[0].Name},
		"num":                {strconv.Itoa(sinfo.BuyNum)},
		"overseas":           {"0"},
		"password":           {""},
		"paymentType":        {"4"},
		"phone":              {sinfo.AddressList[0].Phone},
		"postCode":           {sinfo.AddressList[0].PostCode},
		"provinceId":         {strconv.Itoa(sinfo.AddressList[0].ProvinceId)},
		"provinceName":       {sinfo.AddressList[0].ProvinceName},
		"pru":                {""},
		"skuId":              {strconv.Itoa(goodsId)},
		"token":              {sinfo.Token},
		"townId":             {strconv.Itoa(sinfo.AddressList[0].TownId)},
		"townName":           {sinfo.AddressList[0].TownName},
		"yuShou":             {"true"},
	}
	// 抢购
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
	var pocId = strconv.Itoa(r.Int())[:5]
	for Tools.GoodsInfo[goodsId].SnapUpStatus == false && endTime.Unix() >= time.Now().Unix() && Tools.GoodsInfo[goodsId].SnapUpStop == false {
		//提交抢购
		// {'errorMessage': '很遗憾没有抢到，再接再厉哦。', 'orderId': 0, 'resultCode': 60074, 'skuId': 0, 'success': False}
		// {'errorMessage': '抱歉，您提交过快，请稍后再提交订单！', 'orderId': 0, 'resultCode': 60017, 'skuId': 0, 'success': False}
		// {'errorMessage': '系统正在开小差，请重试~~', 'orderId': 0, 'resultCode': 90013, 'skuId': 0, 'success': False}
		// {"errorMessage":"很遗憾没有抢到，再接再厉哦。","orderId":0,"resultCode":90008,"skuId":0,"success":false}
		// 抢购成功：
		// {"appUrl":"xxxxx","orderId":820227xxxxx,"pcUrl":"xxxxx","resultCode":0,"skuId":0,"success":true,"totalMoney":"xxxxx"}
		Url = fmt.Sprintf("https://marathon.jd.com/seckillnew/orderService/pc/submitOrder.action?skuId=%d", goodsId)
		client = &http.Client{CheckRedirect: nil, Jar: Tools.CookiesJar}
		request, _ = http.NewRequest("POST", Url, strings.NewReader(postData.Encode()))
		//fmt.Println(postData)
		request = urlTools.AddHeader(request, Headers)
		request.Header.Add("Referer", parse.String())
		request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		response, err = client.Do(request)
		if err != nil {
			logtext := fmt.Sprintf("pocId:%s  提交订单出错: %s", pocId, err.Error())
			logChan <- logtext
			logApi.DEBUG(logtext)
			continue
		}
		//defer response.Body.Close()
		//readAll, err = ioutil.ReadAll(response.Body)
		read, err = urlTools.ReadBody(response.Body, 1024)
		_ = response.Body.Close()
		err = json.Unmarshal(read, &qginfo)
		if err != nil {
			//fmt.Println(string(readAll))
			logtext := fmt.Sprintf("pocId:%s json解析错误:%s,text: '%s' ", pocId, err.Error(), string(read))
			logChan <- fmt.Sprintf("pocId:%s 提交订单出错,json解析错误:%s", pocId, err.Error())
			logApi.DEBUG(logtext)
			continue
		}
		if qginfo.Success == true {
			logtext := fmt.Sprintf("pocId:%s 抢购成功，订单号:%d, 总价:%s, 电脑端付款链接:%s", pocId, qginfo.OrderId, qginfo.TotalMoney, qginfo.PcUrl)
			logChan <- logtext
			logApi.DEBUG(logtext)
			Tools.GoodsInfo[goodsId].SnapUpStatus = true
			return
		} else if qginfo.Success == false {
			logtext := fmt.Sprintf("pocId:%s 抢购失败:%s", pocId, string(read))
			logChan <- logtext
			logApi.DEBUG(logtext)
		} else {
			logtext := fmt.Sprintf("pocId:%s 抢购失败，未知错误: %s", pocId, string(read))
			logChan <- logtext
			logApi.DEBUG(logtext)
		}
	}
	if endTime.Unix() < time.Now().Unix() {
		Tools.GoodsInfo[goodsId].SnapUpEndStatus = true
		logtext := fmt.Sprintf("pocId:%s 抢购已经结束，未抢到", pocId)
		logChan <- logtext
		logApi.DEBUG(logtext)
	}
	return
}

func getGoodsName(goodsId int) (name string, err error) {
	var resp *http.Response
	var doc *goquery.Document
	client := &http.Client{}

	Url := fmt.Sprintf("https://item.jd.com/%d.html", goodsId)
	request, _ := http.NewRequest("GET", Url, nil)
	request = urlTools.AddHeader(request, Headers)
	resp, err = client.Do(request)
	if err != nil {
		return
	}

	doc, err = goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return
	}
	_ = resp.Body.Close()

	name = doc.Find(".p-info .p-name").Text()
	//fmt.Println(doc.Text(),resp.Status,resp.Request.URL.String())
	return
}
