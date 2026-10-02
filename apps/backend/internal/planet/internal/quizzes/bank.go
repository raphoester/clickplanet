package quizzes

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"slices"

	quizdata "github.com/raphoester/clickplanet.lol-backend/generated/quiz"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// Bank is every question the game can ask, indexed by what each is about. Read once at boot and
// never written, so it is safe to read from every caller's goroutine without a lock.
type Bank struct {
	name string

	// How hard the draw leans on the countries that are winning, and the board it reads to do it;
	// see weights.go.
	bias   float64
	shares Shares

	// Every question, in the bank's own order.
	all []Question

	// The countries some question is about, sorted, and the questions about each. A country with no
	// questions is not in here at all, which is what keeps the weighted draw from ever picking one.
	subjects  []string
	bySubject map[string][]Question

	// The questions about nowhere in particular. Drawn as if they were one more subject.
	anywhere []Question
}

// Load reads the embedded bank. It is fatal for the caller: a process that means to ask questions
// and cannot read its own is a process that would offer quizzes it cannot put.
// `shares` is the live map, read at every draw: the board moves while the process runs, and a
// snapshot taken at boot would have the quiz asking about last week's leaders forever. Nil is a
// flat draw.
func Load(config Config, shares Shares) (*Bank, error) {
	blob, name, err := quizdata.Bank()
	if err != nil {
		return nil, fmt.Errorf("failed to read the embedded quiz bank: %w", err)
	}

	var file struct {
		Format    int        `json:"format"`
		Questions []Question `json:"questions"`
	}
	if err := json.Unmarshal(blob, &file); err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", name, err)
	}

	// A format this build does not know is a bank written by a newer generator: its questions may
	// mean something else, so it is refused rather than half read.
	if file.Format != format {
		return nil, fmt.Errorf("%s is format %d, this build reads %d", name, file.Format, format)
	}
	if len(file.Questions) == 0 {
		return nil, fmt.Errorf("%s holds no questions", name)
	}

	bank := &Bank{
		name:      name,
		bias:      config.withDefaults().LeaderBias,
		shares:    shares,
		all:       file.Questions,
		bySubject: make(map[string][]Question),
	}

	seen := cpcolls.NewSetWithCapacity[string](len(file.Questions))
	for _, question := range file.Questions {
		if err := question.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if seen.Contains(question.ID) {
			return nil, fmt.Errorf("%s: two questions are called %q", name, question.ID)
		}
		seen.Add(question.ID)

		if question.Subject == "" {
			bank.anywhere = append(bank.anywhere, question)
			continue
		}

		if _, known := bank.bySubject[question.Subject]; !known {
			bank.subjects = append(bank.subjects, question.Subject)
		}
		bank.bySubject[question.Subject] = append(bank.bySubject[question.Subject], question)
	}

	// The subjects are walked in this order by the weighted draw, so it is fixed rather than a map's.
	slices.Sort(bank.subjects)

	return bank, nil
}

// format is the bank layout this build reads. It is in the file so the two can move apart.
const format = 1

// Name is the content-addressed file the bank was read from, for the boot line.
func (b *Bank) Name() string { return b.name }

// Size is how many questions there are.
func (b *Bank) Size() int { return len(b.all) }

// Subjects is how many countries some question is about.
func (b *Bank) Subjects() int { return len(b.subjects) }

// Draw picks a question and deals it into a round: three choices, shuffled, one of them right.
//
// Nothing here fails: a bank that loaded has at least one question, and a draw that cannot read the
// system's randomness falls back to the first of whatever it was choosing between rather than
// refusing to ask anything.
func (b *Bank) Draw() Round {
	return b.round(b.question())
}

func (b *Bank) question() Question {
	subject := b.subject()

	own := b.anywhere
	if subject != "" {
		own = b.bySubject[subject]
	}
	if len(own) == 0 {
		return b.all[index(len(b.all))]
	}

	return own[index(len(own))]
}

// round deals the choices. The wrong ones are drawn from the whole pool the question carries, so
// the same entry is not the same three choices twice: a bank that always dealt the first two would
// be a bank worth writing down.
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

// shuffle is a Fisher-Yates over a copy, from crypto/rand: a quiz whose order a client could
// predict is a quiz that answers itself. A read that fails leaves the rest of the order alone,
// which is still a shuffle of everything before it.
func shuffle(items []string) []string {
	out := make([]string, len(items))
	copy(out, items)

	for i := len(out) - 1; i > 0; i-- {
		j := index(i + 1)
		out[i], out[j] = out[j], out[i]
	}

	return out
}

// index is a uniform number below n, or 0 when the system's randomness cannot be read.
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
