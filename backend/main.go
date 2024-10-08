package main

import (
	"context"
	"log"
	"os"

	speech "cloud.google.com/go/speech/apiv1"
	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"github.com/srgchrksv/geminipodcaster/routes"
	"github.com/srgchrksv/geminipodcaster/services"
	"github.com/srgchrksv/geminipodcaster/storage"
	"google.golang.org/api/option"
)

func main() {

	// Load the .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	apiKey := option.WithAPIKey(os.Getenv("GEMINI_API_KEY"))
	ctx := context.Background()

	// Creates a client.
	clientSpeechToText, err := speech.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer clientSpeechToText.Close()

	clientTextToSpeech, err := texttospeech.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer clientTextToSpeech.Close()

	// Create a new genai client
	client, err := genai.NewClient(ctx, apiKey)
	if err != nil {
		log.Fatalf("Error creating client: %v\n", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash")
	// model configuration
	model.SetTemperature(1)
	model.SetTopK(64)
	model.SetTopP(0.95)
	model.SetMaxOutputTokens(8192)
	model.ResponseMIMEType = "application/json"
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text("Generate a very short, fun and engaging podcast based on the provided context. If podcast context contains message starting with 'USER INTERCATION:' the podcast is in progress and you have to generate podcast continuation in response to the 'USER INTERCATION:', but in new response never include old conversation messages. Example speaker names are 'Host' and 'Guest', dont include 'User'.")},
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

	// Create a new storage instance
	storage := storage.NewStorage()

	// Create a new services instance
	services := services.NewServices(model, storage)
	services.ClientTextToSpeech = clientTextToSpeech
	services.ClientSpeechToText = clientSpeechToText
	services.Voices = []string{"en-AU-Standard-B", "en-AU-Standard-C", "en-IN-Standard-A", "en-IN-Standard-B", "en-GB-Standard-A", "en-GB-Standard-B", "en-US-Standard-A", "en-US-Standard-C"}

	// Create a new gin router
	r := gin.Default()
	routes.RegisterRoutes(r, services)
	r.Run(":8000")
}
