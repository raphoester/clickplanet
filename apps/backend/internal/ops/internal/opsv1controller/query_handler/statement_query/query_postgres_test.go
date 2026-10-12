package statement_query_test

import (
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/opsv1controller/query_handler/statement_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/role"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	server *cppg.TestServer
	owner  *cppg.Postgres
	reader *cppg.Postgres
	query  statement_query.PostgresQuery
}

func (s *testSuite) SetupSuite() {
	s.server = cppg.StartTestServer(s.T())
	s.owner = s.server.OpenSchema(s.T(), "planet", fstest.MapFS{
		"1_create_tiles.up.sql":   {Data: []byte(`CREATE TABLE tiles (id integer PRIMARY KEY, country text NOT NULL)`)},
		"1_create_tiles.down.sql": {Data: []byte(`DROP TABLE tiles`)},
	})

	_, err := s.owner.ExecContext(s.T().Context(), role.Reader)
	s.Require().NoError(err, "the role file must run on a fresh postgres")
	_, err = s.owner.ExecContext(s.T().Context(), role.Reader)
	s.Require().NoError(err, "the role file runs at every deploy, so it must run twice")
	_, err = s.owner.ExecContext(s.T().Context(), `ALTER ROLE ops_reader PASSWORD 'reader'`)
	s.Require().NoError(err)

	s.reader = s.as("ops_reader", "reader")
	s.query = statement_query.NewPostgresQuery(s.reader, 10*time.Second)
}

func (s *testSuite) as(user, password string) *cppg.Postgres {
	one, none := 1, 0
	config := s.server.ConfigFor("public")
	config.User, config.Password = user, password
	config.Pool.MaxOpenConns, config.Pool.MaxIdleConns = &one, &none

	db := cppg.New(config)
	s.Require().NoError(db.Open())
	s.T().Cleanup(func() { _ = db.Close() })

	return db
}

func (s *testSuite) SetupTest() {
	_, err := s.owner.ExecContext(s.T().Context(), `
		TRUNCATE planet.tiles;
		INSERT INTO planet.tiles VALUES (1, 'fr'), (2, 'bg'), (3, 'pt');
	`)
	s.Require().NoError(err)
}

func (s *testSuite) tiles() int {
	var count int
	s.Require().NoError(s.owner.QueryRowContext(s.T().Context(), `SELECT count(*) FROM planet.tiles`).Scan(&count))
	return count
}

func (s *testSuite) TestATableOfAnotherModulesSchemaIsRead() {
	answer, err := s.query.Rows(s.T().Context(), ` SELECT id, country FROM planet.tiles ORDER BY id `, 0)

	s.Require().NoError(err)
	s.Equal([]string{"id", "country"}, answer.GetColumns())
	s.False(answer.GetTruncated())
	s.Require().Len(answer.GetRows(), 3)
	s.Equal([]any{float64(1), "fr"}, answer.GetRows()[0].AsSlice())
	s.Equal([]any{float64(3), "pt"}, answer.GetRows()[2].AsSlice())
}

func (s *testSuite) TestATableMadeAfterTheRoleIsReadToo() {
	_, err := s.owner.ExecContext(s.T().Context(), `
		CREATE SCHEMA IF NOT EXISTS chat;
		CREATE TABLE IF NOT EXISTS chat.messages (body text);
		INSERT INTO chat.messages VALUES ('hello');
	`)
	s.Require().NoError(err)

	answer, err := s.query.Rows(s.T().Context(), `SELECT body FROM chat.messages LIMIT 1`, 0)

	s.Require().NoError(err)
	s.Equal([]any{"hello"}, answer.GetRows()[0].AsSlice())
}

func (s *testSuite) TestAStatementThatWritesIsRefusedAndChangesNothing() {
	for _, statement := range []string{
		`INSERT INTO planet.tiles VALUES (4, 'de')`,
		`UPDATE planet.tiles SET country = 'de'`,
		`DELETE FROM planet.tiles`,
		`TRUNCATE planet.tiles`,
		`DROP TABLE planet.tiles`,
		`CREATE TABLE planet.mine (id integer)`,
		`WITH gone AS (DELETE FROM planet.tiles RETURNING id) SELECT * FROM gone`,
	} {
		_, err := s.query.Rows(s.T().Context(), statement, 0)

		s.Require().ErrorIs(err, statement_query.ErrRefused, statement)
	}

	s.Equal(3, s.tiles())
}

func (s *testSuite) TestSeveralStatementsInOneStringAreRefused() {
	_, err := s.query.Rows(s.T().Context(), `SELECT 1; DELETE FROM planet.tiles`, 0)

	s.Require().ErrorIs(err, statement_query.ErrRefused)
	s.Equal(3, s.tiles())
}

func (s *testSuite) TestTheRoleCannotWriteEvenInAReadWriteTransaction() {
	tx, err := s.reader.BeginTx(s.T().Context(), nil)
	s.Require().NoError(err)
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(s.T().Context(), `SET TRANSACTION READ WRITE`)
	s.Require().NoError(err)

	_, err = tx.ExecContext(s.T().Context(), `DELETE FROM planet.tiles`)

	s.Require().ErrorContains(err, "permission denied")
	s.Equal(3, s.tiles())
}

