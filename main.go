package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	pb "cloud.google.com/go/ai/generativelanguage/apiv1beta/generativelanguagepb"
	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

type Segment struct {
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
}

var podcastTranscript = []Segment{
	{Speaker: "Host", Text: "1"},
	{Speaker: "Guest", Text: "1"},
	{Speaker: "Host", Text: "2"},
	{Speaker: "Guest", Text: "2"},
	{Speaker: "Host", Text: "3"},
	{Speaker: "Guest", Text: "3"},
}

var updatePodcast = []Segment{
	{Speaker: "Host", Text: "NEW1"},
	{Speaker: "Guest", Text: "NEW1"},
	{Speaker: "Host", Text: "NEW2"},
	{Speaker: "Host", Text: "NEW2"},
	{Speaker: "Guest", Text: "NEW3"},
	{Speaker: "Host", Text: "NEW3"},
}

type PodcastSession struct {
	Ctx    context.Context
	Cancel context.CancelFunc
}

var (
	sessions = make(map[string]*PodcastSession)
	mu       sync.Mutex
)

func podcast(podcastTranscript []Segment, updatePodcast []Segment, sessionID string, c *gin.Context) string {
	user := Segment{}

	for {
		podcasting(sessionID, &user, podcastTranscript)
		if user.Speaker != "" {
			user = Segment{}
			podcastTranscript = updatePodcast
			time.Sleep(2 * time.Second)
		} else {
			break
		}
	}
	return "Podcast finished successfully."
}

func startPodcast(c *gin.Context) {
	// var req struct {
	// 	Content []Segment `json:"content"`
	// }
	// if err := c.ShouldBindJSON(&req); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 	return
	// }

	// Generate a unique session ID
	// sessionID := uuid.New().String()

	// Initialize the context and cancel function
	ctx, cancel := context.WithCancel(context.Background())

	// Store the session in the map
	mu.Lock()
	sessions["1"] = &PodcastSession{Ctx: ctx, Cancel: cancel}
	mu.Unlock()

	// Run the podcast function in a separate goroutine
	go func() {
		result := podcast(podcastTranscript, updatePodcast, "1", c)
		fmt.Println(result)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "Podcast started", "session_id": "1"})
}

func podcasting(sessionID string, user *Segment, podcastTranscript []Segment) {
	for i, segment := range podcastTranscript {
		select {
		default:
			fmt.Printf("%s: %s\n", segment.Speaker, segment.Text)
			time.Sleep(3 * time.Second) // Simulate time delay between segments
		case <-sessions[sessionID].Ctx.Done():
			fmt.Println("Podcast interrupted.")
			podcastTranscript = podcastTranscript[:i]
			*user = Segment{Speaker: "USER", Text: "YO"}
			fmt.Printf("%v\n\n, %v", user, podcastTranscript)
			fmt.Println("Podcast interrupted.")
			mu.Lock()
			ctx, cancel := context.WithCancel(context.Background())
			sessions[sessionID] = &PodcastSession{Ctx: ctx, Cancel: cancel}
			mu.Unlock()
			return
		}
	}
}

func stopPodcast(c *gin.Context) {
	// var req struct {
	// 	SessionID string `json:"session_id"`
	// }
	// if err := c.ShouldBindJSON(&req); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 	return
	// }

	mu.Lock()
	// session, exists := sessions[req.SessionID]
	session, exists := sessions["1"]
	mu.Unlock()

	if exists {
		session.Cancel()
		c.JSON(http.StatusOK, gin.H{"message": "Podcast stopped successfully."})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID."})
	}
}

type Part interface {
	toPart() *pb.Part
}

type Podcast struct {
	Podcast []Segment `json:"podcast"`
}

func (p *Podcast) toPart() *pb.Part {
	// Implement the conversion to pb.Part if needed
	return &pb.Part{}
}

func main() {
	// // Example JSON data
	// jsonData := `{
	//     "podcast": [
	//         {"speaker": "Host", "text": "Welcome back to the show! Today we're diving into the world of programming languages, and we have a very special guest who's going to tell us all about Golang."},
	//         {"speaker": "Guest", "text": "Thanks for having me! Golang, or Go as it's often called, is a language created by Google. It's known for being super fast and efficient, which makes it great for building things like web servers and other backend systems."},
	//         {"speaker": "Host", "text": "That's pretty cool! So, is it easy to learn?"},
	//         {"speaker": "Guest", "text": "Go is actually considered one of the easier languages to pick up, especially if you've got some programming experience. It's got a clean, simple syntax and a focus on readability."},
	//         {"speaker": "Host", "text": "That's really encouraging! Thanks so much for sharing all this about Golang. It sounds like a really powerful language that's worth checking out."},
	//         {"speaker": "Guest", "text": "Absolutely! I'd encourage anyone who's interested in programming to give it a try."}
	//     ]
	// }`

	// // Unmarshal JSON data into Go struct
	// var podcast Podcast
	// err := json.Unmarshal([]byte(jsonData), &podcast)
	// if err != nil {
	// 	fmt.Println("Error unmarshaling JSON:", err)
	// 	return
	// }

	// // Print the podcast data
	// for _, segment := range podcast.Podcast {
	// 	fmt.Printf("Speaker: %s\nText: %s\n\n", segment.Speaker, segment.Text)
	// }

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ctx := context.Background()

	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		log.Fatalf("Error creating client: %v\n", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash")

	model.SetTemperature(1)
	model.SetTopK(64)
	model.SetTopP(0.95)
	model.SetMaxOutputTokens(8192)
	model.ResponseMIMEType = "application/json"
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text("Generate a very short, fun and engaging podcast based on the provided content. Example speaker names are 'Host' and 'Guest'.")},
	}
	model.ResponseSchema = &genai.Schema{
		Type:        genai.TypeObject,
		Description: "Return the generated podcast in JSON format",
		Properties: map[string]*genai.Schema{
			"podcast": &genai.Schema{
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"speaker": &genai.Schema{
							Type:        genai.TypeString,
							Description: "Name of the speaker",
						},
						"text": &genai.Schema{
							Type:        genai.TypeString,
							Description: "Text spoken by the speaker",
						},
					},
				},
			},
		},
	}

	session := model.StartChat()
	session.History = []*genai.Content{}

	resp, err := session.SendMessage(ctx, genai.Text("Golang programming language was created by Google. Its a statically typed, compiled language."))
	if err != nil {
		log.Fatalf("Error sending message: %v\n", err)
	}
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			var podcasts Podcast
			if err := json.Unmarshal([]byte(txt), &podcasts); err != nil {
				log.Fatal(err)
			}
			// that's the podcast
			fmt.Println(podcasts.Podcast)
			for _, segment := range podcasts.Podcast {
				fmt.Printf("Speaker: %s\nText: %s\n\n", segment.Speaker, segment.Text)
			}
		}
	}
	for _, elem := range session.History {
		fmt.Printf("%v\n", elem)
	}
	// r := gin.Default()``
	// r.GET("/start", startPodcast)
	// r.GET("/stop", stopPodcast)
	// r.Run(":8080")
}
