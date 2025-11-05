package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"gee"
)

// simple in-memory chat history for demo
var (
	messages   []string
	messagesMu sync.RWMutex
)

func main() {
	// Default engine with Logger + Recovery; add CORS for demo UI/API
	r := gee.Default()
	r.Use(gee.CORS())

	// load templates and static assets
	r.SetFuncMap(map[string]interface{}{
		"now": func() string { return time.Now().Format(time.RFC822) },
	})
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "static")

	// Home and guide
	r.GET("/", func(c *gee.Context) {
		c.HTML(http.StatusOK, "guide.tmpl", gee.H{
			"title": "欢迎访问 Dipperick 的 AI 助手 Web 框架",
			"routes": []gee.H{
				{"method": "GET", "path": "/", "desc": "本页面（使用指南）"},
				{"method": "GET", "path": "/panic", "desc": "触发 panic（演示 Recovery）"},
				{"method": "GET", "path": "/ai", "desc": "AI 助手聊天界面"},
				{"method": "POST", "path": "/api/ai/chat", "desc": "聊天 API（JSON）"},
				{"method": "POST", "path": "/api/ai/reset", "desc": "清空聊天记录"},
				{"method": "GET", "path": "/api/v1/ping", "desc": "健康检查 ping"},
				{"method": "GET", "path": "/api/v1/users/:name", "desc": "路径参数示例"},
			},
		})
	})

	// Panic demo
	r.GET("/panic", func(c *gee.Context) {
		names := []string{"geektutu"}
		c.String(http.StatusOK, "%s", names[100])
	})

	// Grouped routes for versioned API
	api := r.Group("/api")
	v1 := api.Group("/v1")
	{
		v1.GET("/ping", func(c *gee.Context) {
			c.JSON(http.StatusOK, gee.H{"message": "pong", "time": time.Now().Unix()})
		})
	}

	v1Users := v1.Group("/users")
	{
		v1Users.GET("/:name", func(c *gee.Context) {
			name := c.Param("name")
			c.JSON(http.StatusOK, gee.H{"hello": name})
		})
	}

	// AI assistant UI
	r.GET("/ai", func(c *gee.Context) {
		messagesMu.RLock()
		snapshot := append([]string(nil), messages...)
		messagesMu.RUnlock()
		c.HTML(http.StatusOK, "ai.tmpl", gee.H{"title": "Dipperick 的 AI 助手", "history": snapshot})
	})

	// AI assistant API (mocked). In real use, call LLM here.
	r.POST("/api/ai/chat", func(c *gee.Context) {
		question := c.PostForm("q")
		if question == "" {
			c.JSON(http.StatusBadRequest, gee.H{"error": "缺少参数 q"})
			return
		}
		// 简单回声：将用户输入反转作为回答（演示用）
		ans := reverse(question)
		messagesMu.Lock()
		messages = append(messages, fmt.Sprintf("你: %s", question))
		messages = append(messages, fmt.Sprintf("助手: %s", ans))
		messagesMu.Unlock()
		c.JSON(http.StatusOK, gee.H{"answer": ans})
	})

	// Reset chat history
	r.POST("/api/ai/reset", func(c *gee.Context) {
		messagesMu.Lock()
		messages = nil
		messagesMu.Unlock()
		c.JSON(http.StatusOK, gee.H{"ok": true})
	})

	log.Fatal(r.Run(":9999"))
}

func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