func (s *testSuite) TestTheRoleCannotRunAProgramOrReadAFileOfTheServer() {
	for _, statement := range []string{
		`COPY (SELECT 1) TO PROGRAM 'true'`,
		`SELECT pg_read_file('/etc/passwd')`,
	} {
		_, err := s.query.Rows(s.T().Context(), statement, 0)

		s.Require().ErrorIs(err, statement_query.ErrRefused, statement)
		s.Require().ErrorContains(err, "permission denied", statement)
	}
}

func (s *testSuite) TestMoreRowsThanTheLimitAreCutAndSaidSo() {
	answer, err := s.query.Rows(s.T().Context(), `SELECT generate_series(1, 10)`, 3)

	s.Require().NoError(err)
	s.True(answer.GetTruncated())
	s.Len(answer.GetRows(), 3)
}

func (s *testSuite) TestExactlyTheLimitIsNotCut() {
	answer, err := s.query.Rows(s.T().Context(), `SELECT generate_series(1, 3)`, 3)

	s.Require().NoError(err)
	s.False(answer.GetTruncated())
	s.Len(answer.GetRows(), 3)
}

func (s *testSuite) TestNoLimitIsAThousandRows() {
	answer, err := s.query.Rows(s.T().Context(), `SELECT generate_series(1, 1500)`, 0)

	s.Require().NoError(err)
	s.True(answer.GetTruncated())
	s.Len(answer.GetRows(), 1000)
}

func (s *testSuite) TestALimitOverTheHighestIsRefused() {
	_, err := s.query.Rows(s.T().Context(), `SELECT 1`, statement_query.MaxLimit+1)

	s.Require().ErrorIs(err, statement_query.ErrLimitTooHigh)
}

func (s *testSuite) TestAnAnswerOfTooManyBytesIsCutAndSaidSo() {
	answer, err := s.query.Rows(s.T().Context(), `SELECT repeat('x', 1000000) FROM generate_series(1, 20)`, 100)

	s.Require().NoError(err)
	s.True(answer.GetTruncated())
	s.Len(answer.GetRows(), 9)
}

func (s *testSuite) TestNoRowIsStillItsColumns() {
	answer, err := s.query.Rows(s.T().Context(), `SELECT id FROM planet.tiles WHERE id < 0`, 0)

	s.Require().NoError(err)
	s.Equal([]string{"id"}, answer.GetColumns())
	s.Empty(answer.GetRows())
	s.False(answer.GetTruncated())
}

func (s *testSuite) TestEachTypeComesBackAsSomethingJSONCanCarry() {
	answer, err := s.query.Rows(s.T().Context(), `
		SELECT '\xdeadbeef'::bytea, '203.0.113.7'::inet, 1.50::numeric, NULL::text, 'NaN'::float8, 2.5::float8,
		       '0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10'::uuid, '{"a":1}'::jsonb, true,
		       9007199254740993::bigint, '2026-10-08T08:00:00.5Z'::timestamptz AT TIME ZONE 'UTC'
	`, 0)

	s.Require().NoError(err)
	s.Equal([]any{
		`\xdeadbeef`, "203.0.113.7", "1.50", nil, "NaN", 2.5,
		"0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10", `{"a": 1}`, true,
		"9007199254740993", "2026-10-08T08:00:00.5Z",
	}, answer.GetRows()[0].AsSlice())
}

func (s *testSuite) TestAnEmptyStatementIsNotSentToPostgres() {
	_, err := s.query.Rows(s.T().Context(), " \n\t", 0)

	s.Require().ErrorIs(err, statement_query.ErrNoStatement)
}

func (s *testSuite) TestAStatementThatIsNotSQLSaysWhatPostgresSaid() {
	_, err := s.query.Rows(s.T().Context(), `SELEC 1`, 0)

	s.Require().ErrorIs(err, statement_query.ErrRefused)
	s.Require().ErrorContains(err, "syntax error")
}

func (s *testSuite) TestAStatementThatOutlastsItsTimeIsStopped() {
	hasty := statement_query.NewPostgresQuery(s.reader, 300*time.Millisecond)
	start := time.Now()

	_, err := hasty.Rows(s.T().Context(), `SELECT pg_sleep(30)`, 0)

	s.Require().ErrorIs(err, statement_query.ErrTooSlow)
	s.Less(time.Since(start), 5*time.Second)
}

func (s *testSuite) TestALockTakenByOneStatementIsGoneBeforeTheNext() {
	_, err := s.query.Rows(s.T().Context(), `SELECT pg_advisory_lock(42)`, 0)
	s.Require().NoError(err)

	answer, err := s.query.Rows(s.T().Context(), `SELECT count(*) FROM pg_locks WHERE locktype = 'advisory'`, 0)

	s.Require().NoError(err)
	s.Equal([]any{float64(0)}, answer.GetRows()[0].AsSlice())
}

func (s *testSuite) TestARoleThatCannotSignInIsUnreachableNotRefused() {
	stranger := statement_query.NewPostgresQuery(s.as("ops_reader", "not-the-password"), 10*time.Second)

	_, err := stranger.Rows(s.T().Context(), `SELECT 1`, 0)

	s.Require().ErrorIs(err, statement_query.ErrUnreachable)
	s.Require().ErrorContains(err, "password authentication failed")
}
