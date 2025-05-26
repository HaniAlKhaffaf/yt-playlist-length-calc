package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

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

func main() {
	app := fiber.New(fiber.Config{
		EnableTrustedProxyCheck: true,
		TrustedProxies:          []string{"0.0.0.0/0"},
	})

	app.Use(func(c *fiber.Ctx) error {
		origin := c.Get("Origin")

		if origin == "https://calm-souffle-b21063.netlify.app" {
			c.Set("Access-Control-Allow-Origin", origin)
		}

		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
		c.Set("Access-Control-Max-Age", "86400")

		if c.Method() == "OPTIONS" {
			return c.SendStatus(200)
		}

		return c.Next()
	})

	app.Use(logger.New())

	app.Options("/api/playlist/analyze", func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "https://calm-souffle-b21063.netlify.app")
		c.Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Accept, Origin")
		return c.SendStatus(200)
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "YouTube Playlist Length Calculator API",
			"version": "1.0.0",
		})
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "YouTube Playlist Length Calculator API",
			"version": "1.0.0",
			"endpoints": fiber.Map{
				"health":           "GET /health",
				"analyze_playlist": "POST /api/playlist/analyze",
			},
			"usage": fiber.Map{
				"method": "POST",
				"url":    "/api/playlist/analyze",
				"body": fiber.Map{
					"youtube_url": "https://www.youtube.com/playlist?list=PLAYLIST_ID",
				},
			},
		})
	})

	app.Post("/api/playlist/analyze", func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "https://calm-souffle-b21063.netlify.app")

		var youtubeURL string

		if formURL := c.FormValue("youtube_url"); formURL != "" {
			youtubeURL = formURL
		} else {
			var request PlaylistRequest
			if err := c.BodyParser(&request); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error":   "Invalid request format",
					"message": "Please provide youtube_url in form data or JSON",
				})
			}
			youtubeURL = request.YoutubeURL
		}

		if youtubeURL == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Missing youtube_url",
				"message": "Please provide a YouTube playlist URL",
			})
		}

		playlistId := getPlaylistId(youtubeURL)
		if playlistId == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Invalid YouTube URL",
				"message": "Please provide a valid YouTube playlist URL",
			})
		}

		playlist, err := analyzePlaylist(playlistId)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to analyze playlist",
				"message": err.Error(),
			})
		}

		return c.JSON(playlist)
	})

	app.Use(func(c *fiber.Ctx) error {
		return c.Status(404).JSON(fiber.Map{
			"error":   "Route not found",
			"message": "The requested endpoint does not exist",
			"available_endpoints": []string{
				"GET /",
				"GET /health",
				"POST /api/playlist/analyze",
			},
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}
