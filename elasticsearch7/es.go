package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/olivere/elastic/v7"
	"log"
	"strconv"
	"strings"
	"sync"
)

var ctx = context.Background()
var esUrl = "http://127.0.0.1:7106"
var EsClient *elastic.Client

// 初始化es连接
func InitEs() {

	client, err := elastic.NewClient(
		elastic.SetURL(esUrl),
	)
	if err != nil {
		log.Fatal("es 连接失败:", err)
	}
	// ping通服务端，并获得服务端的es版本,本实例的es版本为version 7.6.1
	info, code, err := client.Ping(esUrl).Do(ctx)
	if err != nil {
		// Handle error
		panic(err)
	}
	fmt.Println("Elasticsearch returned with code:\n", code, info.Version.Number)

	EsClient = client
	fmt.Println("es连接成功")
}

// 数据结构
type data struct {
	Id   string `json:"id"`
	Icon string `json:"icon"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// 定义一些变量，mapping为定制的index字段类型
const mapping = `
{
	"settings":{
		"number_of_shards": 1,
		"number_of_replicas": 0
	},
	"mappings":{
			"properties":{
				"name":{
					"type":"keyword"
				},
				"icon":{
					"type":"text"
				},
				"age":{
					"type":"long"
				},
				"id":{
					"type":"text"
				}
			}
	}
}`

func main() {
	// 初始化es配置
	InitEs()
	// 批量添加数据
	//g.Add(2)
	//go batchAdd()
	//go batchCreate()
	//g.Wait()

	// 添加数据
	//	data := data{
	//		Id:   "1",
	//		Icon: "头像",
	//		Name: "名称",
	//		Age:  10,
	//	}
	//	res ,err := Add("test", data)

	// 批量添加
	//var d []interface{}
	//for i := 0;i < 100; i++ {
	//	iStr := strconv.Itoa(i)
	//	v := data{
	//		Id: iStr,
	//		Icon: "icon " + iStr,
	//		Name: "name " + iStr,
	//		Age: i,
	//	}
	//	d = append(d, v)
	//}
	//res ,err := BulkAdd("test", d)

	// 更新数据
	//table := "test"
	//filter := elastic.NewTermQuery("name", "name 0")
	//data := make(map[string]interface{})
	//data["icon"] = "world10"
	//data["age"] = 10
	//res, err := UpWhere(table, filter, data)

	// 删除数据
	filter := elastic.NewTermQuery("name", "name 1")
	res, err := Del("test", filter)

	// 查询
	// 精准查询
	//filter := elastic.NewTermQuery("name", "李白")

	// 分词匹配查询
	// match_all 查询所有
	// filter := elastic.NewMatchAllQuery()
	// match 单个字符匹配
	// filter := elastic.NewMatchQuery("name", "渡汉江")
	// multi_match 多个字段进行匹配查询
	//filter := elastic.NewMultiMatchQuery("白日", "name", "icon")
	// match_phrase 短语匹配
	//filter := elastic.NewMatchPhraseQuery("icon", "途穷反遭俗眼白")

	// fuzzy模糊查询 fuzziness是可以允许纠正错误拼写的个数
	//filter := elastic.NewFuzzyQuery("icon", "夜").Fuzziness(1)

	// wildcard 通配符查询
	//filter := elastic.NewWildcardQuery("name", "静夜*")

	//table := "test"
	//sort := "age asc"
	//page := 0
	//limit := 10
	//field := "name,icon,age"
	//res, err := Query(table, field, filter, sort, page, limit)
	//strD, _ := json.Marshal(res)
	//fmt.Println(string(strD), err)

	if err != nil {
		fmt.Println("失败:", err)
	} else {
		fmt.Println("成功:", res)
	}
	fmt.Println("执行完成")
}

var batchData = make(chan data)
var g sync.WaitGroup

func batchAdd() {
	for i := 1; i <= 1000000; i++ {
		istr := strconv.Itoa(i)
		d := data{
			Id:   "1" + istr,
			Icon: "icon " + istr,
			Name: "name " + istr,
			Age:  i,
		}
		batchData <- d
	}
	g.Done()
}

// 批量添加
func batchCreate() {
	var d []interface{}
	for i := 1; i <= 1000000; i++ {
		d = append(d, <-batchData)
		if i%10000 == 0 {
			res, err := BulkAdd("test", d)
			if err != nil {
				fmt.Println("添加失败:", err)
			}
			d = d[0:0]
			fmt.Println("添加成功:", res)
		}
	}
	g.Done()
}

// 条件更新
func UpWhere(table string, filter elastic.Query, data map[string]interface{}) (bool, error) {
	// 修改数据组装
	if len(data) < 0 {
		return false, errors.New("修改参数不正确")
	}
	scriptStr := ""
	for k := range data {
		scriptStr += "ctx._source." + k + " = params." + k + ";"
	}
	script := elastic.NewScript(scriptStr).Params(data)
	res, err := EsClient.UpdateByQuery(table).
		Query(filter).
		Script(script).
		Do(ctx)
	if err != nil {
		return false, err
	}
	fmt.Println("添加数据成功:", res)
	return true, nil
}

// 批量添加
func BulkAdd(table string, d []interface{}) (bool, error) {
	// 添加索引
	_, err := Addindex("test")
	if err != nil {
		log.Fatal("创建索引失败", err)
	}
	bulkReq := EsClient.Bulk()
	for _, v := range d {
		req := elastic.NewBulkIndexRequest().
			Index(table).
			Doc(v)
		bulkReq = bulkReq.Add(req)
	}
	res, err := bulkReq.Do(ctx)
	if err != nil {
		return false, err
	}
	fmt.Println("添加数据成功:", res)
	return true, nil
}

// 添加数据
func Add(table string, data interface{}) (bool, error) {
	// 添加索引
	_, err := Addindex(table)
	if err != nil {
		log.Fatal("创建索引失败", err)
	}
	// 添加文档
	res, err := EsClient.Index().
		Index(table).
		BodyJson(data).
		Do(ctx)
	if err != nil {
		return false, err
	}
	fmt.Println("添加数据成功:", res)
	return true, nil
}

// 添加文档
func Addindex(table string) (bool, error) {
	// 创建index前，先查看es引擎中是否存在自己想要创建的索引index
	exists, err := EsClient.IndexExists(table).Do(ctx)
	if err != nil {
		fmt.Println("存在索引:", err)
		return true, nil
	}
	if !exists {
		// 如果不存在，就创建
		createIndex, err := EsClient.CreateIndex(table).BodyString(mapping).Do(ctx)
		if err != nil {
			return false, err
		}
		if !createIndex.Acknowledged {
			return false, err
		}
	}
	return true, nil
}

// 修改文档
func Update(table, id string, age int) (bool, error) {
	res, err := EsClient.Update().Index(table).Id(id).
		Script(elastic.NewScriptInline("ctx._source.age = params.num").Lang("painless").Param("num", age)).
		Do(ctx)
	if err != nil {
		return false, err
	}
	fmt.Println("更新信息：", res)
	return true, nil
}

// 删除文档
func Del(table string, filter elastic.Query) (bool, error) {
	res, err := EsClient.DeleteByQuery().
		Query(filter).
		Index(table).
		Do(ctx)
	if err != nil {
		return false, err
	}
	fmt.Println("删除信息：", res)
	return true, nil
}

// 查询数据
func Query(table string, field string, filter elastic.Query, sort string, page int, limit int) (*elastic.SearchResult, error) {
	// 分页数据处理
	isAsc := true
	if sort != "" {
		sortSlice := strings.Split(sort, " ")
		sort = sortSlice[0]
		if sortSlice[1] == "desc" {
			isAsc = false
		}
	}
	// 查询位置处理
	if page <= 1 {
		page = 1
	}

	fsc := elastic.NewFetchSourceContext(true)
	// 返回字段处理
	if field != "" {
		fieldSlice := strings.Split(field, ",")
		if len(fieldSlice) > 0 {
			for _, v := range fieldSlice {
				fsc.Include(v)
			}
		}
	}

	// 开始查询位置
	fromStart := (page - 1) * limit
	res, err := EsClient.Search().
		Index(table).
		FetchSourceContext(fsc).
		Query(filter).
		Sort(sort, isAsc).
		From(fromStart).
		Size(limit).
		Pretty(true).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	return res, nil
}
