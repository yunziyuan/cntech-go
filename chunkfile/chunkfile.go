package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"sync"
)

const tmpFilePath = "./tmp/"

var lock sync.WaitGroup

func main() {
	fileServer()
}

// 文件服务
func fileServer() {
	http.HandleFunc("/chunkfile", chunkFile)
	// 监听8001端口
	err := http.ListenAndServe("0.0.0.0:8001", nil)
	if err != nil {
		log.Fatal("服务启动失败")
	}
}

// 上传文件
func chunkFile(w http.ResponseWriter, r *http.Request) {
	// 设置跨域
	w.Header().Set("Access-Control-Allow-Origin", "*")             //允许访问所有域
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type") //header的类型
	w.Header().Set("content-type", "application/json")             //返回数据格式是json
	// 合并
	res, err := mergeChunk(r)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
	} else {
		// 保存上传节点
		_, _ = w.Write([]byte("上传成功:" + strconv.Itoa(res)))
	}
}

// 合并文件
func mergeChunk(r *http.Request) (int, error) {
	// 上传文件
	_, err := chunkUpload(r)
	if err != nil {
		return 0, errors.New("上传失败")
	}
	// 分片总数
	chunkTotal := r.FormValue("chunktotal")
	// 文件总大小
	fileSize := r.FormValue("filesize")
	// 获取上传文件
	_, fileHeader, err := r.FormFile("file")
	if err != nil {
		return 0, errors.New("上传文件错误")
	}
	// 分片序号与分片总数相等 则合并文件
	total, _ := strconv.Atoi(chunkTotal)
	size, _ := strconv.Atoi(fileSize)
	// 上传总数
	totalLen := 0
	// 最后一个分片上传时,进行分片合并成一个文件
	if isFnish(fileHeader.Filename, total, size) {
		// 新文件创建
		filePath := "./" + fileHeader.Filename
		fileBool, err := createFile(filePath)
		if !fileBool {
			return 0, err
		}
		// 读取文件片段 进行合并
		for i := 0; i < total; i++ {
			lock.Add(1)
			go mergeFile(i, fileHeader.Filename, filePath)
		}
		lock.Wait()
	}
	return totalLen, nil
}

// 判断是否完成  根据现有文件的大小 与 上传文件大小进行匹配
func isFnish(fileName string, chunkTotal, fileSize int) bool {
	var chunkSize int64
	for i := 0; i < chunkTotal; i++ {
		iStr := strconv.Itoa(i)
		// 分片大小获取
		fi, err := os.Stat(tmpFilePath + fileName + "_" + iStr)
		if err == nil {
			chunkSize += fi.Size()
		}
	}
	if chunkSize == int64(fileSize) {
		return true
	}
	return false
}

// 合并切片文件
func mergeFile(i int, fileName, filePath string) {
	// 打开之前上传文件
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	defer file.Close()
	if err != nil {
		log.Fatal("打开之前上传文件不存在")
		//return 0, errors.New("打开之前上传文件不存在")
	}
	// 分片大小获取
	fi, _ := os.Stat(tmpFilePath + fileName + "_0")
	chunkSize := fi.Size()
	// 设置文件写入偏移量
	file.Seek(chunkSize*int64(i), 0)
	iSize := strconv.Itoa(i)
	chunkFilePath := tmpFilePath + fileName + "_" + iSize
	fmt.Printf("分片路径:", chunkFilePath)
	chunkFileObj, err := os.Open(chunkFilePath)
	defer chunkFileObj.Close()
	if err != nil {
		log.Fatal("打开分片文件失败")
		//return 0, errors.New("打开分片文件失败")
	}

	// 上传总数
	totalLen := 0
	// 写入数据
	data := make([]byte, 1024, 1024)
	for {
		tal, err := chunkFileObj.Read(data)
		if err == io.EOF {
			// 删除文件 需要先关闭改文件
			chunkFileObj.Close()
			err := os.Remove(chunkFilePath)
			if err != nil {
				fmt.Println("临时记录文件删除失败", err)
			}
			fmt.Println("文件复制完毕")
			break
		}
		len, err := file.Write(data[:tal])
		if err != nil {
			log.Fatal("文件上传失败")
			//return 0, errors.New("文件上传失败")
		}
		totalLen += len
	}
	lock.Done()
	//return totalLen,nil
}

// 分片上传
func chunkUpload(r *http.Request) (int, error) {
	// 分片序号
	chunkIndex := r.FormValue("chunkindex")
	// 获取上传文件
	upFile, fileHeader, err := r.FormFile("file")

	if err != nil {
		return 0, errors.New("上传文件错误")
	}

	// 新文件创建
	filePath := tmpFilePath + fileHeader.Filename + "_" + chunkIndex
	fileBool, err := createFile(filePath)
	if !fileBool {
		return 0, err
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
	total, err := uploadFile(upFile, count, file, count)
	return total, err
}

// 上传文件
func uploadFile(upfile multipart.File, upSeek int64, file *os.File, fSeek int64) (int, error) {
	// 上传文件大小记录
	fileSzie := 0
	// 设置上传偏移量
	upfile.Seek(upSeek, 0)
	// 设置文件偏移量
	file.Seek(fSeek, 0)
	data := make([]byte, 1024, 1024)
	for {
		total, err := upfile.Read(data)
		if err == io.EOF {
			//fmt.Println("文件复制完毕")
			break
		}
		len, err := file.Write(data[:total])
		if err != nil {
			return 0, errors.New("文件上传失败")
		}
		// 记录上传长度
		fileSzie += len
	}
	return fileSzie, nil
}

// 创建文件
func createFile(filePath string) (bool, error) {
	fileBool, err := fileExists(filePath)
	if fileBool && err == nil {
		return true, errors.New("文件以存在")
	} else {
		newFile, err := os.Create(filePath)
		defer newFile.Close()
		if err != nil {
			return false, errors.New("创建文件失败")
		}
	}
	return true, nil
}

// 判断文件或文件夹是否存在
func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, err
	}
	return false, err
}
