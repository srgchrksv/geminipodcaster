package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"github.com/gorilla/websocket"
	"github.com/srgchrksv/geminipodcaster/models"
	"github.com/srgchrksv/geminipodcaster/storage"
)

var (
	mu sync.Mutex
)

type Services struct {
	Voices             []string
	storage            *storage.Storage
	ClientTextToSpeech *texttospeech.Client
	ClientSpeechToText *speech.Client
	model              *genai.GenerativeModel
}

func NewServices(model *genai.GenerativeModel, storage *storage.Storage) *Services {
	return &Services{
		storage: storage,
		model:   model,
	}
}

func (s *Services) SpeechToText(ctx context.Context, audio []byte) (string, error) {
	resp, err := s.ClientSpeechToText.Recognize(ctx, &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_MP3,
			SampleRateHertz: 16000,
			LanguageCode:    "en-US",
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{Content: audio},
		},
	},
	)
	if err != nil {
		return "", err
	}

	// Extract the transcriptions from the response
	var transcriptions []string
	for _, result := range resp.Results {
		for _, alternative := range result.Alternatives {
			transcriptions = append(transcriptions, alternative.Transcript)
		}
	}
	fmt.Println("RESP:", resp.Results)

	// Combine the transcriptions into a single string
	transcription := strings.Join(transcriptions, " ")

	return transcription, nil
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

func (s Services) Podcast(c *gin.Context, conn *websocket.Conn, sessionId string) {
	// Read binary data from WebSocket message
	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Println("Read error:", err)
		return
	}
	podcastContext := string(message)

	// Create a new chat chatSession
	chatSession := s.model.StartChat()
	chatSession.History = []*genai.Content{}

	// Create a new channel for user interaction prompt store it in users session
	interactionPrompt := make(chan []byte)
	mu.Lock()
	s.storage.PodcastSessions[sessionId] = &models.PodcastSession{
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
				podcastTranscript, err := s.SendMessages(c, s.storage.PodcastSessions[sessionId].ChatSession, s.storage.PodcastSessions[sessionId].PodcastContext)
				// podcastTranscript, err := s.MockSendMessages(c, s.storage.PodcastSessions[sessionId].ChatSession, s.storage.PodcastSessions[sessionId].PodcastContext)
				if err != nil {
					log.Fatal(err)
				}
				err = s.Podcasting(c, conn, sessionId, podcastTranscript, s.ClientTextToSpeech, setVoices, StopChan)
				if err != nil {
					return
				}
			case <-StopChan:
				fmt.Println("Podcast finished successfully")
				conn.Close()
				break P
			}
		}
	}()
}

func (s *Services) Podcasting(c *gin.Context, conn *websocket.Conn, sessionID string, podcastTranscript []models.Segment, clientTextToSpeech *texttospeech.Client, setVoices map[string]string, StopChan chan interface{}) error {
	// history of the conversation that has been done so far, in case of users interaction we wont send all messages as the context again
	var history []*genai.Content
	history = append(history, &genai.Content{Role: "User", Parts: []genai.Part{genai.Text(s.storage.PodcastSessions[sessionID].PodcastContext)}})
	newBatch := make(chan []byte)
	go func(newBatch chan []byte) {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Read error:", err)
			}
			newBatch <- message
			time.Sleep(1 * time.Second)
		}
	}(newBatch)

	//iterate over the podcast transcript
	for i, segment := range podcastTranscript {
		select {
		case <-newBatch:
			fmt.Printf("%s: %s\n", segment.Speaker, segment.Text)
			// write podcast message to the websocket
			err := conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%s: %s\n", segment.Speaker, segment.Text)))
			if err != nil {
				log.Println("Error writing ws message:", err)
				break
			}

			audio, err := s.TextToSpeech(c, segment.Text, setVoices[segment.Speaker])
			if err != nil {
				return err
			}
			err = conn.WriteMessage(websocket.BinaryMessage, audio)
			if err != nil {
				log.Println("Error writing WS binary message:", err)
				return err
			}

			time.Sleep(3 * time.Second) // Simulate time delay between segments
		case userPrompt := <-s.storage.PodcastSessions[sessionID].InteractionPrompt:
			// on user interaction we regenerate the podcast based on the user interaction
			fmt.Printf("\n\n%v\n\n", string(userPrompt))

			// write users intercation to the websocket
			err := conn.WriteMessage(websocket.TextMessage, userPrompt)
			if err != nil {
				fmt.Println("Error writing ws message:", err)
				break
			}

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
			mu.Lock()
			s.storage.PodcastSessions[sessionID].ChatSession.History = history
			s.storage.PodcastSessions[sessionID].InteractionPrompt = make(chan []byte)
			mu.Unlock()

			return err
		}
	}
	close(StopChan)
	return nil
}

func (s Services) UserInteraction(c *gin.Context, sessionID string) error {
	// Retrieve the audio file from the form data
	audioFile, err := c.FormFile("audio_file")
	if err != nil {
		return fmt.Errorf("invalid audio file: %v", err)
	}

	// Open the audio file
	audio, err := audioFile.Open()
	if err != nil {
		return fmt.Errorf("error opening audio file: %v", err)
	}
	defer audio.Close()

	// Read the audio file content
	audioData, err := io.ReadAll(audio)
	if err != nil {
		return fmt.Errorf("error reading audio file: %v", err)
	}

	// Check the session ID
	mu.Lock()
	podcastSession, exists := s.storage.PodcastSessions[sessionID]
	if !exists {
		return fmt.Errorf("session is not in storage.podcastSessions: %v", err)
	}

	// podcastSession.InteractionPrompt <- []byte(fmt.Sprintf("USERS INTERACTION: %v\n", req.UsersInteraction))
	// userInteraction := formData.Value["user_interaction"][0]

	userInteraction, err := s.SpeechToText(c, audioData)
	if err != nil {
		return fmt.Errorf("error processing audio: %v", err)
	}
	podcastSession.InteractionPrompt <- []byte(fmt.Sprintf("USERS INTERACTION: %v\n", userInteraction))
	mu.Unlock()

	return nil
}

func (s *Services) SendMessages(ctx context.Context, session *genai.ChatSession, podcastContext string) ([]models.Segment, error) {
	resp, err := session.SendMessage(ctx, genai.Text(podcastContext))
	if err != nil {
		log.Fatalf("Error sending message: %v\n", err)
	}
	var podcast models.Podcast
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			if err := json.Unmarshal([]byte(txt), &podcast); err != nil {
				log.Fatal(err)
			}
		}
	}
	return podcast.Podcast, nil
}

func (s *Services) MockSendMessages(ctx context.Context, session *genai.ChatSession, podcastContext string) ([]models.Segment, error) {

	log.Printf("Chat Session SEDDD: %+v", session)
	if session == nil {
		return []models.Segment{}, errors.New("chatSession is nil")
	}

	// Add more logging to trace the function execution
	log.Printf("Chat Session MOCKGEM: %+v", session)
	log.Printf("Podcast Context: %+v", podcastContext)

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
	var podcast models.Podcast
	err := json.Unmarshal([]byte(jsonData), &podcast)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return []models.Segment{}, err
	}

	return podcast.Podcast, nil
}
