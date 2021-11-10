package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"os"
	"regexp"
	"encoding/json"
	"io/ioutil"
)

type Url struct{
	urlList []string // 链接列表
	id map[string]bool // 链接ID
	validUrl []string // 有效链接
	errUrl []string // 无效链接
	Pwd map[string]string	//提取码map
}

// 阿里返回状态码
type RespCode struct{
    Code string
	Share_name string
}

// 为了获取重定向的location，要重新实现一个http.Client
var client = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
	},
}

func aliYunCheck(_url *string) (start string, shareName string) {
	log.SetPrefix("aliYunCheck():")
    share_id := (*_url)[30:]
    var respcode RespCode
    url := "https://api.aliyundrive.com/adrive/v3/share_link/get_share_by_anonymous?share_id=" + share_id
    param := map[string]string{
        "share_id": share_id,
    }
    jsonParam, _ := json.Marshal(param)
    req, _ := http.NewRequest("POST", url, strings.NewReader(string(jsonParam)))
    req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 11; SM-G9880) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.37 Mobile Safari/537.36")
    req.Header.Set("Referer", "https://www.aliyundrive.com/")
    resp, err := client.Do(req)
    if err != nil {
        log.Print(err)
		return
    }
    defer resp.Body.Close()
    body, _ := ioutil.ReadAll(resp.Body)
    json.Unmarshal(body, &respcode)
	if respcode.Code == "" {
		start = "√"
		shareName = respcode.Share_name
	} else {
		start = "×"
	}
    return
}

func baiduYunCheck(_url *string) (start string) {
	log.SetPrefix("baiduYunCheck():")
	// 访问网盘链接
	req, _ := http.NewRequest("GET", *_url, nil)
	// UA必须是手机的，否则网页不会重定向
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1 Edg/94.0.4606.81")
	resp, err := client.Do(req)
	if err != nil {
		log.Print(err)
		return
	}
	defer resp.Body.Close()
	// 获取重定向地址
	location, err := resp.Location()
	if err != nil {
		log.Print(err)
		return
	}
	locationUrl := location.String()
	// 检测链接是否失效
	index := strings.Index(locationUrl, "error")
	if index != -1 {
		start = "×"
	} else {
		start = "√"
	}
	return
}

