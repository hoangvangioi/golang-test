package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"vocabulary/config"
	"vocabulary/database"
	"vocabulary/models"

	"github.com/kataras/iris/v12"
)

// GroqRequest struct for the API call
type GroqRequest struct {
	Model          string      `json:"model"`
	Messages       []Message   `json:"messages"`
	ResponseFormat interface{} `json:"response_format,omitempty"`
}

// Message struct for Groq API messages
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GroqResponse struct for the API response
type GroqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// APIResponse is a generic response structure for the REST API
type APIResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// IndexHandler returns a simple welcome message
func IndexHandler(ctx iris.Context) {
	ctx.JSON(APIResponse{
		Status: "success",
		Data:   "Welcome to the Dialog Processing API",
	})
}

// GenerateDialogHandler generates a Vietnamese dialog
func GenerateDialogHandler(ctx iris.Context) {
	cfg, err := config.LoadConfig()
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(APIResponse{Status: "error", Error: fmt.Sprintf("Failed to load config: %v", err)})
		return
	}

	dialogPrompt := `Tạo một hội thoại bằng tiếng Việt, gồm 6 câu, ngắn gọn, đơn giản, hỏi đường đi đến hồ Hoàn Kiếm ở Hà Nội giữa một người Mỹ tên James và người Việt Nam tên Lan. Chỉ xuất ra hội thoại không cần giải thích.`
	dialogRaw, err := callGroqAPI(cfg.GroqAPIKey, dialogPrompt, nil)
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(APIResponse{Status: "error", Error: fmt.Sprintf("Failed to generate dialog: %v", err)})
		return
	}

	dialog := extractDialog(dialogRaw)
	if dialog == "" {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(APIResponse{Status: "error", Error: fmt.Sprintf("No valid dialog found in response: %s", dialogRaw)})
		return
	}

	// Save to database
	dialogModel := models.Dialog{Lang: "vi", Content: dialog}
	dialogID, err := saveDialogToDB(dialogModel)
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(APIResponse{Status: "error", Error: fmt.Sprintf("Failed to save dialog to DB: %v", err)})
		return
	}

	ctx.JSON(APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"dialog":   dialog,
			"dialogID": dialogID,
		},
	})
}

// ExtractWordsHandler extracts important words from a dialog
func ExtractWordsHandler(ctx iris.Context) {
	cfg, err := config.LoadConfig()
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(APIResponse{Status: "error", Error: fmt.Sprintf("Failed to load config: %v", err)})
		return
	}

	dialog := ctx.URLParam("dialog")
	if dialog == "" {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(APIResponse{Status: "error", Error: "Missing 'dialog' parameter"})
		return
	}

	wordsPrompt := fmt.Sprintf(`Từ hội thoại sau, hãy lọc ra danh sách các từ và cụm từ quan trọng, bỏ qua danh từ tên riêng (như James, Lan, Hà Nội, Hoàn Kiếm). Trả về kết quả dưới dạng JSON với cấu trúc {"words": ["word1", "word2", ...]}.
%s`, dialog)
	wordsRaw, err := callGroqAPI(cfg.GroqAPIKey, wordsPrompt, map[string]string{"type": "json_object"})
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(APIResponse{Status: "error", Error: fmt.Sprintf("Failed to extract words: %v", err)})
		return
	}

	var wordsData struct {
		Words []string `json:"words"`
	}
	if err := json.Unmarshal([]byte(wordsRaw), &wordsData); err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(APIResponse{Status: "error", Error: fmt.Sprintf("Failed to unmarshal words JSON: %v (raw data: %s)", err, wordsRaw)})
		return
	}

	ctx.JSON(APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"extractedWords": wordsData.Words,
		},
	})
}

