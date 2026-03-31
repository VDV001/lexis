package domain

// ChatRequest is sent to AI for free practice chat
type ChatRequest struct {
	UserID    string
	Messages  []Message
	System    string
	Model     string
	MaxTokens int
}

type Message struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

// ChatDelta is a streaming chunk from AI
type ChatDelta struct {
	Type       string      `json:"type"` // "delta", "correction", "feedback", "words", "done"
	Content    string      `json:"content,omitempty"`
	Correction *Correction `json:"correction,omitempty"`
	Feedback   *Feedback   `json:"feedback,omitempty"`
	Words      []string    `json:"words,omitempty"`
}

type Correction struct {
	Original    string `json:"original"`
	Fixed       string `json:"fixed"`
	Explanation string `json:"explanation"`
}

type Feedback struct {
	Type string `json:"type"` // "good", "note", "error"
	Text string `json:"text"`
}

// ExerciseRequest is for quiz/translate/gap/scramble generation
type ExerciseRequest struct {
	Mode      string // "quiz", "translate", "gap", "scramble"
	System    string
	Model     string
	MaxTokens int
}

// Exercise is the raw JSON response from AI for any exercise type
type Exercise struct {
	Raw string // Raw JSON from AI, parsed by the usecase layer
}

// CheckRequest is for checking user answers
type CheckRequest struct {
	Mode       string
	System     string
	Model      string
	UserAnswer string
	Context    string // Original exercise JSON for context
	MaxTokens  int
}

// CheckResult is the AI's evaluation of user's answer
type CheckResult struct {
	Raw string // Raw JSON from AI
}
