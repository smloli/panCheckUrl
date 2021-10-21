package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"os"
	"path/filepath"
	"regexp"
)

type Url struct{
	urlList []string
}

// 检测链接有效性
func (url *Url) checkUrl(flag bool, path string) {
	// 有效列表
	oklist := make([]string, 1)
	// 为了获取重定向的location，要重新实现一个http.Client
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
		},
	}
	for _, _url := range (*url).urlList {
		// 访问网盘链接
		req, _ := http.NewRequest("GET", _url, nil)
		// UA必须是手机的，否则网页不会重定向
		req.Header.Add("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1 Edg/94.0.4606.81")
		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
			continue
		}
		defer resp.Body.Close()
		// 获取重定向地址
		location, err := resp.Location()
		if err != nil {
			log.Println(err)
			continue
		}
		locationUrl := location.String()
		// 检测链接是否失效
		index := strings.Index(locationUrl, "error")
		if index != -1 {
			log.Printf("%s  ❌\n", _url)
		} else {
			log.Printf("%s  ✔️\n", _url)
			// 当flag为true时，将有效链接写入oklist
			if flag == true {
				if oklist[0] == "" {
					oklist[0] = _url
					continue
				}
				oklist = append(oklist, []string{_url}...)
			}
		}
	}
	// 当flag为true时，将oklist里的内容写入到loli.txt
	if flag == true {
		f, err := os.Create(path + "loli.txt")
		if err != nil {
			log.Fatal(err)
		}
		for _, v := range oklist {
			_, err := f.WriteString(v + "\n")
			if err != nil {
				fmt.Println(err)
			}
		}
		f.Close()
	}
}

// 读取url.txt文件里的链接
func (url *Url) getUrlList(path string) {
    txtPath := path + "url.txt"
	f, err := os.Open(txtPath)
	if err != nil {
		log.Fatal(err)
	}
    fileinfo, err := os.Stat(txtPath)
    if err != nil {
        log.Fatal("软件运行目录里未找到文件url.txt")
    }
    data := make([]byte, fileinfo.Size())
    _, err = f.Read(data)
    if err != nil {
        log.Fatal(err)
    }
    url.regexpUrl(&data)
    f.Close()
}

// 正则匹配url
func (url *Url) regexpUrl(data *[]byte) {
	re, err := regexp.Compile("(http[s]?://pan.baidu.com/s/[\\d a-z A-Z _ -]+)")
	if err != nil {
		fmt.Println(err)
	}
	res := re.FindAllSubmatch(*data, -1)
    // 将匹配到的url写入到url.urlList
    for _, v := range res {
		_url := strings.TrimSpace(string(v[1]))
        if url.urlList[0] == "" {
			url.urlList[0] = _url
			continue
		}
		url.urlList = append(url.urlList, []string{_url}...)
	}
}

func main() {
	var url Url
	var num string
	var loli string
	var tmp string
	var flag bool  // 批量检测自动将有效链接写入文件
	url.urlList = make([]string, 1)
	// 自动匹配当前系统的路径分隔符
	urlPath := filepath.Dir(os.Args[0]) + filepath.FromSlash("/")
	fmt.Println("-------------百度网盘链接有效性检测-------------")
	fmt.Println()
	fmt.Println("-----------------支持的链接格式-----------------")
	fmt.Println("https://pan.baidu.com/s/1lXSQI-33cEXB8GMXNAFlrQ")
	fmt.Println("链接:https://pan.baidu.com/s/1U88Wwm560vbvyJX0cw9J-Q 提取码:7deh")
	fmt.Println("链接: http://pan.baidu.com/s/1c0Er78G 密码: 2cci")
	fmt.Println("链接: https://pan.baidu.com/s/1YZnL2-TC3Wy5bshU7fntxg 提取码: qku6 复制这段内容后打开百度网盘手机App，操作更方便哦")
	fmt.Println("------------------------------------------------")
	fmt.Print("0.单个检测\n1.批量检测（读取软件运行目录url.txt文件里的每一行链接，检测完自动将有效链接写入到loli.txt）\n")
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
			url.getUrlList(urlPath)
	}
	url.checkUrl(flag, urlPath)
}