// TranslateWordsHandler translates words from Vietnamese to English
func TranslateWordsHandler(ctx iris.Context) {
	cfg, err := config.LoadConfig()
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(APIResponse{Status: "error", Error: fmt.Sprintf("Failed to load config: %v", err)})
		return
	}

	var request struct {
		Words []string `json:"words"`
	}
	if err := ctx.ReadJSON(&request); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(APIResponse{Status: "error", Error: fmt.Sprintf("Invalid JSON body: %v", err)})
		return
	}

	if len(request.Words) == 0 {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(APIResponse{Status: "error", Error: "No words provided for translation"})
		return
	}

	var wordsToTranslate []string
	for _, word := range request.Words {
		wordsToTranslate = append(wordsToTranslate, fmt.Sprintf(`{"vi": "%s"}`, word))
	}

	translatePrompt := fmt.Sprintf(`Dịch từng từ hoặc cụm từ trong danh sách dưới sang tiếng Anh, trả về JSON với cấu trúc {"translated_words": [{"vi": "word", "en": "translation"}, ...]}.
[%s]`, strings.Join(wordsToTranslate, ","))
	translatedRaw, err := callGroqAPI(cfg.GroqAPIKey, translatePrompt, map[string]string{"type": "json_object"})
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(APIResponse{Status: "error", Error: fmt.Sprintf("Failed to translate words: %v", err)})
		return
	}

	var translatedData struct {
		TranslatedWords []map[string]string `json:"translated_words"`
	}
	if err := json.Unmarshal([]byte(translatedRaw), &translatedData); err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(APIResponse{Status: "error", Error: fmt.Sprintf("Failed to unmarshal translated JSON: %v (raw data: %s)", err, translatedRaw)})
		return
	}

	ctx.JSON(APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"translatedWords": translatedData.TranslatedWords,
		},
	})
}

// SaveWordsHandler saves words and their translations to the database with dialog relation
func SaveWordsHandler(ctx iris.Context) {
	var request struct {
		DialogID        int64               `json:"dialogID"`
		TranslatedWords []map[string]string `json:"translatedWords"`
	}
	if err := ctx.ReadJSON(&request); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(APIResponse{Status: "error", Error: fmt.Sprintf("Invalid JSON body: %v", err)})
		return
	}

	if request.DialogID == 0 {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(APIResponse{Status: "error", Error: "Missing 'dialogID'"})
		return
	}

	if len(request.TranslatedWords) == 0 {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(APIResponse{Status: "error", Error: "No translated words provided"})
		return
	}

	var savedWords []map[string]interface{}
	for _, translatedWord := range request.TranslatedWords {
		viWord := translatedWord["vi"]
		enWord := translatedWord["en"]

		wordModel := models.Word{Lang: "vi", Content: viWord, Translate: enWord}
		wordID, err := saveWordToDB(wordModel)
		if err != nil {
			log.Printf("Failed to save word '%s' to DB: %v", viWord, err)
			continue
		}

		if err := createWordDialogRelation(request.DialogID, wordID); err != nil {
			log.Printf("Failed to create relation between dialog %d and word %d: %v", request.DialogID, wordID, err)
			continue
		}

		savedWords = append(savedWords, map[string]interface{}{
			"vi":     viWord,
			"en":     enWord,
			"wordID": wordID,
		})
	}

	ctx.JSON(APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"savedWords": savedWords,
			"dialogID":   request.DialogID,
		},
	})
}

// Helper functions remain unchanged
func callGroqAPI(apiKey, prompt string, responseFormat interface{}) (string, error) {
	client := &http.Client{}
	reqBody := GroqRequest{
		Model:          "deepseek-r1-distill-llama-70b",
		Messages:       []Message{{Role: "user", Content: prompt}},
		ResponseFormat: responseFormat,
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to Groq API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Groq API returned an error: %s - %s", resp.Status, string(bodyBytes))
	}

	var respBody GroqResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", fmt.Errorf("failed to decode Groq API response: %w", err)
	}

	if len(respBody.Choices) == 0 || respBody.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("no valid content in Groq API response")
	}

	return respBody.Choices[0].Message.Content, nil
}

func extractDialog(raw string) string {
	thinkEnd := strings.Index(raw, "</think>")
	if thinkEnd != -1 {
		return strings.TrimSpace(raw[thinkEnd+len("</think>"):])
	}
	return strings.TrimSpace(raw)
}

func saveDialogToDB(dialog models.Dialog) (int64, error) {
	var id int64
	err := database.DB.QueryRow("INSERT INTO dialog (lang, content) VALUES ($1, $2) RETURNING id", dialog.Lang, dialog.Content).Scan(&id)
	return id, err
}

func saveWordToDB(word models.Word) (int64, error) {
	var id int64
	err := database.DB.QueryRow("SELECT id FROM word WHERE content = $1 AND lang = $2", word.Content, word.Lang).Scan(&id)
	if err == nil {
		return id, nil
	}
	err = database.DB.QueryRow("INSERT INTO word (lang, content, translate) VALUES ($1, $2, $3) RETURNING id", word.Lang, word.Content, word.Translate).Scan(&id)
	return id, err
}

func createWordDialogRelation(dialogID, wordID int64) error {
	_, err := database.DB.Exec("INSERT INTO word_dialog (dialog_id, word_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", dialogID, wordID)
	return err
}
