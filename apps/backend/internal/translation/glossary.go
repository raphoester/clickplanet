package translation

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// The game's words, each with what it means and the word chosen in each language: every call carries them, so the
// page, the quiz and the titles say "tile" the same way.
//
//go:embed glossary.json
var glossaryJSON []byte

type Term struct {
	English string
	Means   string
	// Language → the word chosen for it.
	Words map[Language]string
}

type Glossary []Term

func ReadGlossary() (Glossary, error) {
	var file struct {
		Terms []map[string]string `json:"terms"`
	}
	if err := json.Unmarshal(glossaryJSON, &file); err != nil {
		return nil, fmt.Errorf("the glossary does not read: %w", err)
	}
	glossary := make(Glossary, 0, len(file.Terms))
	for _, fields := range file.Terms {
		term := Term{English: fields["en"], Means: fields["means"], Words: map[Language]string{}}
		for name, word := range fields {
			if name != "en" && name != "means" {
				term.Words[Language(name)] = word
			}
		}
		glossary = append(glossary, term)
	}
	return glossary, nil
}
