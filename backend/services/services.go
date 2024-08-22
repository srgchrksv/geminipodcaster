package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"github.com/gorilla/websocket"
	"github.com/srgchrksv/geminipodcaster/models"
)

type PodcastSession struct {
	InteractionPrompt chan []byte
	PodcastContext    string
	ChatSession       *genai.ChatSession
}

var (
	mu sync.Mutex
)

type Services struct {
	Voices             []string
	PodcastSessions    map[string]*PodcastSession
	ClientTextToSpeech *texttospeech.Client
}

func NewServices() *Services {
	return &Services{
		PodcastSessions: make(map[string]*PodcastSession),
	}
}

func (s *Services) TextToSpeech(ctx context.Context, text, voice string) ([]byte, error) {
	// Perform the text-to-speech request on the text input with the selected
	// voice parameters and audio file type.
	req := texttospeechpb.SynthesizeSpeechRequest{
		// Set the text input to be synthesized.
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: text},
		},
		// Build the voice request, select the language code ("en-US") and the SSML
		// voice gender ("neutral").
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: "en-US",
			SsmlGender:   texttospeechpb.SsmlVoiceGender_NEUTRAL,
			Name:         voice,
		},
		// Select the type of audio file you want returned.
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_MP3,
		},
	}

	resp, err := s.ClientTextToSpeech.SynthesizeSpeech(ctx, &req)
	if err != nil {
		return nil, err
	}
	return resp.AudioContent, nil
}

func (s Services) Podcast(c *gin.Context, conn *websocket.Conn, model models.Gemini, sessionId string, clientTextToSpeech *texttospeech.Client) {
	// Read binary data from WebSocket message
	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Println("Read error:", err)
		return
	}
	podcastContext := string(message)
	log.Println("Podcast context", podcastContext)

	// Create a new chat chatSession
	chatSession := model.StartChat()
	chatSession.History = []*genai.Content{}

	// Create a new channel for user interaction prompt store it in users session

	interactionPrompt := make(chan []byte)
	mu.Lock()
	s.PodcastSessions[sessionId] = &PodcastSession{
		InteractionPrompt: interactionPrompt,
		PodcastContext:    podcastContext,
		ChatSession:       chatSession,
	}
	mu.Unlock()

	// channel for stopping the podcast
	StopChan := make(chan interface{})

	// set random voices for the podcast
	setVoices := make(map[string]string)
	randomInt := rand.Intn(len(s.Voices))
	setVoices["Host"] = s.Voices[randomInt]
	randomDecrement := randomInt - 1
	if randomDecrement < 0 {
		setVoices["Guest"] = s.Voices[randomDecrement+2]
	} else {
		setVoices["Guest"] = s.Voices[randomDecrement]
	}

	// Start the podcast
	go func() {
	P:
		for {
			select {
			default:
				podcastTranscript, err := model.MockGemini(c, chatSession)
				// podcastTranscript, err := model.SendMessages(c, s.PodcastSessions[sessionId].ChatSession, podcastContext)
				if err != nil {
					log.Fatal(err)
				}
				err = s.Podcasting(c, conn, sessionId, podcastTranscript, chatSession, s.PodcastSessions[sessionId].PodcastContext, clientTextToSpeech, setVoices, StopChan)
				if err != nil {
					return
				}
				time.Sleep(2 * time.Second)
			case <-StopChan:
				fmt.Println("Podcast finished successfully")
				conn.Close()
				break P
			}
		}
	}()
}

func (s *Services) Podcasting(c *gin.Context, conn *websocket.Conn, sessionID string, podcastTranscript []models.Segment, session *genai.ChatSession, podcastContext string, clientTextToSpeech *texttospeech.Client, setVoices map[string]string, StopChan chan interface{}) error {
	// history of the conversation that has been done so far, in case of users interaction we wont send all messages as the context again
	var history []*genai.Content
	history = append(history, &genai.Content{Role: "User", Parts: []genai.Part{genai.Text(podcastContext)}})

	//iterate over the podcast transcript
	for i, segment := range podcastTranscript {
		select {
		default:
			fmt.Printf("%s: %s\n", segment.Speaker, segment.Text)

			// write podcast message to the websocket
			err := conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%s: %s\n", segment.Speaker, segment.Text)))
			if err != nil {
				fmt.Println("Error writing ws message:", err)
				break
			}

			audio, err := s.TextToSpeech(c, segment.Text, setVoices[segment.Speaker])
			if err != nil {
				return err
			}
			err = conn.WriteMessage(websocket.BinaryMessage, audio)
			if err != nil {
				fmt.Println("Error writing WS binary message:", err)
				return err
			}

			time.Sleep(3 * time.Second) // Simulate time delay between segments
		case userPrompt := <-s.PodcastSessions[sessionID].InteractionPrompt:
			// on user interaction we regenerate the podcast based on the user interaction
			fmt.Printf("\n\n%v\n\n", string(userPrompt))

			// write users intercation to the websocket
			err := conn.WriteMessage(websocket.TextMessage, userPrompt)
			if err != nil {
				fmt.Println("Error writing ws message:", err)
				break
			}

			mu.Lock()
			s.PodcastSessions[sessionID] = &PodcastSession{InteractionPrompt: make(chan []byte)}
			mu.Unlock()

			podcastData, err := json.Marshal(models.Podcast{Podcast: podcastTranscript[:i+1]})
			if err != nil {
				log.Fatal(err)
			}
			userData, err := json.Marshal(models.Segment{Speaker: "User", Text: string(userPrompt)})
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

func (s Services) UserInteraction(c *gin.Context, sessionID string) error {

	// Process form data
	formData, err := c.MultipartForm()
	if err != nil {
		return fmt.Errorf("error invalid form data: %v", err)
	}

	// Set the session ID
	mu.Lock()
	podcastSession, exists := s.PodcastSessions[sessionID]
	if !exists {
		return fmt.Errorf("session is not in podcastSessions: %v", err)
	}
	// podcastSession.InteractionPrompt <- []byte(fmt.Sprintf("USERS INTERACTION: %v\n", req.UsersInteraction))
	userInteraction := formData.Value["user_interaction"][0]
	podcastSession.InteractionPrompt <- []byte(fmt.Sprintf("USERS INTERACTION: %v\n", userInteraction))
	mu.Unlock()

	return nil
}
