package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "modernc.org/sqlite"
)

const (
	API_KEY = "sk-b828e1362f614206bde9ddf87b2476a2"
	API_URL = "https://api.deepseek.com/chat/completions"
	DB_FILE = "./chat.db"
)

var db *sql.DB

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Message string `json:"message"`
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite", DB_FILE)
	if err != nil {
		log.Fatal(err)
	}
	sqlStmt := `CREATE TABLE IF NOT EXISTS messages (id INTEGER PRIMARY KEY AUTOINCREMENT, role TEXT NOT NULL, content TEXT NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatal(err)
	}
}

func saveMessage(role, content string) {
	_, err := db.Exec("INSERT INTO messages (role, content) VALUES (?, ?)", role, content)
	if err != nil {
		log.Println("保存失败:", err)
	}
}

func main() {
	initDB()
	defer db.Close()

	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodOptions},
	}))

	e.GET("/api/history", func(c echo.Context) error {
		rows, err := db.Query("SELECT role, content FROM messages ORDER BY id ASC")
		if err != nil {
			return c.JSON(500, map[string]string{"error": "读取失败"})
		}
		defer rows.Close()
		var history []Message
		for rows.Next() {
			var msg Message
			rows.Scan(&msg.Role, &msg.Content)
			history = append(history, msg)
		}
		return c.JSON(200, history)
	})

	e.DELETE("/api/history", func(c echo.Context) error {
		db.Exec("DELETE FROM messages")
		return c.JSON(200, map[string]string{"message": "已清空"})
	})

	e.POST("/api/chat", func(c echo.Context) error {
		req := new(ChatRequest)
		if err := c.Bind(req); err != nil {
			return err
		}
		saveMessage("user", req.Message)

		payload, _ := json.Marshal(map[string]interface{}{
			"model":    "deepseek-chat",
			"messages": []map[string]string{{"role": "user", "content": req.Message}},
			"stream":   true,
		})

		httpReq, _ := http.NewRequest("POST", API_URL, bytes.NewBuffer(payload))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+API_KEY)

		resp, err := (&http.Client{}).Do(httpReq)
		if err != nil {
			return c.String(500, "AI 连接失败")
		}
		defer resp.Body.Close()

		c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
		c.Response().WriteHeader(http.StatusOK)

		var fullContent strings.Builder
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					break
				}
				var streamResp struct {
					Choices []struct {
						Delta struct {
							Content string `json:"content"`
						} `json:"delta"`
					} `json:"choices"`
				}
				if err := json.Unmarshal([]byte(data), &streamResp); err == nil {
					if len(streamResp.Choices) > 0 {
						content := streamResp.Choices[0].Delta.Content
						fullContent.WriteString(content)
						// 直接发送原始字符，不加 data: 前缀，前端处理更简单
						fmt.Fprint(c.Response().Writer, content)
						c.Response().Flush()
					}
				}
			}
		}
		if fullContent.Len() > 0 {
			saveMessage("assistant", fullContent.String())
		}
		return nil
	})

	e.Logger.Fatal(e.Start(":8080"))
}
