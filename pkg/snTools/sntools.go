package snTools

import (
	"SnapUp/pkg/urlTools"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	cookiejar "github.com/orirawlings/persistent-cookiejar"
	"image"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	//"text/template/parse"

	//cookiejar "github.com/juju/persistent-cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

type SnTools interface {
	GetLoginQrcode() (qrcode image.Image, err error)                           //获取登录二维码
	CheckLogin() (state string, err error)                                     //检查是否扫描登录成功
	CheckCookies() (code int, err error)                                       //检查cookies是否过期
	Reservation(JdUrl string) (goodsId int, err error)                         //预约
	SnapUpStartSurplusTime(goodsId int) (SurplusTime time.Duration, err error) //等待抢购开始还有多长时间
	SnapUp(goodsId int, logChan chan<- string) (err error)                     //抢购
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

type SnInfo struct {
	LoginStatus bool
	GoodsInfo   map[int]*GoodsInfo
	CookiesJar  *cookiejar.Jar
	Eid         string
	Fp          string
}

var Headers = map[string]string{"User-Agent": "Mozilla/5.0(Linux; U;SNEBUY-APP;9.5.4-396;SNCLIENT; Android 9; zh; ONEPLUS A5010) AppleWebKit/533.0 (KHTML, like Gecko) Version/4.0 Mobile Safari/533.1 maa/2.2.2",
	"Accept":          "text/html,application/xhtml+xml,application/xml,application/json;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
	"Connection":      "keep-alive",
	"Cache-Control":   "max-age=0",
	"Accept-Language": "zh-CN,zh;q=0.9"}

var CookiesFile = "sn.cookies"
var LocalZone = time.FixedZone("CST", int((8 * time.Hour).Seconds()))

func Init() *SnInfo {
	info := &SnInfo{GoodsInfo: make(map[int]*GoodsInfo)}
	return info
}

func (Tools *SnInfo) GetLoginQrcode() (qrcode image.Image, err error) {
	var request *http.Request
	var response *http.Response
	var readAll []byte

	type tokenInfo struct {
		Code  string `json:"code"`
		Token string `json:"token"`
	}
	var tkinfo tokenInfo

	Tools.CookiesJar, err = urlTools.InitCookieJar(Tools.CookiesJar, CookiesFile)
	if err != nil {
		return
	}

	client := &http.Client{CheckRedirect: nil, Jar: Tools.CookiesJar}
	// 打开登录页
	LoginUrl := "https://passport.suning.com/ids/login"
	request, _ = http.NewRequest("GET", LoginUrl, nil)
	request = urlTools.AddHeader(request, Headers)
	response, err = client.Do(request)
	if err != nil {
		return
	}
	_ = response.Body.Close()

	Url := "https://mmds.suning.com/mmds/webCollectInit.json?appCode=qEmt9X4YmoV2Vye8"
	request, _ = http.NewRequest("GET", Url, nil)
	request = urlTools.AddHeader(request, Headers)
	response, err = client.Do(request)
	if err != nil {
		return
	}
	readAll, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}
	_ = response.Body.Close()

	compile := regexp.MustCompile(`(?s:\{.*\})`)
	findString := compile.FindString(string(readAll))

	err = json.Unmarshal([]byte(findString), &tkinfo)
	if err != nil {
		return
	}

	if tkinfo.Code != "200" {
		err = fmt.Errorf("获取token异常")
		return
	}

	//token := uuid.New()
	token := tkinfo.Token
	u, _ := url.Parse("https://suning.com")
	cookies := []*http.Cookie{}
	expires, _ := time.Parse("2006-01-02T15:04:05Z", "9999-12-31T23:59:59Z")
	cookies = append(cookies, &http.Cookie{Name: "token", Value: token, Path: "/", Domain: ".suning.com", Expires: expires})
	Tools.CookiesJar.SetCookies(u, cookies)
	fmt.Println(Tools.CookiesJar.AllCookies())

	QrcodeUrl := "https://passport.suning.com/ids/qrLoginUuidGenerate.htm"
	var QueryValues url.Values = map[string][]string{"image": {"true"}, "yys": {strconv.FormatInt(time.Now().UnixNano(), 10)[:13]}, "t": {strconv.FormatInt(time.Now().UnixNano(), 10)[:13]}}
	parse, _ := url.Parse(QrcodeUrl)
	parse.RawQuery = QueryValues.Encode()
	request, _ = http.NewRequest("GET", parse.String(), nil)
	request = urlTools.AddHeader(request, Headers)
	request.Header.Add("Referer", "https://passport.suning.com/ids/login")
	response, err = client.Do(request)
	if err != nil {
		return
	}
	if response.StatusCode == 200 {
		var body []byte
		body, err = ioutil.ReadAll(response.Body)
		qrcode, _, err = image.Decode(bytes.NewReader(body))
	} else {
		err = fmt.Errorf("状态码：%s,获取登录二维码失败", response.Status)
	}
	_ = response.Body.Close()
	return
}

func (Tools *SnInfo) CheckLogin() (state string, err error) {
	type QrCheckInfo struct {
		State string `json:"state"`
	}
	var request *http.Request
	var response *http.Response
	var uuid string
	var readAll []byte
	var qcinfo QrCheckInfo
	var postData url.Values

	client := &http.Client{CheckRedirect: nil, Jar: Tools.CookiesJar}
	Url := "https://passport.suning.com/ids/qrLoginStateProbe"
	//var n int = 85
	u, _ := url.Parse(Url)
	cookies := Tools.CookiesJar.Cookies(u)
	for _, v := range cookies {
		if v.Name == "ids_qr_uuid" {
			uuid = v.Value
			break
		}
	}
	postData = url.Values{
		"service":  {""},
		"terminal": {"PC"},
		"uuid":     {uuid},
	}
	for {
		request, _ = http.NewRequest("POST", Url, strings.NewReader(postData.Encode()))
		//fmt.Println(postData)
		request = urlTools.AddHeader(request, Headers)
		request.Header.Add("Referer", "https://passport.suning.com/ids/login?method=GET&loginTheme=b2c")
		request.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
		response, err = client.Do(request)
		if err != nil {
			return
		}

		readAll, err = ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Println(readAll)
			return
		}
		_ = response.Body.Close()

		err = json.Unmarshal(readAll, &qcinfo)
		if err != nil {
			return
		}
		//0：未扫描
		//1：手机扫描了二维码
		//2：手机确认授权
		//3：过期，二维码已失效
		//4：系统异常；账号异常，请使用账号密码登录
		//5：系统锁；您的账号已被锁定，请1小时后再试
		//6：人工锁；账号锁定，请联系客服
		//10、11、12：风控；账号存在风险，请使用账号密码登录
		//14：取消登录授权
		state = qcinfo.State
		switch state {
		case "0":
			time.Sleep(2 * time.Second)
			continue
		case "1":
			time.Sleep(2 * time.Second)
			continue
		case "2":
			Tools.LoginStatus = true
			err = Tools.CookiesJar.Save()
			if err != nil {
				state = "100"
			}
			return
		case "3":
			err = fmt.Errorf("二维码已失效")
			return
		case "14":
			err = fmt.Errorf("取消登录授权")
			return
		default:
			err = fmt.Errorf("登录异常，请检查帐号是否被限制")
			return
		}
	}
}

