package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

//go:embed frontend/dist/*
var staticFiles embed.FS

type PlaylistRequest struct {
	YoutubeURL string `json:"youtube_url"`
}

type Video struct {
	Id          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Thumbnail   string `json:"thumbnail"`
	Duration    string `json:"duration"`
	DurationSec int    `json:"duration_sec"`
}

type Playlist struct {
	Id               string  `json:"id"`
	Title            string  `json:"title"`
	Description      string  `json:"description"`
	Thumbnail        string  `json:"thumbnail"`
	Videos           []Video `json:"videos"`
	TotalDurationSec int     `json:"total_duration_sec"`
}

func getPlaylistId(youtubeURL string) string {
	listURL, err := url.Parse(youtubeURL)
	if err != nil {
		return ""
	}
	return listURL.Query().Get("list")
}

func getDurationSec(isoDuration string) int {
	re := regexp.MustCompile(`PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?`)
	matches := re.FindStringSubmatch(isoDuration)

	var hours, minutes, seconds int
	var err error

	if matches[1] != "" {
		hours, err = strconv.Atoi(matches[1])
		if err != nil {
			return 0
		}
	}

	if matches[2] != "" {
		minutes, err = strconv.Atoi(matches[2])
		if err != nil {
			return 0
		}
	}

	if matches[3] != "" {
		seconds, err = strconv.Atoi(matches[3])
		if err != nil {
			return 0
		}
	}

	return hours*3600 + minutes*60 + seconds
}

func getPlaylistVideos(playlistId string) ([]Video, int, error) {
	youtubeClient, err := getYoutubeClient(context.Background())
	if err != nil {
		return nil, 0, err
	}

	var allVideoIDs []string
	nextPageToken := ""

	for {
		call := youtubeClient.PlaylistItems.List([]string{"contentDetails"}).
			PlaylistId(playlistId).
			MaxResults(50)

		if nextPageToken != "" {
			call = call.PageToken(nextPageToken)
		}

		response, err := call.Do()
		if err != nil {
			return nil, 0, err
		}

		if len(response.Items) == 0 {
			break
		}

		for _, item := range response.Items {
			allVideoIDs = append(allVideoIDs, item.ContentDetails.VideoId)
		}

		if response.NextPageToken == "" {
			break
		}
		nextPageToken = response.NextPageToken
	}

	if len(allVideoIDs) == 0 {
		return nil, 0, fmt.Errorf("playlist is empty or private")
	}

	var allVideos []Video
	var totalDurationSec int

	for i := 0; i < len(allVideoIDs); i += 50 {
		end := i + 50
		if end > len(allVideoIDs) {
			end = len(allVideoIDs)
		}

		batch := allVideoIDs[i:end]
		videoIDsString := strings.Join(batch, ",")

		videosResponse, err := youtubeClient.Videos.List([]string{"snippet", "contentDetails"}).Id(videoIDsString).Do()
		if err != nil {
			return nil, 0, fmt.Errorf("error getting video details: %v", err)
		}

		for _, item := range videosResponse.Items {
			video := Video{
				Id:          item.Id,
				Title:       item.Snippet.Title,
				Description: item.Snippet.Description,
				Thumbnail:   item.Snippet.Thumbnails.Default.Url,
				Duration:    item.ContentDetails.Duration,
				DurationSec: getDurationSec(item.ContentDetails.Duration),
			}
			allVideos = append(allVideos, video)
			totalDurationSec += video.DurationSec
		}
	}

	return allVideos, totalDurationSec, nil
}

func analyzePlaylist(playlistId string) (*Playlist, error) {
	youtubeClient, err := getYoutubeClient(context.Background())
	if err != nil {
		return nil, err
	}

	response, err := youtubeClient.Playlists.List([]string{"snippet"}).Id(playlistId).Do()
	if err != nil {
		return nil, err
	}

	if len(response.Items) == 0 {
		return nil, fmt.Errorf("playlist not found or private")
	}

	videos, totalDurationSec, err := getPlaylistVideos(playlistId)
	if err != nil {
		return nil, err
	}

	playlistData := response.Items[0]

	playlist := &Playlist{
		Id:               playlistData.Id,
		Title:            playlistData.Snippet.Title,
		Description:      playlistData.Snippet.Description,
		Thumbnail:        playlistData.Snippet.Thumbnails.Default.Url,
		Videos:           videos,
		TotalDurationSec: totalDurationSec,
	}
	return playlist, nil
}

