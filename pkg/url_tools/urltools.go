package url_tools

import (
	"fmt"
	cookiejar "github.com/orirawlings/persistent-cookiejar"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	//cookiejar "github.com/juju/persistent-cookiejar"
	"os"
)

var CacheDir = "cache"

func AddHeader(request *http.Request, headers map[string]string) (NewRequest *http.Request) {
	//request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36")
	//request.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	//request.Header.Add("Connection", "keep-alive")
	//request.Header.Add("Cache-Control", "max-age=0")
	//request.Header.Add("Upgrade-Insecure-Requests", "1")
	//request.Header.Add("Accept-Language", "zh-CN,zh;q=0.9")
	for k, v := range headers {
		request.Header.Add(k, v)
	}
	//request.Header.Add("Accept-Encoding","gzip, deflate, br")
	return request
}

func InitCookieJar(Cookiejar *cookiejar.Jar, File string) (NewCookiejar *cookiejar.Jar, err error) {
	if Cookiejar == nil {
		//NewCookiejar, _ = cookiejar.New(nil)
		//var stat os.FileInfo
		//if stat, err = os.Stat(CacheDir); err != nil && os.IsExist(err) {
		//	return
		//} else if os.IsNotExist(err) {
		//	err = os.MkdirAll(CacheDir, os.ModePerm)
		//	if err != nil {
		//		return
		//	}
		//} else if ! stat.IsDir() {
		//	err = fmt.Errorf("目录'%s'已存在，且是个文件，请检查。",CacheDir)
		//	return
		//}
		err = CreateDir(CacheDir)
		if err != nil {
			return
		}
		var path string
		path, err = AsbPath()
		if err != nil {
			return
		}
		// 调试
		path = "."

		NewCookiejar, _ = cookiejar.New(&cookiejar.Options{Filename: path + "/" + CacheDir + "/" + File, PersistSessionCookies: true})
		//NewCookiejar, _ = cookiejar.New(nil)
		return
	}
	NewCookiejar = Cookiejar
	return
}

func WriteHTML(content string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, strings.TrimSpace(content))
	})
}

func SaveHtml(filename string, content string) (filepath string, err error) {
	//var file *os.File
	err = CreateDir(CacheDir)
	if err != nil {
		return
	}
	var path string
	path, err = AsbPath()
	if err != nil {
		return
	}
	// 调试
	path = "."
	//file, err = os.OpenFile(path+"/"+CacheDir+"/"+filename+".html", os.O_CREATE|os.O_RDWR, 0640)
	//if err != nil {
	//	return
	//}
	//defer file.Close()
	filepath = path + "/" + CacheDir + "/" + filename + ".html"
	err = ioutil.WriteFile(filepath, []byte(content), 0640)
	//if err != nil {
	//	return
	//}
	return
}

func CreateDir(dir string) (err error) {
	var stat os.FileInfo
	if stat, err = os.Stat(CacheDir); err != nil && os.IsExist(err) {
		return
	} else if os.IsNotExist(err) {
		err = os.MkdirAll(CacheDir, os.ModePerm)
		if err != nil {
			return
		}
	} else if !stat.IsDir() {
		err = fmt.Errorf("目录'%s'已存在,且是个文件,请检查.", CacheDir)
		return
	}
	return
}

func AsbPath() (path string, err error) {
	// 获取可执行文件相对于当前工作目录的相对路径
	path = filepath.Dir(os.Args[0])
	// 根据相对路径获取可执行文件的绝对路径
	path, err = filepath.Abs(path)
	return
}

func FindChromePath() string {
	for _, path := range [...]string{
		// Unix-like
		"headless_shell",
		"headless-shell",
		"chromium",
		"chromium-browser",
		"google-chrome",
		"google-chrome-stable",
		"google-chrome-beta",
		"google-chrome-unstable",
		"/usr/bin/google-chrome",

		// Windows
		"chrome",
		"chrome.exe", // in case PATHEXT is misconfigured
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		filepath.Join(os.Getenv("USERPROFILE"), `AppData\Local\Google\Chrome\Application\chrome.exe`),

		// Mac
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
	} {
		found, err := exec.LookPath(path)
		if err == nil {
			return found
		}
	}
	// Fall back to something simple and sensible, to give a useful error
	// message.
	return ""
}

//func SaveCookie(Prefix string,UserName string,Cookiejar *cookiejar.Jar) (err error) {
//	//fmt.Println(response.Cookies())
//	//f, _ := os.OpenFile("test.cookies", os.O_RDWR|os.O_CREATE, 0600)
//	//defer f.Close()
//	//enc := gob.NewEncoder(f)
//	//enc.Encode(response.Cookies())
//	//os.OpenFile("")
//	//Cookiejar.Save()
//	var stat os.FileInfo
//	var f *os.File
//	if stat, err = os.Stat(CacheDir); err != nil && os.IsExist(err) {
//		err = fmt.Errorf("保存cookie时出错,ERROR: %s",err)
//		return
//	} else if os.IsNotExist(err) {
//		err = os.MkdirAll(CacheDir, os.ModePerm)
//		if err != nil {
//			return
//		}
//	} else if ! stat.IsDir() {
//		err = fmt.Errorf("'%s'文件已存在，且不是个目录。",CacheDir)
//		return
//	}
//	f, err = os.OpenFile(fmt.Sprintf("%s-%s.cookie",Prefix,UserName), os.O_RDWR|os.O_CREATE, 0600)
//	defer f.Close()
//	if err != nil {
//		return
//	}
//	enc := gob.NewEncoder(f)
//	fmt.Println(Cookiejar)
//	err = enc.Encode(Cookiejar)
//	return
//}

//func AddCookies(request *http.Request, cookies []*http.Cookie) (NewRequest *http.Request) {
//	for _, v := range cookies {
//		request.AddCookie(v)
//	}
//	return
//}