func (Tools *SnInfo) CheckCookies() (code int, err error) {
	var request *http.Request
	var response *http.Response
	Tools.CookiesJar, err = urlTools.InitCookieJar(Tools.CookiesJar, CookiesFile)
	if err != nil {
		return
	}
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}, Jar: Tools.CookiesJar}

	Url := "https://my.suning.com/msi2pc/memberInfo.do"
	request, _ = http.NewRequest("GET", Url, nil)
	request = urlTools.AddHeader(request, Headers)
	response, err = client.Do(request)
	if err != nil {
		return
	}
	_ = response.Body.Close()
	code = response.StatusCode
	fmt.Println(response.Request)
	if code == 200 {
		Tools.LoginStatus = true
	} else if code == 302 {
		err = fmt.Errorf("cookices 已过期，请重新登录")
	} else {
		err = fmt.Errorf("检查cookices时发生异常")
	}
	return
}

func (Tools *SnInfo) Reservation(JdUrl string) (goodsId int, err error) {
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

func (Tools *SnInfo) SnapUpStartSurplusTime(goodsId int) (SurplusTime time.Duration, err error) {
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

func (Tools *SnInfo) SnapUp(goodsId int, logChan chan<- string) (err error) {
	//err = Tools.SnapUpStartSurplusTime(goodId)
	var request *http.Request
	var response *http.Response
	var readAll []byte
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
	err = json.Unmarshal([]byte(allString), &qgurl)
	if err != nil {
		return fmt.Errorf("json解析错误:%s,text: '%s' ", err.Error(), allString)
	}
	if qgurl.Url == "" {
		err = fmt.Errorf("获取抢购链接失败")
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
		return
	}
	//defer response.Body.Close()
	readAll, err = ioutil.ReadAll(response.Body)
	_ = response.Body.Close()
	err = json.Unmarshal(readAll, &sinfo)
	if err != nil {
		//logChan <- err.Error()  + " " + request.URL.String()
		//continue
		return fmt.Errorf("json解析错误:%s,text: '%s' ", err.Error(), string(readAll))
	}
	if sinfo.Code != "200" {
		//logChan <- string(readAll) + " "+response.Request.URL.String()
		//continue
		return
		//return fmt.Errorf(sinfo.Msg)
	}
	fmt.Println(string(readAll))
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
			logChan <- fmt.Sprintf("pocId:%s  %s", pocId, err.Error())
			continue
		}
		//defer response.Body.Close()
		readAll, err = ioutil.ReadAll(response.Body)
		_ = response.Body.Close()
		err = json.Unmarshal(readAll, &qginfo)
		if err != nil {
			//fmt.Println(string(readAll))
			logChan <- fmt.Sprintf("pocId:%s json解析错误:%s,text: '%s' ", pocId, err.Error(), string(readAll))
			continue
		}
		if qginfo.Success == true {
			logChan <- fmt.Sprintf("pocId:%s 抢购成功，订单号:%d, 总价:%s, 电脑端付款链接:%s", pocId, qginfo.OrderId, qginfo.TotalMoney, qginfo.PcUrl)
			Tools.GoodsInfo[goodsId].SnapUpStatus = true
			return
		} else {
			logChan <- fmt.Sprintf("pocId:%s 抢购失败:%s", pocId, string(readAll))
		}
	}
	if endTime.Unix() < time.Now().Unix() {
		Tools.GoodsInfo[goodsId].SnapUpEndStatus = true
		logChan <- fmt.Sprintf("pocId:%s 抢购已经结束，未抢到", pocId)
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
