package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	jsonString := `{
		"status": "success",
		"data": {
			"user_id": 1001,
			"profile": {
				"name": "Gemini",
				"settings": [
					{"theme": "dark", "enabled": true},
					{"theme": "light", "enabled": false}
				]
			}
		}
	}`

	// 声明一个 map 来接收反序列化的数据
	var dataMap map[string]interface{}

	// 执行 Unmarshal
	if err := json.Unmarshal([]byte(jsonString), &dataMap); err != nil {
		fmt.Println("JSON Unmarshal 错误:", err)
		return
	}

	// 2. **标准库 Map 的访问方式 (需要类型断言)**
	// 访问路径: data -> profile -> settings -> 第一个元素 [0] -> theme 字段

	// 1. 访问 "data" 字段（类型断言为 map[string]interface{}）
	data, ok := dataMap["data"].(map[string]interface{})
	if !ok { /* 错误处理 */ }

	// 2. 访问 "profile" 字段
	profile, ok := data["profile"].(map[string]interface{})
	if !ok { /* 错误处理 */ }

	// 3. 访问 "settings" 字段（类型断言为 []interface{}，即数组）
	settings, ok := profile["settings"].([]interface{})
	if !ok || len(settings) == 0 { /* 错误处理 */ }

	// 4. 访问数组的第一个元素 [0]
	firstSetting, ok := settings[0].(map[string]interface{})
	if !ok { /* 错误处理 */ }

	// 5. 访问 "theme" 字段
	theme, ok := firstSetting["theme"].(string)
	if !ok { /* 错误处理 */ }

	fmt.Println("--- 标准库访问 ---")
	fmt.Printf("获取到的主题 (theme): %s\n", theme) 
	// 预期输出: dark
}