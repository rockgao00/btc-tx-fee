package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 输出CVS文件的每一行
func readCSV(filePath string) {
	dat, err := ioutil.ReadFile(filePath)
	rowvalue := 0.0
	sum, cutsum, cutav := 0.0, 0.0, 0.0
	csvline, i := 0, 0
	cutnum := 0
	slice := make([]float64, 1000)
	const rownum = 1 //读取csv文件的第2列数据

	if err != nil {
		log.Fatal(err)
	}
	r := csv.NewReader(strings.NewReader(string(dat[:])))

	for {
		record, err := r.Read()

		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		rowvalue, err = strconv.ParseFloat(record[rownum], 64)
		sum = rowvalue + sum

		if csvline > 999 {
			break
		}
		slice[csvline] = rowvalue
		csvline++
	}
	csvline = csvline - 1                            //剪掉首行
	cutnum = int(math.Ceil(float64(csvline) * 0.05)) //向上取整

	slice = slice[1 : csvline+1] //去掉首行数据

	sort.Float64s(slice)

	//求滤波后的和
	for i = cutnum; i < csvline-cutnum; i++ {
		cutsum = cutsum + slice[i]
		//fmt.Println(i,slice[i])
	}

	//求滤波后的均值占比
	cutav = cutsum * 100 / float64(csvline-2*cutnum) / 3.125
	fmt.Printf("滤除头尾5%%后的占比: %.5f%%\n", cutav)
	time.Sleep(100 * time.Second)
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("请输入顶部区块高度: ")
	startBlockStr, _ := reader.ReadString('\n')
	startBlockStr = strings.TrimSpace(startBlockStr)
	startBlock, err := parseBlockHeight(startBlockStr)
	if err != nil {
		fmt.Println("无效的起始区块高度")
		return
	}

	fmt.Print("请输入底部区块高度: ")
	endBlockStr, _ := reader.ReadString('\n')
	endBlockStr = strings.TrimSpace(endBlockStr)
	endBlock, err := parseBlockHeight(endBlockStr)
	if err != nil {
		fmt.Println("无效的结束区块高度")
		return
	}

	if startBlock < endBlock {
		fmt.Println("起始区块高度应大于或等于结束区块高度")
		return
	}

	fileName := "reward_fees.csv"

	// 创建CSV文件
	file, err := os.Create(fileName)
	if err != nil {
		fmt.Println("无法创建文件:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入CSV文件头
	writer.Write([]string{"height", "reward_fees"})
	writer.Flush()

	for i := startBlock; i >= endBlock; i-- {
		url := fmt.Sprintf("https://www.oklink.com/zh-hans/btc/block/%d", i)
		fmt.Println(url)
		fee, err := fetchFee(url)
		if err != nil {
			fmt.Printf("无法获取区块 %d 的数据: %v\n", i, err)
			continue
		}
		feeStr := fmt.Sprintf("%.8f", fee/100000000)
		writer.Write([]string{strconv.Itoa(i), feeStr})
		writer.Flush()
		//time.Sleep(1 * time.Second) // 添加延迟以避免请求过于频繁
	}
	fmt.Println("CSV文件已生成:", fileName)
	readCSV(fileName)
}

// parseBlockHeight 解析含有逗号的区块高度字符串
func parseBlockHeight(blockStr string) (int, error) {
	// 移除非数字字符
	re := regexp.MustCompile(`[^\d]`)
	cleanStr := re.ReplaceAllString(blockStr, "")
	return strconv.Atoi(cleanStr)
}

// fetchFee 从指定URL获取fee字段
func fetchFee(url string) (float64, error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP请求失败: %s", resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0, err
	}

	html, err := doc.Html()
	if err != nil {
		return 0, err
	}

	// 使用正则表达式提取fee字段
	re := regexp.MustCompile(`"fee":(\d+)`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		return 0, fmt.Errorf("未找到fee字段")
	}

	fee, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, err
	}

	return fee, nil
}