// 检测链接有效性
func (url *Url) checkUrl(flag bool) {
	// 有效列表
	url.validUrl = make([]string, 1)
	url.id = make(map[string]bool)
	url.errUrl = make([]string, 1)
	var start string
	var shareName string
	var repeatUrl int //重复链接计数
	count := 1	//链接计数
	ferror, err := os.OpenFile("error.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
	log.SetPrefix("checkUrl():")
	log.SetOutput(ferror)
	defer func() {
		err := recover()
		if err != nil {
			log.Print(err)
		}
		ferror.Close()
	}()
	for _, _url := range (*url).urlList {
		// 去重
		if !url.id[_url] {
			url.id[_url] = true
		} else {
			fmt.Printf("发现重复链接，已跳过！  %s \n", _url)
			repeatUrl++
			continue
		}
		index := strings.Index(_url, "baidu")
		if index != -1 {
			start = baiduYunCheck(&_url)	// 百度网盘检测
			if start == "" {
				continue
			}
			_url += " " + url.Pwd[_url]
			fmt.Printf("%d  %s  %s\n", count, _url, start)
		} else {
			start, shareName = aliYunCheck(&_url)	// 阿里云盘检测
			// 输出阿里云盘分享链接的文件名
			if start == "√" {
				// 有提取码的加入提取码，没有的默认为空
				_url = shareName + " " + _url + " " + url.Pwd[_url]
			} else if start == "" {
				continue
			}
			fmt.Printf("%d  %s  %s\n", count, _url, start)
		}
		count++
		// flag == true 就记录
		if flag {
			if start == "√" {
				if url.validUrl[0] == "" {
					url.validUrl[0] = _url
					continue
				}
				url.validUrl = append(url.validUrl, []string{_url}...)
			} else if start == "×"{
				if url.errUrl[0] == "" {
					url.errUrl[0] = _url
					continue
				}
				url.errUrl = append(url.errUrl, []string{_url}...)
			}
		}
	}
	// 当flag为true时，将oklist里的内容写入到loli.txt
	// 失效链接写入失效链接.txt
	if flag {
		floli, err := os.Create("loli.txt")
		if err != nil {
			fmt.Println(err)
			return
		}
		for _, v := range url.validUrl {
			_, err := floli.WriteString(v + "\n")
			if err != nil {
				fmt.Println(err)
			}
		}
		floli.Close()
		ferrUrl, err := os.Create("失效链接.txt")
		if err != nil {
			fmt.Println(err)
			return
		}
		for _, v := range url.errUrl {
			_, err := ferrUrl.WriteString(v + "\n")
			if err != nil {
				fmt.Println(err)
			}
		}
		ferrUrl.Close()
	}
	fmt.Println("--------------------检测结果--------------------")
	fmt.Printf("有效链接：%d/%d\n", len(url.validUrl), len(url.urlList))
	fmt.Printf("失效链接：%d/%d\n", len(url.errUrl), len(url.urlList))
	if repeatUrl != 0 {
		fmt.Printf("重复链接：%d/%d\n", repeatUrl, len(url.urlList))
	}
}

// 读取url.txt文件里的链接
func (url *Url) getUrlList() {
	f, err := os.Open("url.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
    fi, _ := f.Stat()
    if err != nil {
        log.Fatal("url.txt文件不存在")
    }
    data := make([]byte, fi.Size())
    _, err = f.Read(data)
    if err != nil {
        log.Fatal(err)
    }
    url.regexpUrl(&data)
}

// 正则匹配url
func (url *Url) regexpUrl(data *[]byte) {
	url.Pwd = make(map[string]string)
	// 无提取码规则
	re := regexp.MustCompile("(http[s]?://[www pan]+.[a-z]+.com/s/[0-9a-zA-Z_-]+)")
	// 提取码规则1
	rePwd1 := regexp.MustCompile(`(提取码: [0-9a-zA-Z]{4})\s?\n链接：(http[s]?://[www pan]+.[a-z]+.com/s/[0-9a-zA-Z_-]+)`)
	// 提取码规则2	百度、阿里云都适用
	rePwd2 := regexp.MustCompile(`(http[s]?://[www pan]+.[a-z]+.com/s/[0-9a-zA-Z_-]+)\s(提取码:[\s]?[0-9a-zA-Z]{4})`)
	res := re.FindAllSubmatch(*data, -1)
	resPwd1 := rePwd1.FindAllSubmatch(*data, -1)
	resPwd2 := rePwd2.FindAllSubmatch(*data, -1)
    // 将匹配到的url写入到url.urlList
    for _, v := range res {
		_url := strings.TrimSpace(string(v[1]))
        if url.urlList[0] == "" {
			url.urlList[0] = _url
			continue
		}
		url.urlList = append(url.urlList, []string{_url}...)
	}
	// 将有提取码的链接写到map里
	for _, v := range resPwd1 {
		_url := strings.TrimSpace(string(v[2]))
		url.Pwd[_url] = string(v[1])
	}
	for _, v := range resPwd2 {
		_url := strings.TrimSpace(string(v[1]))
		url.Pwd[_url] = string(v[2])
	}
}

func main() {
	var url Url
	var num string
	var loli string
	var tmp string
	var flag bool  // 检测模式
	url.urlList = make([]string, 1)
	fmt.Println("-------------百度、阿里云盘链接有效性检测-------------")
	fmt.Println()
	fmt.Println("-----------------支持的链接格式-----------------")
	fmt.Println("https://pan.baidu.com/s/1lXSQI-33cEXB8GMXNAFlrQ")
	fmt.Println("链接:https://pan.baidu.com/s/1U88Wwm560vbvyJX0cw9J-Q 提取码:7deh")
	fmt.Println("链接: http://pan.baidu.com/s/1c0Er78G 密码: 2cci")
	fmt.Println("链接: https://pan.baidu.com/s/1YZnL2-TC3Wy5bshU7fntxg 提取码: qku6 复制这段内容后打开百度网盘手机App，操作更方便哦")
	fmt.Println("https://www.aliyundrive.com/s/6riFVSGytcv")
	fmt.Println("我用阿里云盘分享了「loli.7z.png」，你可以不限速下载🚀 复制这段内容打开「阿里云盘」App 即可获取 链接：https://www.aliyundrive.com/s/bEBTKwaCK4K")
	fmt.Println("------------------------------------------------")
	fmt.Print("0.单个检测\n1.批量检测（读取软件运行目录url.txt文件里的每一行链接，检测完自动将有效链接导出至loli.txt）\n")
	fmt.Println("------------------------------------------------")
	fmt.Print("num:")
	fmt.Scanln(&num)
	switch num {
		case "0":
			fmt.Print("url:")
			// 处理字符串里的空格,然后拼接
			for {
				n, _ := fmt.Scanf("%s", &tmp)
				if n == 0 {
					break
				}
				loli += tmp + " "
			}
            urlData := []byte(loli)
            url.regexpUrl(&urlData)
		case "1":
			flag = true
			url.getUrlList()
	}
	url.checkUrl(flag)
}
