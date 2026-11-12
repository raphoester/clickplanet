package postgres_rating_store

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// Store keeps every answer counted (bonuses.quiz_answers) and each question's rating (bonuses.quiz_questions).
func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

func (s *Store) SaveAnswer(ctx context.Context, answer ratings.Answer) error {
	var account *uuid.UUID
	if answer.Account != ratings.NoAccount {
		id := uuid.UUID(answer.Account)
		account = &id
	}

	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO quiz_answers (question_id, account_id, outcome, answer_ms, answered_at)
		VALUES ($1, $2, $3, $4, $5)
	`, string(answer.Question), account, string(answer.Outcome), max(answer.After.Milliseconds(), 0),
		answer.At); err != nil {
		return fmt.Errorf("failed to save an answer to %s: %w", answer.Question, err)
	}
	return nil
}

func (s *Store) SaveRating(ctx context.Context, question ratings.QuestionID, rating ratings.Rating) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO quiz_questions (question_id, rating, answers, right_answers) VALUES ($1, $2, $3, $4)
		ON CONFLICT (question_id) DO UPDATE
		SET rating = excluded.rating, answers = excluded.answers, right_answers = excluded.right_answers
	`, string(question), rating.Level(), rating.Answers(), rating.Right()); err != nil {
		return fmt.Errorf("failed to save the rating of %s: %w", question, err)
	}
	return nil
}

// Ratings is every question an answer moved, as it was last saved.
func (s *Store) Ratings(ctx context.Context) (map[ratings.QuestionID]ratings.Rating, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx,
		`SELECT question_id, rating, answers, right_answers FROM quiz_questions`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the questions' ratings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	kept := make(map[ratings.QuestionID]ratings.Rating)
	for rows.Next() {
		var (
			question       string
			level          float64
			answers, right int
		)
		if err := rows.Scan(&question, &level, &answers, &right); err != nil {
			return nil, fmt.Errorf("failed to read a question's rating: %w", err)
		}
		kept[ratings.QuestionID(question)] = ratings.RatingOf(level, answers, right)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the questions' ratings: %w", err)
	}

	return kept, nil
}

// ForgetAccount blanks an account's id on every answer it gave: the answers still count for their questions.
func (s *Store) ForgetAccount(ctx context.Context, account ratings.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx,
		`UPDATE quiz_answers SET account_id = NULL WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to forget the answers of %s: %w", account, err)
	}
	return nil
}
