package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func main() {
	// First login
	loginReq := map[string]string{"email": "admin@example.com", "password": "password"}
	loginBody, _ := json.Marshal(loginReq)
	resp, err := http.Post("http://localhost:8000/api/v1/auth/login", "application/json", bytes.NewBuffer(loginBody))
	if err != nil {
		fmt.Println("Login error:", err)
		return
	}
	var loginRes map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&loginRes)
	token := loginRes["data"].(map[string]interface{})["access_token"].(string)

	// Now stream
	streamReq := map[string]string{"session_id": "test-session", "message": "tujian dari SOP vendor IT"}
	streamBody, _ := json.Marshal(streamReq)
	req, _ := http.NewRequest("POST", "http://localhost:8000/api/v1/chat/stream", bytes.NewBuffer(streamBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	sResp, err := client.Do(req)
	if err != nil {
		fmt.Println("Stream error:", err)
		return
	}

	reader := bufio.NewReader(sResp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println("Read error:", err)
			break
		}
		if strings.HasPrefix(line, "event: sources") {
			fmt.Print("FOUND SOURCES EVENT: ")
			nextLine, _ := reader.ReadString('\n')
			fmt.Print(nextLine)
		} else if strings.HasPrefix(line, "event:") {
			fmt.Print(line)
		}
	}
}
