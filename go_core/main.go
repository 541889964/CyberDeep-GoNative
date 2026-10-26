package core

import (
	"bufio"
	"crypto/sha256"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

var client tls_client.HttpClient

func init() {
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(60),
		tls_client.WithClientProfile(profiles.Chrome_120),
		tls_client.WithDisableHttp3(),
	}
	var err error
	client, err = tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		fmt.Println("Init client error:", err)
	}
}

type StreamListener interface {
	OnChunk(text string)
	OnFinish()
	OnError(errMsg string)
}

func SendMessage(token string, prompt string, listener StreamListener) {
	go func() {
		reqCh, _ := http.NewRequest("POST", "https://chat.deepseek.com/api/v0/chat/create_pow_challenge", strings.NewReader(`{"target_path":"/api/v0/chat/completion"}`))
		reqCh.Header.Set("Authorization", "Bearer "+token)
		reqCh.Header.Set("Content-Type", "application/json")
		reqCh.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		reqCh.Header[http.HeaderOrderKey] = []string{"authorization", "content-type", "user-agent"}
		
		chRes, err := client.Do(reqCh)
		if err != nil {
			listener.OnError("请求Challenge失败: " + err.Error())
			return
		}
		defer chRes.Body.Close()
		
		var chData map[string]interface{}
		json.NewDecoder(chRes.Body).Decode(&chData)
		
		data, ok := chData["data"].(map[string]interface{})
		if !ok {
			listener.OnError("Token 失效或接口被拦截")
			return
		}
		
		biz := data["biz_data"].(map[string]interface{})
		challengeObj := biz["challenge"].(map[string]interface{})
		
		salt := challengeObj["salt"].(string)
		challenge := challengeObj["challenge"].(string)
		difficulty := int(challengeObj["difficulty"].(float64))
		signature := challengeObj["signature"].(string)
		expireAt := int64(challengeObj["expire_at"].(float64))
		
		base := fmt.Sprintf("%s_%d_", salt, expireAt)
		answer := -1
		for i := 0; i < difficulty; i++ {
			h := sha256.New()
			h.Write([]byte(fmt.Sprintf("%s%d", base, i)))
			hashStr := hex.EncodeToString(h.Sum(nil))
			if hashStr[:4] == challenge[:4] {
				answer = i
				break
			}
		}

		powPayload := map[string]interface{}{
			"algorithm": challengeObj["algorithm"],
			"challenge": challenge,
			"salt":      salt,
			"signature": signature,
			"answer":    answer,
			"target_path": "/api/v0/chat/completion",
		}
		powJson, _ := json.Marshal(powPayload)
		powB64 := base64.StdEncoding.EncodeToString(powJson)

		bodyMap := map[string]interface{}{
			"chat_session_id": "",
			"prompt":          prompt,
			"ref_file_ids":    []string{},
		}
		bodyBytes, _ := json.Marshal(bodyMap)
		
		req, _ := http.NewRequest("POST", "https://chat.deepseek.com/api/v0/chat/completion", strings.NewReader(string(bodyBytes)))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-DS-PoW-Response", powB64)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req.Header[http.HeaderOrderKey] = []string{"authorization", "content-type", "x-ds-pow-response", "user-agent"}

		resp, err := client.Do(req)
		if err != nil {
			listener.OnError("对话请求失败: " + err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			listener.OnError(fmt.Sprintf("对话报错 HTTP %d", resp.StatusCode))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				dataStr := line[6:]
				if dataStr == "[DONE]" { continue }
				var chunk map[string]interface{}
				if err := json.Unmarshal([]byte(dataStr), &chunk); err == nil {
					if v, ok := chunk["v"].(string); ok {
						listener.OnChunk(v)
					}
				}
			}
		}
		listener.OnFinish()
	}()
}
