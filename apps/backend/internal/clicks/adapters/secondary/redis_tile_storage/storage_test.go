package redis_tile_storage_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/redis_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xenvs"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
	storage *redis_tile_storage.Storage
	redis   *xenvs.Redis
}

func (s *testSuite) SetupSuite() {
	var err error
	s.redis, err = xenvs.NewRedis()
	s.Require().NoError(err)

	setAndPublishOnStreamSha1 := s.redis.ScriptsMap["setAndPublishOnStream"]
	s.storage = redis_tile_storage.New(
		s.redis.Client,
		redis_tile_storage.Config{SetAndPublishOnStreamSha1: setAndPublishOnStreamSha1},
		logging.NewSLogger(),
	)
}

func (s *testSuite) TearDownSuite() {
	s.Require().NoError(s.redis.Destroy())
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.redis.Clean())
}

func (s *testSuite) TestSetAndPublish() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	listener, err := s.storage.Subscribe(ctx)
	s.Require().NoError(err)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			s.T().Errorf("timeout")
		case val := <-listener:
			s.Assert().Equal("fr", val.Value)
			s.Assert().Equal(uint32(10), val.Tile)
			s.Assert().Equal("", val.Previous)
		}
	}()

	err = s.storage.Set(context.Background(), 10, "fr")
	s.Require().NoError(err)

	wg.Wait()
}

func (s *testSuite) TestSetAndPublishWithOverride() {
	previousValue := "us"
	newValue := "fr"

	err := s.storage.Set(context.Background(), 10, previousValue)
	s.Require().NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	listener, err := s.storage.Subscribe(ctx)
	s.Require().NoError(err)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			s.T().Errorf("timeout")
		case val := <-listener:
			s.Assert().Equal(newValue, val.Value)
			s.Assert().Equal(uint32(10), val.Tile)
			s.Assert().Equal(previousValue, val.Previous)
		}
	}()

	err = s.storage.Set(context.Background(), 10, newValue)
	s.Require().NoError(err)

	wg.Wait()
}

func (s *testSuite) TestSetAndPublishWithOverrideAndNoChange() {
	constantValue := "fr"

	err := s.storage.Set(context.Background(), 10, constantValue)
	s.Require().NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	listener, err := s.storage.Subscribe(ctx)
	s.Require().NoError(err)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			s.T().Logf("as expected, no message was received")
		case val := <-listener:
			s.T().Errorf("unexpected value %v", val)
		}
	}()

	err = s.storage.Set(context.Background(), 10, constantValue)
	s.Require().NoError(err)

	wg.Wait()
}

func (s *testSuite) TestSetAndPublishWithALotOfConcurrentMessages() {
	const (
		messages  = 100_000
		senders   = 32
		receivers = 16
	)

	constantValue := "fr"

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	listener, err := s.storage.Subscribe(ctx)
	s.Require().NoError(err)

	rcvMap := make(map[uint32]struct{}, messages)
	rcvMapMu := sync.Mutex{}

	// closed as soon as every expected tile has been seen, so the receivers stop
	// waiting on a listener that has nothing left to deliver.
	allReceived := make(chan struct{})
	allReceivedOnce := sync.Once{}

	wgRcv := sync.WaitGroup{}
	for i := 0; i < receivers; i++ {
		wgRcv.Add(1)
		go func() {
			defer wgRcv.Done()
			for {
				select {
				case <-allReceived:
					return
				case <-ctx.Done():
					return
				case value, ok := <-listener:
					if !ok {
						return
					}

					s.Assert().Equal(constantValue, value.Value)

					rcvMapMu.Lock()
					_, duplicate := rcvMap[value.Tile]
					rcvMap[value.Tile] = struct{}{}
					complete := len(rcvMap) == messages
					rcvMapMu.Unlock()

					s.Assert().False(duplicate, "tile %d was received twice", value.Tile)

					if complete {
						allReceivedOnce.Do(func() { close(allReceived) })
					}
				}
			}
		}()
	}

	// Bounded sender pool: one goroutine per tile makes the test hostage to the
	// scheduler under -race without exercising anything the pool doesn't.
	tiles := make(chan uint32)
	wgSend := sync.WaitGroup{}
	for i := 0; i < senders; i++ {
		wgSend.Add(1)
		go func() {
			defer wgSend.Done()
			for tile := range tiles {
				if err := s.storage.Set(ctx, tile, constantValue); err != nil {
					s.T().Errorf("failed to set tile %d: %v", tile, err)
					return
				}
			}
		}()
	}

sending:
	for i := uint32(1); i <= messages; i++ {
		select {
		case tiles <- i:
		case <-ctx.Done(): // every sender gave up; don't block the test forever
			break sending
		}
	}
	close(tiles)

	wgSend.Wait()
	wgRcv.Wait()

	s.Require().NoError(ctx.Err(), "timed out before receiving every update")
	s.Assert().Equal(messages, len(rcvMap))
}

func (s *testSuite) TestGetStateByBatch() {
	constantValue := "fr"

	err := s.storage.Set(context.Background(), 10, constantValue)
	s.Require().NoError(err)

	err = s.storage.Set(context.Background(), 20, constantValue)
	s.Require().NoError(err)

	err = s.storage.Set(context.Background(), 30, constantValue)
	s.Require().NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	state, err := s.storage.GetStateBatch(ctx, 10, 30) // bounds are inclusive
	s.Require().NoError(err)

	s.Assert().Equal(3, len(state))
	s.Assert().Equal(constantValue, state[10])
	s.Assert().Equal(constantValue, state[20])
	s.Assert().Equal(constantValue, state[30])
}

func (s *testSuite) TestPastUpdates() {
	xAdd := func(t time.Time, i int) error {
		return s.redis.Client.XAdd(context.Background(), &redis.XAddArgs{
			Stream: "tileUpdates", // stream name corresponds in ./stream_storage.go
			ID:     fmt.Sprintf("%d-%d", t.UnixMilli(), i),
			Values: []string{"t", fmt.Sprintf("%d", i), "n", "fr", "o", "us"},
		}).Err()
	}

	xAdd1Time := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // 00:00:00

	err := xAdd(xAdd1Time, 10)
	s.Require().NoError(err)

	xAdd2Time := xAdd1Time.Add(45 * time.Minute) // 00:45:00
	err = xAdd(xAdd2Time, 11)

	xAdd3Time := xAdd2Time.Add(10 * time.Minute) // 00:55:00
	err = xAdd(xAdd3Time, 12)

	queryTime := xAdd3Time.Add(30 * time.Minute) // 01:25:00

	pastUpdates, err := s.storage.PastUpdates(context.Background(), 1*time.Hour, queryTime)
	s.Require().NoError(err)
	s.Assert().Equal(2, len(pastUpdates))

	s.Assert().Equal(uint32(11), pastUpdates[0].Tile)
	s.Assert().Equal("fr", pastUpdates[0].Value)

	s.Assert().Equal(uint32(12), pastUpdates[1].Tile)
	s.Assert().Equal("fr", pastUpdates[1].Value)
}
