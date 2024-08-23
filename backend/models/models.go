package models

import "github.com/google/generative-ai-go/genai"

type Segment struct {
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
}
type Podcast struct {
	Podcast []Segment `json:"podcast"`
}

type PodcastSession struct {
	InteractionPrompt chan []byte
	PodcastContext    string
	ChatSession       *genai.ChatSession
}
