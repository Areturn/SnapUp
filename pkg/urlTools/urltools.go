package urlTools

import (
	"fmt"
	cookiejar "github.com/orirawlings/persistent-cookiejar"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	//cookiejar "github.com/juju/persistent-cookiejar"
	"os"
)

var CacheDir = "cache"

func AddHeader(request *http.Request, headers map[string]string) (NewRequest *http.Request) {
	for k, v := range headers {
		request.Header.Add(k, v)
	}
	//request.Header.Add("Accept-Encoding","gzip, deflate, br")
	return request
}

func InitCookieJar(Cookiejar *cookiejar.Jar, File string) (NewCookiejar *cookiejar.Jar, err error) {
	var path string
	if Cookiejar == nil {
		path, err = CreateDir(CacheDir)
		if err != nil {
			return
		}
		//path, err = AsbPath()
		//if err != nil {
		//	return
		//}
		// ide调试时开启
		//path = "./cache"

		NewCookiejar, _ = cookiejar.New(&cookiejar.Options{Filename: path + "/" + File, PersistSessionCookies: true})
		//fmt.Println(path + "/" + File)
		//fmt.Println(path + "/" + CacheDir + "/" + File)
		//ioutil.WriteFile("/tmp/test1.log",[]byte(path + "/" + CacheDir + "/" + File),0644)
		//NewCookiejar, _ = cookiejar.New(nil)
		return
	}
	NewCookiejar = Cookiejar
	return
}

func WriteHTML(content string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, strings.TrimSpace(content))
	})
}

func SaveHtml(filename string, content string) (filepath string, err error) {
	//var file *os.File
	var path string
	path, err = CreateDir(CacheDir)
	if err != nil {
		return
	}
	//var path string
	//path, err = AsbPath()
	//if err != nil {
	//	return
	//}
	// ide调试时开启
	//path = "./cache"

	//file, err = os.OpenFile(path+"/"+CacheDir+"/"+filename+".html", os.O_CREATE|os.O_RDWR, 0640)
	//if err != nil {
	//	return
	//}
	//defer file.Close()
	filepath = path + "/" + filename + ".html"
	err = ioutil.WriteFile(filepath, []byte(content), 0640)
	//if err != nil {
	//	return
	//}
	return
}

func CreateDir(dir string) (dirPath string, err error) {
	var stat os.FileInfo
	var path string
	if !regexp.MustCompile(`^/`).MatchString(dir) {
		path, err = AsbPath()
		if err != nil {
			return
		}
		dir = path + "/" + dir
	}
	if stat, err = os.Stat(dir); err != nil && os.IsExist(err) {
		return
	} else if os.IsNotExist(err) {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return
		}
	} else if !stat.IsDir() {
		err = fmt.Errorf("目录'%s'已存在,且是个文件,请检查.", dir)
		return
	}
	return dir, nil
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

func ReadBody(body io.Reader, byteCount int) (read []byte, err error) {
	if byteCount == 0 {
		read, err = ioutil.ReadAll(body)
		if err != nil {
			return
		}
	} else if byteCount < 0 {
		err = fmt.Errorf("'byteCount' 不能是负数")
		return
	} else {
		var n int
		buf := make([]byte, byteCount)
		n, err = body.Read(buf)
		if err != nil && err != io.EOF {
			return
		}
		if err == io.EOF {
			err = nil
		}
		read = buf[:n]
	}
	return
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
