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
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Bank struct {
	version string

	bias   float64
	shares Shares

	all []Question

	subjects  []string
	bySubject map[string][]Question

	anywhere []Question
}

func Load(config Config, shares Shares) (*Bank, error) {
	blob := quizdata.Bank()

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
		bias:      config.withDefaults().LeaderBias,
		shares:    shares,
		all:       file.Questions,
		bySubject: make(map[string][]Question),
	}

	seen := cpcolls.NewSetWithCapacity[string](len(file.Questions))
	for _, question := range file.Questions {
		if err := question.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", source, err)
		}
		if seen.Contains(question.ID) {
			return nil, fmt.Errorf("%s: two questions are called %q", source, question.ID)
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
