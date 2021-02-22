package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {
	server()
}

// 上传
func upload(w http.ResponseWriter, r *http.Request) {
	res, err := upload2(r)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
	} else {
		// 保存上传节点
		_, _ = w.Write([]byte("上传成功:" + strconv.Itoa(res)))
	}
}

func upload2(r *http.Request) (int, error) {
	// 获取上传文件
	upfile, fileHeader, err := r.FormFile("file")
	if err != nil {
		return 0, errors.New("上传文件错误")
	}
	headerByte, _ := json.Marshal(fileHeader.Header)
	log.Printf("当前文件：Filename - >{%s}, Size -> {%v}, FileHeader -> {%s}", fileHeader.Filename, fileHeader.Size, string(headerByte))

	// 新文件创建
	filePath := "./" + fileHeader.Filename
	fileBool, err := isFileExists(filePath)
	if fileBool && err == nil {
		fmt.Println("文件已存在")
	} else {
		newfile, err := os.Create(filePath)
		defer newfile.Close()
		if err != nil {
			return 0, errors.New("创建文件失败")
		}
	}
	// 获取现在文件大小
	fi, _ := os.Stat(filePath)
	// 判断文件是否传输完成
	if fi.Size() == fileHeader.Size {
		return 0, errors.New("文件已存在, 不继续上传")
	}
	start := strconv.Itoa(int(fi.Size()))

	// 进行断点上传
	// 打开之前上传文件
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	defer file.Close()
	if err != nil {
		return 0, errors.New("打开之前上传文件不存在")
	}

	// 将数据写入文件
	count, _ := strconv.ParseInt(start, 10, 64)
	fmt.Println("已上传:", count)
	// 设置读，写的偏移量
	upfile.Seek(count, 0)
	file.Seek(count, 0)
	data := make([]byte, 1024, 1024)
	upTotal := 0
	for {
		total, err := upfile.Read(data)
		if err == io.EOF {
			fmt.Println("文件复制完毕")
			break
		}
		len, err := file.Write(data[:total])
		if err != nil {
			return 0, errors.New("文件上传失败")
		}
		upTotal += len
		// 记录上传长度
		count += int64(len)

		// 模拟断开
		//if count > 4438903 {
		//	log.Fatal("模拟上传中断")
		//}
	}
	fmt.Println("文件上传长度:", upTotal)
	return upTotal, nil
}

// 上传方法1
func upload1(r *http.Request) (int, error) {
	// 获取上传文件
	upfile, fileHeader, err := r.FormFile("file")
	if err != nil {
		return 0, errors.New("上传文件错误")
	}
	headerByte, _ := json.Marshal(fileHeader.Header)
	log.Printf("当前文件：Filename - >{%s}, Size -> {%v}, FileHeader -> {%s}", fileHeader.Filename, fileHeader.Size, string(headerByte))

	// 新文件创建
	filePath := "./" + fileHeader.Filename
	fileBool, err := isFileExists(filePath)
	if fileBool && err == nil {
		fmt.Println("文件已存在")
	} else {
		newfile, err := os.Create(filePath)
		defer newfile.Close()
		if err != nil {
			return 0, errors.New("创建文件失败")
		}
	}
	// 上传文件开始位置
	start := "0"
	// 查询本地是否有断点记录
	var breakPointFile *os.File
	breakPointFilePath := filePath + ".txt"
	breakPointFileBool, err := isFileExists(breakPointFilePath)

	// 如果文件存在 但断点记录不存在  则文件已上传
	if fileBool && !breakPointFileBool {
		return 0, errors.New("文件已存在, 不继续上传")
	}

	//fmt.Println("断点文件检测:", breakPointFileBool,err)
	if breakPointFileBool && err == nil {
		// 读取断点位置
		breakPointFile, err = os.OpenFile(breakPointFilePath, os.O_CREATE|os.O_RDWR, os.ModePerm)
		defer breakPointFile.Close()
		if err != nil {
			return 0, errors.New("断点文件打开失败")
		}
		res, _ := ioutil.ReadAll(breakPointFile)
		fmt.Println("断点文件写入字段长度:", string(res))
		start = string(res)
	} else {
		// 创建断点文件
		breakPointFile, err = os.Create(breakPointFilePath)
		defer breakPointFile.Close()
		if err != nil {
			return 0, errors.New("断点文件创建失败")
		}
		len, err := breakPointFile.WriteString("0")
		if len <= 0 || err != nil {
			return 0, errors.New("断点文件写入失败")
		}
		fmt.Println("断点文件创建成功")
	}

	// 进行断点上传
	// 打开之前上传文件
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	defer file.Close()
	if err != nil {
		return 0, errors.New("打开之前上传文件不存在")
	}

	// 将数据写入文件
	count, _ := strconv.ParseInt(start, 10, 64)
	fmt.Println("已上传:", count)
	// 设置读，写的偏移量
	upfile.Seek(count, 0)
	file.Seek(count, 0)
	data := make([]byte, 1024, 1024)
	upTotal := 0
	for {
		total, err := upfile.Read(data)
		if err == io.EOF {
			// 删除文件 需要先关闭改文件
			breakPointFile.Close()
			err := os.Remove(breakPointFilePath)
			if err != nil {
				fmt.Println("临时记录文件删除失败", err)
			}
			fmt.Println("文件复制完毕")
			break
		}
		len, err := file.Write(data[:total])
		if err != nil {
			return 0, errors.New("文件上传失败")
		}
		upTotal += len
		// 记录上传长度
		count += int64(len)
		breakPointFile.Seek(0, 0)
		breakPointFile.WriteString(strconv.Itoa(int(count)))

		// 模拟断开
		//if count > 4438903 {
		//	log.Fatal("模拟上传中断")
		//}
	}
	fmt.Println("文件上传长度:", upTotal)
	return upTotal, nil
}

// 判断文件或文件夹是否存在
func isFileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, err
	}
	return false, err
}

// http 服务
func server() {
	// 注册一个上传文件服务
	http.HandleFunc("/upload", upload)

	// 监听8000端口
	err := http.ListenAndServe("0.0.0.0:8000", nil)
	if err != nil {
		log.Fatal("服务启动失败")
	}
}
