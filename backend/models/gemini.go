package models

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/generative-ai-go/genai"
)

type Segment struct {
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
}
type Podcast struct {
	Podcast []Segment `json:"podcast"`
}

type Gemini struct {
	model *genai.GenerativeModel
}

func NewGemini(model *genai.GenerativeModel) *Gemini {
	return &Gemini{model: model}
}

func (g Gemini) SendMessages(ctx context.Context, session *genai.ChatSession, podcastContext string) ([]Segment, error) {
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

func (g Gemini) StartChat() *genai.ChatSession {
	return &genai.ChatSession{}
}

func (g Gemini) MockGemini(ctx context.Context, session *genai.ChatSession) ([]Segment, error) {
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
