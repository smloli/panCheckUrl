package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/mitchellh/go-ps"
)

// 判断当前终端是否支持输出颜色
var outputColor = true

type Loli struct {
	urlList []string          // 链接列表
	id      map[string]bool   // 链接ID
	okUrl   []string          // 有效链接
	errUrl  []string          // 无效链接
	Pwd     map[string]string //提取码map
}

// 阿里
type AliResponse struct {
	Code       string
	Share_name string
}

// 夸克
type QuarkResponse struct {
	Code int
}

var client = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func get(url string, headers map[string]string) (*http.Response, []byte, error) {
	req, _ := http.NewRequest("GET", url, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, data, nil
}

func post(url string, headers map[string]string, body io.Reader) ([]byte, error) {
	req, _ := http.NewRequest("POST", url, body)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return data, nil
}

func aliYunCheck(url string) (ok bool) {
	var r AliResponse
	log.SetPrefix("aliYunCheck():")
	share_id := (url)[30:]
	url = "https://api.aliyundrive.com/adrive/v3/share_link/get_share_by_anonymous?share_id=" + share_id
	param := map[string]string{
		"share_id": share_id,
	}
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Linux; Android 11; SM-G9880) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.37 Mobile Safari/537.36",
		"Referer":    "https://www.aliyundrive.com/",
	}
	data, _ := json.Marshal(param)
	resp, err := post(url, headers, bytes.NewReader(data))
	if err != nil {
		return false
	}
	json.Unmarshal(resp, &r)
	return r.Code == ""
}

func quarkCheck(url string) (ok bool) {
	var r QuarkResponse
	log.SetPrefix("quarkCheck():")
	re := regexp.MustCompile(`https://pan.quark.cn/s/(\w+)[\?]?`)
	pwd_id := re.FindStringSubmatch(url)[1]
	url = "https://pan.quark.cn/1/clouddrive/share/sharepage/token?pr=ucpro&fr=h5"
	param := map[string]string{
		"pwd_id": pwd_id,
	}
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Linux; Android 10; SM-G981B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.162 Mobile Safari/537.36",
		"Referer":    "https://pan.quark.cn",
	}
	data, _ := json.Marshal(param)
	resp, err := post(url, headers, bytes.NewReader(data))
	if err != nil {
		return false
	}
	json.Unmarshal(resp, &r)
	return (r.Code == 0 || r.Code == 41008)
}

func baiduYunCheck(url string) bool {
	log.SetPrefix("baiduYunCheck():")
	// 获取重定向地址
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1 Edg/94.0.4606.81",
	}

	resp, _, err := get(url, headers)
	if err != nil {
		return false
	}
	location, err := resp.Location()
	if err != nil {
		log.Print(err)
		return false
	}
	locationUrl := location.String()
	// 检测链接是否失效
	errorIndex := strings.Index(locationUrl, "error")
	return errorIndex == -1
}

func Check115(url string) bool {
	log.SetPrefix("Check115():")
	url = "https://webapi.115.com/share/snap?share_code=" + url[18:]
	_, data, err := get(url, nil)
	if err != nil {
		return false
	}
	errorIndex := strings.Index(string(data), `"errno":4100012`)
	return errorIndex != -1
}

func printColor(str string, ok bool) {
	if ok {
		fmt.Println(str)
	} else {
		if outputColor {
			fmt.Printf("\033[0;31;40m%s\033[0m\n", str)
		} else {
			fmt.Printf("%s x\n", str)
		}
	}
}

