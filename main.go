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

	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

type Segment struct {
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
}

type UserPrompt struct {
	Text string `json:"text"`
}

type PodcastSession struct {
	UserPrompt chan []byte
}
type Podcast struct {
	Podcast []Segment `json:"podcast"`
}

var (
	sessions = make(map[string]*PodcastSession)
	mu       sync.Mutex
)

func main() {

	// Load the .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	apiKey := option.WithAPIKey(os.Getenv("GEMINI_API_KEY"))
	ctx := context.Background()

	// Create a new genai client
	client, err := genai.NewClient(ctx, apiKey)
	if err != nil {
		log.Fatalf("Error creating client: %v\n", err)
	}
	defer client.Close()

	// Get the generative model
	model := client.GenerativeModel("gemini-1.5-flash")

	// model configuration
	model.SetTemperature(1)
	model.SetTopK(64)
	model.SetTopP(0.95)
	model.SetMaxOutputTokens(8192)
	model.ResponseMIMEType = "application/json"
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text("Generate a very short, fun and engaging podcast based on the provided context. If you see a context message starting with USER INTERCATION, regenerate podcast based on the USER INTERCATION message, try to fullfill USER INTERCATION. If USER INTERCATION, respond only with new conversation messages. Example speaker names are 'Host' and 'Guest' dont include 'User'. ")},
	}
	// schema for structured response
	model.ResponseSchema = &genai.Schema{
		Type:        genai.TypeObject,
		Description: "Return the generated podcast in JSON format",
		Properties: map[string]*genai.Schema{
			"podcast": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"speaker": {
							Type:        genai.TypeString,
							Description: "Name of the speaker",
						},
						"text": {
							Type:        genai.TypeString,
							Description: "Text spoken by the speaker",
						},
					},
				},
			},
		},
	}

	// Create a new gin router
	r := gin.Default()
	// Register the route with a closure to pass the api_key
	r.GET("/start", func(c *gin.Context) {
		startPodcast(c, model)
	})
	r.GET("/interact", podcastInteraction)
	// Start the server
	r.Run(":8080")
}

func startPodcast(c *gin.Context, model *genai.GenerativeModel) {
	// var req struct {
	// 	Content []Segment `json:"content"`
	// }
	// if err := c.ShouldBindJSON(&req); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 	return
	// }
	// Generate a unique session ID
	// sessionID := uuid.New().String()

	// thats the base context for the podcast will be coming from the client
	podcastContext := "Golang programming language was created by Google. Its a statically typed, compiled language."

	// Create a new chat session
	session := model.StartChat()
	session.History = []*genai.Content{}

	// Create a new channel for user interaction prompt store it in users session
	text := make(chan []byte)
	mu.Lock()
	sessions["1"] = &PodcastSession{UserPrompt: text}
	mu.Unlock()

	// channel for stopping the podcast
	StopChan := make(chan interface{})

	// Start the podcast
	go func() {
	P:
		for {
			select {
			default:
				podcastTranscript, err := gemini(c, session, podcastContext)
				if err != nil {
					log.Fatal(err)
				}
				err = podcasting("1", podcastTranscript, session, podcastContext, StopChan)
				if err != nil {
					return
				}
				time.Sleep(2 * time.Second)
			case <-StopChan:
				fmt.Println("Podcast finished successfully")
				break P
			}
		}
	}()
	fmt.Println("Podcast started")
	c.JSON(http.StatusOK, gin.H{"message": "Podcast started", "session_id": "1"})
}

func podcasting(sessionID string, podcastTranscript []Segment, session *genai.ChatSession, podcastContext string, StopChan chan interface{}) error {
	// history of the conversation that has been done so far, in case of users interaction we wont send all messages as the context again
	var history []*genai.Content
	history = append(history, &genai.Content{Role: "User", Parts: []genai.Part{genai.Text(podcastContext)}})

	var conversation []Segment

	//iterate over the podcast transcript
	for _, segment := range podcastTranscript {
		conversation = append(conversation, Segment{Speaker: segment.Speaker, Text: segment.Text})
		select {
		default:
			fmt.Printf("%s: %s\n", segment.Speaker, segment.Text)
			time.Sleep(3 * time.Second) // Simulate time delay between segments
		case userPrompt := <-sessions[sessionID].UserPrompt:
			// on user interaction we regenerate the podcast based on the user interaction
			fmt.Printf("\n\n%v\n\n", string(userPrompt))
			mu.Lock()
			sessions[sessionID] = &PodcastSession{UserPrompt: make(chan []byte)}
			mu.Unlock()

			podcastData, err := json.Marshal(Podcast{Podcast: conversation})
			if err != nil {
				log.Fatal(err)
			}
			userData, err := json.Marshal(Segment{Speaker: "User", Text: string(userPrompt)})
			if err != nil {
				log.Fatal(err)
			}
			newContent := &genai.Content{
				Role:  "Model",
				Parts: []genai.Part{genai.Text(podcastData)},
			}
			history = append(history, newContent)
			history = append(history, &genai.Content{Role: "User", Parts: []genai.Part{genai.Text(string(userData))}})
			session.History = history
			return nil
		}
	}
	close(StopChan)

	return nil
}

// route for users interaction
func podcastInteraction(c *gin.Context) {
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
		session.UserPrompt <- []byte("USERS INTERACTION: Can you tell me something about Kubernetes that was built with golang?")
		c.JSON(http.StatusOK, gin.H{"message": "Users interaction."})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID."})
	}
}

func gemini(ctx context.Context, session *genai.ChatSession, podcastContext string) ([]Segment, error) {
	resp, err := session.SendMessage(ctx, genai.Text(podcastContext))
	if err != nil {
		log.Fatalf("Error sending message: %v\n", err)
	}
	var podcast Podcast
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			if err := json.Unmarshal([]byte(txt), &podcast); err != nil {
				log.Fatal(err)
			}
		}
	}

	return podcast.Podcast, nil
}

func mockGemini(ctx context.Context, session genai.ChatSession) ([]Segment, error) {
	// Example JSON data
	jsonData := `{
	    "podcast": [
	        {"speaker": "Host", "text": "Welcome back to the show! Today we're diving into the world of programming languages, and we have a very special guest who's going to tell us all about Golang."},
	        {"speaker": "Guest", "text": "Thanks for having me! Golang, or Go as it's often called, is a language created by Google. It's known for being super fast and efficient, which makes it great for building things like web servers and other backend systems."},
	        {"speaker": "Host", "text": "That's pretty cool! So, is it easy to learn?"},
	        {"speaker": "Guest", "text": "Go is actually considered one of the easier languages to pick up, especially if you've got some programming experience. It's got a clean, simple syntax and a focus on readability."},
	        {"speaker": "Host", "text": "That's really encouraging! Thanks so much for sharing all this about Golang. It sounds like a really powerful language that's worth checking out."},
	        {"speaker": "Guest", "text": "Absolutely! I'd encourage anyone who's interested in programming to give it a try."}
	    ]
	}`

	// Unmarshal JSON data into Go struct
	var podcast Podcast
	err := json.Unmarshal([]byte(jsonData), &podcast)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return []Segment{}, err
	}

	return podcast.Podcast, nil
}
