package narration

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

// A round of the war report's bench (docs/video-lab.md): the stories the editor chose at one moment of the game, and
// each written in each tone by each model. Nothing of it is posted.
type Bench struct {
	// The moment the findings measured.
	At        time.Time
	Stories   []posts.Story
	Editorial posts.Editorial
	Entries   []Entry
}

// One script of the bench. Story is its story's place in Stories, from 1. Failed says why there is no script.
type Entry struct {
	Story  int
	Tone   ToneName
	Model  Model
	Script Script
	Spend  posts.Spend
	Failed string
}

// What the round cost, the editor's call included.
func (b Bench) USD() float64 {
	total := b.Editorial.Spend.USD
	for _, entry := range b.Entries {
		total += entry.Spend.USD
	}
	return total
}
