package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type RequestBody struct {
	AppKey    string `json:"appkey"`
	Gt        string `json:"gt"`
	Challenge string `json:"challenge,omitempty"` // 设置omitempty以便在空值时不发送该字段
	ItemID    string `json:"itemid"`
	Referer   string `json:"referer,omitempty"`   // 设置omitempty以便在空值时不发送该字段
}

type RecognizeResponse struct {
	ResultID string `json:"resultid"`
}

type ResultResponse struct {
	Status  int                    `json:"status"`
	Msg     string                 `json:"msg"`
	Data    map[string]interface{} `json:"data"`
	Time    int                    `json:"time"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody RequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 检查是否包含所有必要的字段
	if reqBody.AppKey == "" || reqBody.Gt == "" || reqBody.ItemID == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// 如果referer不为空，则添加到请求体中
	if reqBody.Referer != "" {
		reqBody.Referer = reqBody.Referer
	}

	// 如果challenge不为空，则添加到请求体中
	if reqBody.Challenge != "" {
		reqBody.Challenge = reqBody.Challenge
	}

	// 向 http://api.ttocr.com/api/recognize 发起POST请求
	recognizeReqBody, err := json.Marshal(reqBody)
	if err != nil {
		http.Error(w, "Error creating request", http.StatusInternalServerError)
		return
	}

	resp, err := http.Post("http://api.ttocr.com/api/recognize", "application/json", bytes.NewBuffer(recognizeReqBody))
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(w, "Error calling recognize API", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var recognizeResp RecognizeResponse
	err = json.NewDecoder(resp.Body).Decode(&recognizeResp)
	if err != nil {
		http.Error(w, "Error decoding recognize response", http.StatusInternalServerError)
		return
	}

	// 使用resultid和appkey发起第二个请求
	resultReqBody := map[string]string{
		"resultid": recognizeResp.ResultID,
		"appkey":   reqBody.AppKey,
	}

	var resultResp ResultResponse
	startTime := time.Now()
	for {
		resultReqBodyJson, _ := json.Marshal(resultReqBody)
		resp, err := http.Post("http://api.ttocr.com/api/results", "application/json", bytes.NewBuffer(resultReqBodyJson))
		if err != nil || resp.StatusCode != http.StatusOK {
			http.Error(w, "Error calling results API", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		err = json.NewDecoder(resp.Body).Decode(&resultResp)
		if err != nil {
			http.Error(w, "Error decoding results response", http.StatusInternalServerError)
			return
		}

		if resultResp.Status == 1 || time.Since(startTime).Seconds() > 60 {
			break
		}

		time.Sleep(2 * time.Second)
	}

	responseJson, err := json.Marshal(resultResp)
	if err != nil {
		http.Error(w, "Error creating response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}

func main() {
	http.HandleFunc("/", handler)
	fmt.Println("Server is running on port 23333...")
	log.Fatal(http.ListenAndServe(":23333", nil))
}
