package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"log"
	"math"
)

var RedisConn *redis.Client
var ctx = context.Background()

func InitRedis() {
	RedisConn = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:7109",
		Password: "",
		DB:       0,
	})
	//defer RedisConn.Close()
}

func main() {
	InitRedis()
	// 初始化点赞
	key := "click2"
	//bol, err := initClick(key, 33)
	//if !bol || err != nil {
	//	log.Fatal("初始化数据失败:",err)
	//}

	// 点赞
	bol, err := clickSet(key, 87, 1)
	if !bol || err != nil {
		log.Fatal("点赞失败:", err)
	}
	fmt.Println("点赞成功")

	// 统计点赞
	num, err := clickNum(key)
	if err != nil {
		log.Fatal("点赞获取失败:", err)
	}
	fmt.Println("点赞人数:", num)
}

// 初始化点赞数据 key: 存在业务 total: 存储总数量
// 使用0字符串的二进制作为二进制
func initClick(key string, total int64) (bool, error) {
	// 拼接字符长度
	if total <= 0 {
		return false, errors.New("总长度不正确")
	}
	str := "0"
	var l int
	if total > 8 {
		l = int(math.Ceil(float64(total) / 8.00))
		for i := 1; i < l; i++ {
			str += "0"
		}
	}
	// 计算需要置0的位置
	var sitZeros []int64
	for i := 0; i < l*8; i++ {
		fSit := int64(2 + 8*i)
		if fSit >= int64(l*8) {
			break
		}
		sits := []int64{
			fSit, fSit + 1,
		}
		sitZeros = append(sitZeros, sits...)
	}
	// 字符:      0000
	// 对应二进制: 00110000 00110000 00110000 00110000
	// 对应下标:   01234567 8.....15 16....23 24....31
	err := RedisConn.Set(ctx, key, str, 0).Err()
	if err != nil {
		fmt.Println("设置失败:", err)
	}
	// 将对应1的位置修改为0
	// 需要置0的位置 2,3,10,11,18,19,26,27
	for _, v := range sitZeros {
		err = RedisConn.SetBit(ctx, key, v, 0).Err()
		if err != nil {
			fmt.Println("置0失败:", err)
			return false, err
		}
	}
	return true, nil
}

// 获取点赞数量 使用32个二进制位表示
func clickNum(key string) (int64, error) {
	// 统计所有的位置的1的数量
	// 字符:        0000
	// 对应位置下标: 0123
	l, err := RedisConn.StrLen(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	sit := redis.BitCount{Start: 0, End: l - 1}
	num, err := RedisConn.BitCount(ctx, key, &sit).Result()
	if err != nil {
		return 0, err
	}
	return num, nil
}

// 点赞设置 sit:点赞下标  value 设置值 只能是 0或者1
func clickSet(key string, sit int64, value int) (bool, error) {
	err := RedisConn.SetBit(ctx, key, sit, value).Err()
	if err != nil {
		fmt.Println("置0失败:", err)
		return false, err
	}
	return true, nil
}

func stringToBin(s string) (binString string) {
	for _, c := range s {
		binString = fmt.Sprintf("%s%b", binString, c)
	}
	return
}

func add() {
	InitRedis()
	err := RedisConn.Set(ctx, "name", "那么慢", 0).Err()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("add success")
	val, err := RedisConn.Get(ctx, "name").Result()
	fmt.Println(val, err)
}
