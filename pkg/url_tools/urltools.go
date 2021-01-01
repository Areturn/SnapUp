package url_tools

import (
	"net/http"
	"net/http/cookiejar"
)

func AddHeader(request *http.Request) (NewRequest *http.Request) {
	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36")
	request.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	request.Header.Add("Connection", "keep-alive")
	request.Header.Add("Cache-Control", "max-age=0")
	request.Header.Add("Upgrade-Insecure-Requests", "1")
	request.Header.Add("Accept-Language", "zh-CN,zh;q=0.9")
	//request.Header.Add("Accept-Encoding","gzip, deflate, br")
	return request
}

func InitCookieJar(Cookiejar *cookiejar.Jar) (NewCookiejar *cookiejar.Jar) {
	if Cookiejar == nil {
		NewCookiejar, _ = cookiejar.New(nil)
	}
	return
}

func AddCookies(request *http.Request, cookies []*http.Cookie) (NewRequest *http.Request) {
	for _, v := range cookies {
		request.AddCookie(v)
	}
	return
}