// 检测链接有效性
func (loli *Loli) checkUrl() {
	dominlist := []string{"pan.baidu", "aliyundrive", "115.com", "pan.quark"}
	loli.id = make(map[string]bool)
	var ok bool
	var repeatUrl int //重复链接计数
	basePath, _ := os.Executable()
	errPath := filepath.Join(filepath.Dir(basePath), "error.log")
	errFile, err := os.Create(errPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	log.SetPrefix("checkUrl():")
	log.SetOutput(errFile)
	defer func() {
		err := recover()
		if err != nil {
			log.Print(err)
		}
		errFile.Close()
	}()
	for count, url := range loli.urlList {
		// 去重
		if !loli.id[url] {
			loli.id[url] = true
		} else {
			str := fmt.Sprintf("%d 重复链接，已忽略 %s", count+1, url)
			printColor(str, false)
			repeatUrl++
			// count++
			continue
		}
		var index int
		for i, v := range dominlist {
			if index = strings.Index(url, v); index != -1 {
				index = i
				break
			}
		}
		var str string
		switch index {
		case 0:
			ok = baiduYunCheck(url) // 百度网盘检测
			str = fmt.Sprintf("%d %s %s", count+1, url, loli.Pwd[url])
			printColor(str, ok)
		case 1:
			ok = aliYunCheck(url) // 阿里云盘检测
			// 有提取码的加入提取码，没有的默认为空
			// 输出阿里云盘分享链接的文件名
			str = fmt.Sprintf("%d %s %s", count+1, url, loli.Pwd[url])
			printColor(str, ok)
		case 2:
			ok = Check115(url)
			str = fmt.Sprintf("%d %s %s", count+1, url, loli.Pwd[url])
			printColor(str, ok)
		case 3:
			ok = quarkCheck(url)
			str = fmt.Sprintf("%d %s%s", count+1, url, loli.Pwd[url])
			printColor(str, ok)
		}
		if ok {
			loli.okUrl = append(loli.okUrl, url)
		} else {
			loli.errUrl = append(loli.errUrl, url)
		}
	}
	okUrlPath := filepath.Join(filepath.Dir(basePath), "有效链接.txt")
	f, err := os.Create(okUrlPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, v := range loli.okUrl {
		_, err := f.WriteString(v + "\n")
		if err != nil {
			fmt.Println(err)
		}
	}
	f.Close()
	errUrlPath := filepath.Join(filepath.Dir(basePath), "失效链接.txt")
	f, err = os.Create(errUrlPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, v := range loli.errUrl {
		_, err := f.WriteString(v + "\n")
		if err != nil {
			fmt.Println(err)
		}
	}
	f.Close()
	fmt.Println("--------------------检测结果--------------------")
	fmt.Printf("有效链接：%d/%d\n", len(loli.okUrl), len(loli.urlList))
	fmt.Printf("失效链接：%d/%d\n", len(loli.errUrl), len(loli.urlList))
	if repeatUrl != 0 {
		fmt.Printf("重复链接：%d/%d\n", repeatUrl, len(loli.urlList))
	}
}

// 按任意键退出...
func enterKeyExit() {
	fmt.Print("按任意键退出...")
	fmt.Scanln()
	os.Exit(0)
}

// 读取url.txt文件里的链接
func (loli *Loli) getUrlList() error {
	var path string
	fmt.Print("请将要检测的文件拖动到这里:")
	fmt.Scanln(&path)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	loli.regexpUrl(data)
	return nil
}

// 正则匹配url
func (loli *Loli) regexpUrl(data []byte) {
	loli.Pwd = make(map[string]string)
	// 百度 阿里匹配链接
	rebaiduAli := regexp.MustCompile(`(http[s]?://[www pan]+.[a-z]+.com/s/[\w-]+)`)
	// 115匹配链接
	re115 := regexp.MustCompile("(https://115.com/s/.+?)[? #]+")
	// 夸克匹配链接
	reQuark := regexp.MustCompile(`https://pan.quark.cn/s/\w+`)
	// 提取码规则1 阿里
	rebaiduAliPwd := regexp.MustCompile(`(提取码: \w{4})\s?\n链接：(http[s]?://[www pan]+.[a-z]+.com/s/[\w-]+)`)
	// 提取码规则2	百度、阿里云都适用
	rebaiduAliPwd2 := regexp.MustCompile(`(http[s]?://[www pan]+.[a-z]+.com/s/[\w-]+)\s(提取码:[\s]?\w{4})`)
	// 115提取码规则1
	re115Pwd := regexp.MustCompile(`(https://115.com/s/\w+)#\r\n.+?\n访问码：(.{4})`)
	// 115提取码规则2
	re115Pwd2 := regexp.MustCompile(`(https://115.com/s/.+?)[? #]+password=(.{4})`)
	// 夸克提取码规则
	reQuarkPwd := regexp.MustCompile(`(https://pan.quark.cn/s/\w+)\?passcode=(.{4})`)
	resbaiduAli := rebaiduAli.FindAllSubmatch(data, -1)
	res115 := re115.FindAllSubmatch(data, -1)
	resQuark := reQuark.FindAllSubmatch(data, -1)
	resbaiduAliPwd := rebaiduAliPwd.FindAllSubmatch(data, -1)
	resbaiduAliPwd2 := rebaiduAliPwd2.FindAllSubmatch(data, -1)
	res115Pwd := re115Pwd.FindAllSubmatch(data, -1)
	res115Pwd2 := re115Pwd2.FindAllSubmatch(data, -1)
	resQuarkPwd := reQuarkPwd.FindAllSubmatch(data, -1)
	// 将匹配到的阿里、百度链接写入url.urlList
	for _, v := range resbaiduAli {
		url := strings.TrimSpace(string(v[1]))
		loli.urlList = append(loli.urlList, url)
	}
	// 115链接写入url.urlList
	for _, v := range res115 {
		url := strings.TrimSpace(string(v[1]))
		loli.urlList = append(loli.urlList, url)
	}
	// 夸克链接写入url.urlList
	for _, v := range resQuark {
		url := strings.TrimSpace(string(v[0]))
		loli.urlList = append(loli.urlList, url)
	}
	// 将百度、阿里提取码和链接写入map
	for _, v := range resbaiduAliPwd {
		url := strings.TrimSpace(string(v[2]))
		loli.Pwd[url] = string(v[1])
	}
	for _, v := range resbaiduAliPwd2 {
		url := strings.TrimSpace(string(v[1]))
		loli.Pwd[url] = string(v[2])
	}
	// 115提取码和链接写入map
	for _, v := range res115Pwd {
		url := strings.TrimSpace(string(v[1]))
		loli.Pwd[url] = "?password=" + string(v[2])
	}
	for _, v := range res115Pwd2 {
		url := strings.TrimSpace(string(v[1]))
		loli.Pwd[url] = "?password=" + string(v[2])
	}
	// 夸克提取码和链接写入map
	for _, v := range resQuarkPwd {
		url := strings.TrimSpace(string(v[1]))
		loli.Pwd[url] = "?passcode=" + string(v[2])
	}
}

// 检测版本
func init() {
	const version = "v2.1.1"
	url := "https://docs.qq.com/dop-api/opendoc?id=DT3NEWFlERWdsSU5l&normal=1"
	headers := map[string]string{
		"Referer":    "https://docs.qq.com/dop-api/opendoc?u=b428edf70fd8491680c0d496077dc2f0&id=DT3NEWFlERWdsSU5l&normal=1",
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.0.0 Safari/537.36",
	}
	_, data, err := get(url, headers)
	if err != nil {
		panic(err)
	}
	re := regexp.MustCompile("loli{(.+?),(.+?),(.+?)}loli")
	res := re.FindAllSubmatch(data, -1)
	ver := string(res[0][1])
	updateContent := string(res[0][2])
	link := strings.Split(string(res[0][3]), "\\n")
	if ver != version {
		fmt.Printf("当前版本：%s\n最新版本：%s\n更新内容：%s\n阿里云盘：%s\nGithub：%s\n\n", version, ver, updateContent, link[0], link[1])
		enterKeyExit()
	}
}

func main() {
	var loli Loli
	// 判断当前终端是否支持显示颜色
	if runtime.GOOS == "windows" {
		pid := os.Getppid()
		process, err := ps.FindProcess(pid)
		if err != nil {
			log.Println(err)
		}
		if pName := process.Executable(); pName == "cmd.exe" || pName == "explorer.exe" {
			outputColor = false
		}
	}
	fmt.Println("-------------百度、阿里、115、夸克网盘链接有效性检测-------------")
	fmt.Print("检测完自动导出有效链接\n")
	fmt.Println("------------------------------------------------")

	if err := loli.getUrlList(); err != nil {
		fmt.Println(err)
		enterKeyExit()
	}
	loli.checkUrl()
}
