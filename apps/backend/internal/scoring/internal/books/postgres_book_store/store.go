package postgres_book_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ books.Books = (*Store)(nil)

const bookColumns = `
	SELECT book.book_id, book.starts_at, closed.ended_at, tiebreak.round_id
	FROM books AS book
	LEFT JOIN closed_books AS closed USING (book_id)
	LEFT JOIN book_tiebreaks AS tiebreak USING (book_id)
`

type scanner interface {
	Scan(dest ...any) error
}

func scanBook(row scanner) (books.Book, error) {
	var (
		id       string
		book     books.Book
		endedAt  sql.NullTime
		tiebreak sql.NullString
	)
	if err := row.Scan(&id, &book.StartsAt, &endedAt, &tiebreak); err != nil {
		return books.Book{}, err //nolint:wrapcheck // each caller says what it read.
	}
	book.ID, book.StartsAt = books.BookID(id), book.StartsAt.UTC()
	if endedAt.Valid {
		book = book.ClosedAt(endedAt.Time, books.RoundID(tiebreak.String))
	}
	return book, nil
}

func (s *Store) Find(ctx context.Context, id books.BookID) (books.Book, bool, error) {
	book, err := scanBook(s.db.On(ctx).QueryRowContext(ctx, bookColumns+` WHERE book.book_id = $1`, string(id)))
	if errors.Is(err, sql.ErrNoRows) {
		return books.Book{}, false, nil
	}
	if err != nil {
		return books.Book{}, false, fmt.Errorf("failed to read the book: %w", err)
	}
	return book, true, nil
}

func (s *Store) All(ctx context.Context) ([]books.Book, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, bookColumns+` ORDER BY book.starts_at, book.book_id`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the books: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var all []books.Book
	for rows.Next() {
		book, err := scanBook(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to read a book: %w", err)
		}
		all = append(all, book)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the books: %w", err)
	}
	return all, nil
}

func (s *Store) Open(ctx context.Context, book books.Book) (bool, error) {
	opening, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO books (book_id, starts_at) VALUES ($1, $2) ON CONFLICT (book_id) DO NOTHING
	`, string(book.ID), book.StartsAt)
	if err != nil {
		return false, fmt.Errorf("failed to open the book: %w", err)
	}
	return applied(opening, "failed to open the book")
}

func applied(result sql.Result, failure string) (bool, error) {
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("%s: %w", failure, err)
	}
	return affected > 0, nil
}

func (s *Store) Close(ctx context.Context, book books.Book) (bool, error) {
	closed := false
	err := s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		closing, err := tx.ExecContext(ctx, `
			INSERT INTO closed_books (book_id, ended_at) SELECT book_id, $2 FROM books WHERE book_id = $1
			ON CONFLICT (book_id) DO NOTHING
		`, string(book.ID), book.EndsAt)
		if err != nil {
			return fmt.Errorf("failed to close the book: %w", err)
		}
		if closed, err = applied(closing, "failed to close the book"); err != nil || !closed || book.Tiebreak == "" {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO book_tiebreaks (book_id, round_id) VALUES ($1, $2)`,
			string(book.ID), string(book.Tiebreak)); err != nil {
			return fmt.Errorf("failed to keep the book's tiebreak: %w", err)
		}
		return nil
	})
	return closed, err //nolint:wrapcheck // each step named what failed.
}
