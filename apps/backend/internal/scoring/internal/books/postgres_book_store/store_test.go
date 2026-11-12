package postgres_book_store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/postgres_book_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	books.BooksContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "scoring", migrations.FS)
	store := postgres_book_store.New(s.db)
	s.NewStore = func() books.Books { return store }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.BooksContractSuite.SetupTest()
}

func (s *testSuite) TestAnOpenWaitsForTheCloseThatHoldsTheBookAndFindsItMissed() {
	ctx := s.T().Context()
	store := postgres_book_store.New(s.db)
	missed := books.NewMissedBook("season-0", time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC))
	locked, release, closed := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		closed <- s.db.InTx(ctx, func(ctx context.Context) error {
			if err := store.Lock(ctx, missed.ID); err != nil {
				return err //nolint:wrapcheck // the store says what failed.
			}
			close(locked)
			<-release
			return store.KeepMissed(ctx, missed) //nolint:wrapcheck // the store says what failed.
		})
	}()
	select {
	case <-locked:
	case err := <-closed:
		s.Require().FailNow("the close never held the book", err)
	}

	opening := make(chan bool, 1)
	go func() {
		found := false
		s.NoError(s.db.InTx(ctx, func(ctx context.Context) error {
			if err := store.Lock(ctx, missed.ID); err != nil {
				return err //nolint:wrapcheck // the store says what failed.
			}
			var err error
			_, found, err = store.Missed(ctx, missed.ID)
			return err //nolint:wrapcheck // the store says what failed.
		}))
		opening <- found
	}()

	s.Never(func() bool { return len(opening) > 0 }, 300*time.Millisecond, 10*time.Millisecond, "the open waits for the close")
	close(release)
	s.Require().NoError(<-closed)
	s.True(<-opening, "once the close commits, the open finds the book missed")
}
