package domain

type Mode string

const (
	ModeChat      Mode = "chat"
	ModeQuiz      Mode = "quiz"
	ModeTranslate Mode = "translate"
	ModeGap       Mode = "gap"
	ModeScramble  Mode = "scramble"
)

var validModes = map[Mode]bool{
	ModeChat:      true,
	ModeQuiz:      true,
	ModeTranslate: true,
	ModeGap:       true,
	ModeScramble:  true,
}

func (m Mode) IsValid() bool {
	return validModes[m]
}
