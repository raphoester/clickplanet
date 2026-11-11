package quizzes

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"slices"

	quizdata "github.com/raphoester/clickplanet.lol-backend/generated/quiz"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings"
)

// Rated is the rating of each question its answers moved (ratings.Book): one no answer moved is at its prior.
type Rated interface {
	Rated(question ratings.QuestionID) (ratings.Rating, bool)
}

type Bank struct {
	version string

	shares Shares
	rated  Rated

	all  []Question
	byID map[ratings.QuestionID]int

	subjects  []string
	bySubject map[string]about

	anywhere []Question
}

type about struct {
	named   []Question
	unnamed []Question
}

func (a about) size() int { return len(a.named) + len(a.unnamed) }

func (a about) with(question Question) about {
	if question.NamesSubject {
		a.named = append(a.named, question)
	} else {
		a.unnamed = append(a.unnamed, question)
	}

	return a
}

func Load(shares Shares, rated Rated) (*Bank, error) {
	return read(quizdata.Bank(), shares, rated)
}

// A field the build does not know, such as one the generator added since, is read past: a bank of a later build loads.
func read(blob []byte, shares Shares, rated Rated) (*Bank, error) {
	var file struct {
		Format    int        `json:"format"`
		Questions []Question `json:"questions"`
	}
	if err := json.Unmarshal(blob, &file); err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", source, err)
	}

	if file.Format != format {
		return nil, fmt.Errorf("%s is format %d, this build reads %d", source, file.Format, format)
	}
	if len(file.Questions) == 0 {
		return nil, fmt.Errorf("%s holds no questions", source)
	}

	bank := &Bank{
		version:   versionOf(blob),
		shares:    shares,
		rated:     rated,
		all:       file.Questions,
		byID:      make(map[ratings.QuestionID]int, len(file.Questions)),
		bySubject: make(map[string]about),
	}

	for i, question := range file.Questions {
		if err := question.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", source, err)
		}
		if _, seen := bank.byID[question.ID]; seen {
			return nil, fmt.Errorf("%s: two questions are called %q", source, question.ID)
		}
		bank.byID[question.ID] = i

		if question.Subject == "" {
			bank.anywhere = append(bank.anywhere, question)
			continue
		}

		if _, known := bank.bySubject[question.Subject]; !known {
			bank.subjects = append(bank.subjects, question.Subject)
		}
		bank.bySubject[question.Subject] = bank.bySubject[question.Subject].with(question)
	}

	slices.Sort(bank.subjects)

	return bank, nil
}

const (
	format = 1
	source = "bank.json"
)

func versionOf(blob []byte) string {
	sum := sha256.Sum256(blob)
	return hex.EncodeToString(sum[:4])
}

func (b *Bank) Version() string { return b.version }

func (b *Bank) Size() int { return len(b.all) }

func (b *Bank) Subjects() int { return len(b.subjects) }

// Rating is how hard a question is now: what its answers moved it to, else its prior. A question the bank does not hold
// is at FallbackPrior.
func (b *Bank) Rating(question ratings.QuestionID) ratings.Rating {
	if b.rated != nil {
		if rating, ok := b.rated.Rated(question); ok {
			return rating
		}
	}

	chance := FallbackPrior
	if i, ok := b.byID[question]; ok {
		chance = b.all[i].startingChance()
	}
	return ratings.PriorOf(chance)
}

// Draw picks a subject leaning to the board, then a question of that subject's pool weighed by how hard it is.
func (b *Bank) Draw(lean float64, fit Fit) Round {
	return b.round(b.question(lean, fit))
}

// A pool with nothing to draw (every question under the floor) is left out and the lean draws again among the rest;
// with nothing to draw anywhere, the whole bank is drawn flat.
func (b *Bank) question(lean float64, fit Fit) Question {
	pools, weights := b.pools(lean)
	for {
		i, ok := drawn(weights)
		if !ok {
			return b.all[index(len(b.all))]
		}

		pool := pools[i]
		if !fit.on() {
			if len(pool) == 0 {
				return b.all[index(len(b.all))]
			}
			return pool[index(len(pool))]
		}

		if question, ok := b.fitted(pool, fit); ok {
			return question
		}
		weights[i] = 0
	}
}

func (b *Bank) round(question Question) Round {
	wrong := shuffle(question.Wrong)[:Choices-1]

	options := make([]string, 0, Choices)
	options = append(options, question.Answer)
	options = append(options, wrong...)
	options = shuffle(options)

	correct := 0
	for i, option := range options {
		if option == question.Answer {
			correct = i
			break
		}
	}

	return Round{Question: question, Options: options, Correct: correct}
}

func shuffle(items []string) []string {
	out := make([]string, len(items))
	copy(out, items)

	for i := len(out) - 1; i > 0; i-- {
		j := index(i + 1)
		out[i], out[j] = out[j], out[i]
	}

	return out
}

func index(n int) int {
	if n <= 1 {
		return 0
	}

	drawn, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0
	}

	return int(drawn.Int64())
}