func getYoutubeClient(ctx context.Context) (*youtube.Service, error) {
	youtubeClient, err := youtube.NewService(ctx, option.WithAPIKey(os.Getenv("YOUTUBE_API_KEY")))
	if err != nil {
		return nil, err
	}
	return youtubeClient, nil
}

// Helper function to get MIME type based on file extension
func getMimeType(filename string) string {
	switch {
	case strings.HasSuffix(filename, ".js"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(filename, ".mjs"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(filename, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(filename, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(filename, ".png"):
		return "image/png"
	case strings.HasSuffix(filename, ".jpg"), strings.HasSuffix(filename, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(filename, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(filename, ".ico"):
		return "image/x-icon"
	case strings.HasSuffix(filename, ".woff"):
		return "font/woff"
	case strings.HasSuffix(filename, ".woff2"):
		return "font/woff2"
	case strings.HasSuffix(filename, ".ttf"):
		return "font/ttf"
	case strings.HasSuffix(filename, ".json"):
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

func main() {
	app := fiber.New()
	app.Use(cors.New())
	app.Use(logger.New())

	// API routes FIRST
	app.Post("/api/playlist/analyze", func(c *fiber.Ctx) error {
		var request PlaylistRequest
		if err := c.BodyParser(&request); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request")
		}

		playlistId := getPlaylistId(request.YoutubeURL)
		if playlistId == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid YouTube URL")
		}

		playlist, err := analyzePlaylist(playlistId)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		return c.JSON(playlist)
	})

	// Extract static files
	staticFileSystem, err := fs.Sub(staticFiles, "frontend/dist")
	if err != nil {
		log.Fatal("Failed to load static files:", err)
	}

	// Serve static files with proper MIME types
	app.Get("/*", func(c *fiber.Ctx) error {
		path := c.Path()

		// Skip API routes
		if strings.HasPrefix(path, "/api/") {
			return c.Next()
		}

		// Remove leading slash for file system access
		filePath := strings.TrimPrefix(path, "/")

		// If requesting root, serve index.html
		if filePath == "" {
			filePath = "index.html"
		}

		// Try to read the file first
		content, err := fs.ReadFile(staticFileSystem, filePath)
		if err != nil {
			// If it's an asset file that doesn't exist, return 404
			if strings.Contains(filePath, "/assets/") ||
				strings.HasSuffix(filePath, ".js") ||
				strings.HasSuffix(filePath, ".css") ||
				strings.HasSuffix(filePath, ".png") ||
				strings.HasSuffix(filePath, ".svg") ||
				strings.HasSuffix(filePath, ".ico") {
				return c.Status(404).SendString("File not found")
			}

			// For other routes (SPA routing), serve index.html
			indexContent, indexErr := fs.ReadFile(staticFileSystem, "index.html")
			if indexErr != nil {
				return c.Status(404).SendString("index.html not found")
			}
			c.Set("Content-Type", "text/html; charset=utf-8")
			return c.Send(indexContent)
		}

		// Set proper content type based on file extension
		mimeType := getMimeType(filePath)
		c.Set("Content-Type", mimeType)

		// Set cache headers for static assets
		if strings.HasSuffix(filePath, ".js") ||
			strings.HasSuffix(filePath, ".css") ||
			strings.HasSuffix(filePath, ".png") ||
			strings.HasSuffix(filePath, ".svg") ||
			strings.HasSuffix(filePath, ".ico") {
			c.Set("Cache-Control", "public, max-age=31536000")
		}

		return c.Send(content)
	})

	// Use PORT environment variable for Render compatibility
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🚀 Server running on port %s\n", port)
	fmt.Println("📍 API endpoint: POST /api/playlist/analyze")
	fmt.Println("🌐 Frontend available at: /")

	app.Listen(":" + port)
}
