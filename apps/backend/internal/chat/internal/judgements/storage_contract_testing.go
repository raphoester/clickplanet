//go:build testing

package judgements

import (
	"context"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type StorageContractSuite struct {
	suite.Suite

	NewStorage func() Storage

	storage Storage
}

var contractAt = time.Date(2026, 11, 7, 12, 0, 0, 123_456_000, time.UTC)

func (s *StorageContractSuite) SetupTest() {
	s.storage = s.NewStorage()
}

func (s *StorageContractSuite) saved(snapshot Snapshot) Judgement {
	judgement, err := snapshot.Restore()
	s.Require().NoError(err)
	s.Require().NoError(s.storage.Save(context.Background(), judgement))
	return judgement
}

func (s *StorageContractSuite) TestAJudgementIsReadBackByItsReport() {
	kept := s.saved(Snapshot{
		Report: ReportID{15: 1}, Message: "m-1", Verdict: Acted, Judged: true, Flagged: true,
		Reason: "offence 3 (chat): a slur; 1 message removed, muted 1 hour", At: contractAt,
	})

	read, err := s.storage.OfReport(context.Background(), ReportID{15: 1})

	s.Require().NoError(err)
	s.Equal(kept.TakeSnapshot(), read.TakeSnapshot())
}

func (s *StorageContractSuite) TestAReportNeverJudgedIsNotJudged() {
	_, err := s.storage.OfReport(context.Background(), ReportID{15: 1})
	s.Require().ErrorIs(err, ErrNotJudged)

	_, err = s.storage.OfMessage(context.Background(), "m-1")
	s.Require().ErrorIs(err, ErrNotJudged)
}

func (s *StorageContractSuite) TestAReportJudgedTwiceKeepsTheFirst() {
	first := s.saved(Snapshot{Report: ReportID{15: 1}, Verdict: Dismissed, Judged: true, Reason: "first", At: contractAt})
	s.saved(Snapshot{Report: ReportID{15: 1}, Verdict: Acted, Judged: true, Reason: "second", At: contractAt.Add(time.Minute)})

	read, err := s.storage.OfReport(context.Background(), ReportID{15: 1})

	s.Require().NoError(err)
	s.Equal(first.TakeSnapshot(), read.TakeSnapshot())
}

func (s *StorageContractSuite) TestAMessageIsJudgedByTheFirstJudgementThatJudgedIt() {
	s.saved(Snapshot{Report: ReportID{15: 1}, Message: "m-1", Verdict: LeftOpen, Reason: "the cap", At: contractAt})
	judged := s.saved(Snapshot{
		Report: ReportID{15: 2}, Message: "m-1", Verdict: Acted, Judged: true, Reason: "offence 1", At: contractAt.Add(time.Minute),
	})
	s.saved(Snapshot{
		Report: ReportID{15: 3}, Message: "m-1", Verdict: Dismissed, Judged: true, Reason: "later", At: contractAt.Add(time.Hour),
	})
	s.saved(Snapshot{Report: ReportID{15: 4}, Message: "m-2", Verdict: Dismissed, Judged: true, Reason: "another", At: contractAt})

	read, err := s.storage.OfMessage(context.Background(), "m-1")

	s.Require().NoError(err)
	s.Equal(judged.TakeSnapshot(), read.TakeSnapshot())
}

func (s *StorageContractSuite) TestAReportOfAPlayerStandsForNoMessage() {
	s.saved(Snapshot{Report: ReportID{15: 1}, Message: messages.MessageID(""), Verdict: Acted, Judged: true, Reason: "r", At: contractAt})

	_, err := s.storage.OfMessage(context.Background(), "")

	s.Require().ErrorIs(err, ErrNotJudged)
}

func (s *StorageContractSuite) TestThePruneDeletesWhatIsOlderThanTheCutoff() {
	s.saved(Snapshot{Report: ReportID{15: 1}, Verdict: Acted, Judged: true, Reason: "old", At: contractAt})
	s.saved(Snapshot{Report: ReportID{15: 2}, Verdict: Acted, Judged: true, Reason: "new", At: contractAt.Add(time.Hour)})

	deleted, err := s.storage.DeleteBefore(context.Background(), contractAt.Add(time.Minute))

	s.Require().NoError(err)
	s.Equal(int64(1), deleted)
	_, err = s.storage.OfReport(context.Background(), ReportID{15: 1})
	s.Require().ErrorIs(err, ErrNotJudged)
	_, err = s.storage.OfReport(context.Background(), ReportID{15: 2})
	s.Require().NoError(err)
}
