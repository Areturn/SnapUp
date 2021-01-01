package main

import "SnapUp/pkg/jd_tools"

func main() {
	//var QueryValues url.Values = map[string][]string{"1":[]string{"fasdf"},"2":[]string{"XXXX"}}
	//parse, _ := url.Parse("https://qr.m.jd.com/show")
	//parse.RawQuery = QueryValues.Encode()
	//fmt.Println(parse.String())
	//formatInt := strconv.FormatInt(time.Now().UnixNano(), 10)
	//fmt.Println(formatInt[:13])
	//fmt.Println(time.Now().Unix())

	var tools jd_tools.JdInfo
	tools.GetLoginQrcode()

	//var ck = []*http.Cookie{}
	//f, _ := os.Open("../test.cookies")
	//encoder := gob.NewDecoder(f)
	//_ = encoder.Decode(&ck)
	//fmt.Println(ck)
	//r := rand.New(rand.NewSource(time.Now().UnixNano()))
	//
	//fmt.Println(strconv.Itoa(r.Int())[:9])

	//type j struct {
	//	Code int	`json:"code"`
	//	Msg string	`json:"msg"`
	//}
	//var k j
	//s := `jQuery1230253({
	//	"code" : 201,
	//		"msg" : "二维码未扫描，请扫描二维码"
	//})`
	//compile := regexp.MustCompile(`(?s:\{.*\})`)
	//allString := compile.FindString(s)
	//fmt.Println(allString)
	//json.Unmarshal([]byte(allString),&k)
	//fmt.Println(k)

}
