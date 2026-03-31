package domain

import "fmt"

type Mode string

const (
	ModeChat      Mode = "chat"
	ModeQuiz      Mode = "quiz"
	ModeTranslate Mode = "translate"
	ModeGap       Mode = "gap"
	ModeScramble  Mode = "scramble"
)

type PromptSettings struct {
	UserName         string
	TargetLanguage   string
	ProficiencyLevel string
	VocabularyType   string
}

var levelContextMap = map[string]string{
	"a2": "A2 (elementary, very simple structures only)",
	"b1": "B1 (intermediate, Russian developer, limited active production)",
	"b2": "B2 (upper-intermediate, can handle complex sentences)",
	"c1": "C1 (advanced, near-native fluency)",
}

var vocabContextMap = map[string]string{
	"tech":     "Focus on technical vocabulary: software development, APIs, backend, infrastructure, Git, databases.",
	"literary": "Focus on general/literary vocabulary.",
	"business": "Focus on business vocabulary: meetings, presentations, negotiations, corporate communication.",
}

func BuildSystemPrompt(s PromptSettings, mode Mode) string {
	levelCtx := levelContextMap[s.ProficiencyLevel]
	if levelCtx == "" {
		levelCtx = levelContextMap["b1"]
	}
	vocabCtx := vocabContextMap[s.VocabularyType]
	if vocabCtx == "" {
		vocabCtx = vocabContextMap["tech"]
	}

	switch mode {
	case ModeChat:
		return chatPrompt(s.UserName, levelCtx, vocabCtx)
	case ModeQuiz:
		return quizPrompt(levelCtx, vocabCtx)
	case ModeTranslate:
		return translatePrompt(levelCtx, vocabCtx)
	case ModeGap:
		return gapPrompt(levelCtx, vocabCtx)
	case ModeScramble:
		return scramblePrompt(levelCtx, vocabCtx)
	default:
		return chatPrompt(s.UserName, levelCtx, vocabCtx)
	}
}

func chatPrompt(userName, levelCtx, vocabCtx string) string {
	return fmt.Sprintf(`You are an honest, direct English tutor for %s, a Russian developer. Level: %s. %s
Respond ONLY raw JSON (no markdown, no code fences):
{"reply":"2-4 sentences English reply appropriate for level","correction":null,"feedback":{"type":"good|note|error","text":"Russian feedback 1-2 sentences"},"error_type":null,"new_words":["word1","word2"]}
If error: "correction":{"original":"exact words with error","fixed":"corrected","explanation":"Russian brief explanation"}
"error_type": articles|tenses|prepositions|phrasal|vocabulary|word_order|null
Be honest. Never pretend errors don't exist.`, userName, levelCtx, vocabCtx)
}

func quizPrompt(levelCtx, vocabCtx string) string {
	return fmt.Sprintf(`Generate a %s English grammar/vocabulary question for a Russian developer. %s
ONLY raw JSON:
{"type":"grammar|vocabulary|phrasal","question":"Question text","options":["A","B","C","D"],"correct":0,"explanation":"Объяснение на русском (2-3 предложения)","error_type":"articles|tenses|prepositions|phrasal|vocabulary|word_order","words":["word1","word2"],"confidence":75}
Make distractors plausible. confidence = difficulty 50-95.`, levelCtx, vocabCtx)
}

func translatePrompt(levelCtx, vocabCtx string) string {
	return fmt.Sprintf(`Generate a Russian sentence for %s English learner to translate. %s
ONLY raw JSON:
{"russian":"Русское предложение","expected":"Expected translation","hint":"Ключевое слово","words":["word1"],"error_type":"tenses|prepositions|vocabulary|word_order"}`, levelCtx, vocabCtx)
}

func gapPrompt(levelCtx, vocabCtx string) string {
	return fmt.Sprintf(`Generate fill-in-the-gap for %s English learner. %s
ONLY raw JSON:
{"before":"Text before gap","answer":"correct word","after":"text after","options":["correct","wrong1","wrong2","wrong3"],"explanation":"Russian 2 sentences","error_type":"articles|tenses|prepositions|phrasal|vocabulary|word_order","words":["word"]}
First option MUST be the correct answer. Server shuffles them.`, levelCtx, vocabCtx)
}

func scramblePrompt(levelCtx, vocabCtx string) string {
	return fmt.Sprintf(`Generate word-scramble sentence for %s English learner. %s
ONLY raw JSON:
{"words":["Put","these","words","in","order"],"correct":"Put these words in order","translation":"Русский перевод","explanation":"Почему такой порядок (на русском)","vocab":["word1"]}`, levelCtx, vocabCtx)
}
