package ratings

import (
	"maps"
	"sync"
)

// Book is the rating of every question an answer moved, in memory, as bonuses.quiz_questions committed it. A question
// it holds nothing of is at its prior, which is the bank's to say. The draw reads it and never postgres.
type Book struct {
	mu   sync.RWMutex
	kept map[QuestionID]Rating
}

// NewBook starts from the ratings postgres keeps, read at boot.
func NewBook(kept map[QuestionID]Rating) *Book {
	return &Book{kept: maps.Clone(kept)}
}

// Rated is the rating answers moved a question to, and false for a question no answer moved.
func (b *Book) Rated(question QuestionID) (Rating, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	rating, ok := b.kept[question]
	return rating, ok
}

// Keep writes a question's rating once its answer committed.
func (b *Book) Keep(question QuestionID, rating Rating) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.kept == nil {
		b.kept = make(map[QuestionID]Rating)
	}
	b.kept[question] = rating
}
