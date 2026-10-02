package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const threshold = 0.9

type question struct {
	Type         string `json:"type"`         // "noul" / "choice" / "score"
	Instructions string `json:"instructions"` // Jevに判定させたい内容
}

type requestInput struct {
	State     string              `json:"state"` // 判定対象の発言
	Questions map[string]question `json:"questions"`
}

type requestBody struct {
	Model string       `json:"model"`
	Input requestInput `json:"input"`
}

type answer struct {
	Type string  `json:"type"` // "noul" / "choice" / "score"
	Noul float64 `json:"noul"` // noul型のときの確率値（0〜1）
}

type responseBody struct {
	Result struct {
		Result struct {
			Answers map[string]answer `json:"answers"`
		} `json:"result"`
	} `json:"result"`
}

func judge(text string) (float64, error) {
	url := fmt.Sprintf(
		"https://api.cloudflare.com/client/v4/accounts/%s/ai/run",
		os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
	)

	body := requestBody{
		Model: "typesafe/jev",
		Input: requestInput{
			State: text,
			Questions: map[string]question{
				"evidence": {
					Type:         "noul",
					Instructions: "この発言は、検証可能な根拠（数字、出典、具体的な事実）を示しているか？",
				},
			},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("CLOUDFLARE_API_TOKEN"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("status=%d body=%s", resp.StatusCode, data)
	}

	var parsed responseBody
	if err := json.Unmarshal(data, &parsed); err != nil {
		return 0, err
	}

	return parsed.Result.Result.Answers["evidence"].Noul, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "使い方: go run main.go \"発言\"")
		os.Exit(1)
	}
	text := os.Args[1]

	evidence, err := judge(text)
	if err != nil {
		fmt.Fprintln(os.Stderr, "エラー:", err)
		os.Exit(1)
	}

	fmt.Printf("evidence=%.2f\n", evidence)

	if evidence >= threshold {
		fmt.Println("……")
	} else {
		fmt.Println("それってあなたの感想ですよね？")
	}
}
