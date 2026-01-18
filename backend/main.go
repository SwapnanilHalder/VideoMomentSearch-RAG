package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	DBFile         = "../db.json"
	OllamaEmbedURL = "http://localhost:11434/api/embeddings"
	OllamaChatURL  = "http://localhost:11434/api/generate"
	EmbedModel     = "bge-m3"
	GenModel       = "llama3.2"
	TopK           = 3
)


type SearchRequest struct {
	Query string `json:"query" binding:"required"`
}

type VideoSegment struct {
	VideoID    string    `json:"video_id"`
	VideoTitle string    `json:"video_title"`
	StartTime  float64   `json:"start_time"`
	EndTime    float64   `json:"end_time"`
	Text       string    `json:"text"`
	Embedding  []float64 `json:"embedding"`
}

type SearchResult struct {
	VideoID      string  `json:"video_id"`
	VideoTitle   string  `json:"video_title"`
	StartSeconds int     `json:"start_seconds"`
	Timestamp    string  `json:"timestamp"`
	Explanation  string  `json:"explanation"`
	Snippet      string  `json:"snippet"`
	Score        float64 `json:"score"`
}

type ScoredSegment struct {
	Segment *VideoSegment
	Score   float64
}

var database []VideoSegment


func main() {
	if err := loadDatabase(); err != nil {
		fmt.Printf("❌ Critical Error: %v\n", err)
		return
	}
	fmt.Printf("✅ Loaded %d video segments into memory.\n", len(database))

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.POST("/search", handleSearch)

	fmt.Println("🚀 Server running on http://localhost:8080")
	r.Run(":8080")
}


func handleSearch(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	fmt.Printf("🔍 Searching for: %s\n", req.Query)

	queryVec, err := getEmbedding(req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ollama embedding failed"})
		return
	}

	topMatches := findTopMatches(queryVec, TopK)
	if len(topMatches) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No matches found"})
		return
	}

	var results []SearchResult

	for _, match := range topMatches {
		seg := match.Segment

		explanation, err := generateWhy(req.Query, seg.Text)
		if err != nil {
			explanation = "Could not generate explanation."
		}

		adjustedStart := int(seg.StartTime) - 20
		if adjustedStart < 0 {
			adjustedStart = 0
		}

		cleanVideoID := strings.TrimSpace(seg.VideoID)

		results = append(results, SearchResult{
			VideoID:      cleanVideoID,
			VideoTitle:   seg.VideoTitle,
			StartSeconds: adjustedStart,
			Timestamp:    fmt.Sprintf("%ds", adjustedStart),
			Explanation:  explanation,
			Snippet:      seg.Text,
			Score:        match.Score,
		})
	}

	c.JSON(http.StatusOK, results)
}


func loadDatabase() error {
	file, err := os.Open(DBFile)
	if err != nil {
		return fmt.Errorf("could not open db.json: %w", err)
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(&database)
}

func findTopMatches(queryVec []float64, k int) []ScoredSegment {
	var scored []ScoredSegment

	for i := range database {
		score := calculateCosine(queryVec, database[i].Embedding)
		scored = append(scored, ScoredSegment{
			Segment: &database[i],
			Score:   score,
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	if len(scored) < k {
		return scored
	}
	return scored[:k]
}

func calculateCosine(vec1, vec2 []float64) float64 {
	var dot, mag1, mag2 float64
	for i := 0; i < len(vec1); i++ {
		dot += vec1[i] * vec2[i]
		mag1 += vec1[i] * vec1[i]
		mag2 += vec2[i] * vec2[i]
	}
	if mag1 == 0 || mag2 == 0 {
		return 0
	}
	return dot / (math.Sqrt(mag1) * math.Sqrt(mag2))
}

func getEmbedding(text string) ([]float64, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":  EmbedModel,
		"prompt": text,
	})
	resp, err := http.Post(OllamaEmbedURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil { return nil, err }
	defer resp.Body.Close()
	
	var result struct { Embedding []float64 `json:"embedding"` }
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil { return nil, err }
	return result.Embedding, nil
}

func generateWhy(userQuery, content string) (string, error) {
	prompt := fmt.Sprintf("Query: %s\nContext: %s\nExplain in 1 sentence why this context matches.", userQuery, content)
	reqBody, _ := json.Marshal(map[string]interface{}{
		"model": GenModel, "prompt": prompt, "stream": false,
	})
	respGen, err := http.Post(OllamaChatURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil { return "", err }
	defer respGen.Body.Close()
	
	var result struct { Response string `json:"response"` }
	if err := json.NewDecoder(respGen.Body).Decode(&result); err != nil { return "", err }
	return result.Response, nil
}