package main

import (
	"encoding/json"
	"fmt"
	"log"
)

func main() {
	// 假设这是 API 返回的 JSON 字节数据
	jsonResponse := []byte(`{
		"status": 200,
		"message": "Success",
		"data": {
			"user_id": 1001,
			"username": "Gopher",
			"score": 98.5,
			"tags": ["golang", "dev", "api"]
		}
	}`)

	// 1. 定义目标 map 变量
	// interface{} 可以存储任何类型的值
	var result map[string]interface{} 

	// 2. 将 JSON 数据解析到 map 中
	err := json.Unmarshal(jsonResponse, &result)
	if err != nil {
		log.Fatalf("JSON 解析失败: %v", err)
	}

	fmt.Println("✅ JSON 成功解析为 map[string]interface{}")
	fmt.Printf("整个 Map: %+v\n", result)

	// 3. 如何安全地访问数据

	// 访问顶层键值
	status := result["status"] // JSON 数字默认解析为 float64
	fmt.Printf("\n状态码 (Type: %T): %.0f\n", status, status)

	// 访问嵌套的 'data'
	dataMap, ok := result["data"].(map[string]interface{})
	if !ok {
		log.Fatal("无法断言 data 键为 map")
	}

	// 访问嵌套 map 中的键值
	username := dataMap["username"].(string) // 字符串默认解析为 string
	fmt.Printf("用户名 (Type: %T): %s\n", username, username)

	// 访问数组
	tags, ok := dataMap["tags"].([]interface{}) // 数组默认解析为 []interface{}
	if ok {
		fmt.Printf("第一个 Tag (Type: %T): %s\n", tags[0].(string), tags[0].(string))
	}
}