package messagebus

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"go.uber.org/atomic"

	"golang.org/x/sync/errgroup"

	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"

	"github.com/simple-container-com/go-aws-lambda-sdk/pkg/logger"
)

type TestMsg struct {
	ID string `bson:"ID"`
}

func TestMessageBus(t *testing.T) {
	RegisterTestingT(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connStr, cleanup, err := startMongodb()
	Expect(err).To(BeNil())
	defer cleanup()

	broker, err := NewMessageBus[TestMsg](ctx, logger.NewLogger(), connStr, "messages")
	Expect(err).To(BeNil())

	sessionID := lo.RandomString(5, lo.LowerCaseLettersCharset)

	errG, ctx := errgroup.WithContext(ctx)

	receivedWrongEvent := atomic.NewBool(false)
	receivedCorrectEvent := atomic.NewBool(false)
	// expect to receive my id
	errG.Go(func() error {
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return broker.OnMessage(ctx, sessionID,
			func() {
				// spam with different ids
				lo.ForEach(lo.Range(10), func(idx int, _ int) {
					err = broker.Send(ctx, fmt.Sprintf("not-my-id-%d", idx), TestMsg{
						ID: fmt.Sprintf("not-my-id-%d", idx),
					})
					Expect(err).To(BeNil())
				})
				// send my id
				err = broker.Send(ctx, sessionID, TestMsg{
					ID: "test-id",
				})
				Expect(err).To(BeNil())
			},
			func(evt TestMsg) {
				if evt.ID != "test-id" {
					receivedWrongEvent.Store(true)
				} else {
					receivedCorrectEvent.Store(true)
				}
			})
	})
	_ = errG.Wait()
	Expect(receivedWrongEvent.Load()).To(BeFalse())
	Expect(receivedCorrectEvent.Load()).To(BeTrue())
}

func startMongodb() (string, func(), error) {
	ctx := context.Background()
	mongodbContainer, err := mongodb.Run(ctx, "mongo:6", mongodb.WithReplicaSet("test"))
	if err != nil {
		return "", nil, err
	}
	connStr, err := mongodbContainer.ConnectionString(ctx)
	if err != nil {
		return "", nil, err
	}
	urlObj, err := url.Parse(connStr)
	if err != nil {
		return "", nil, err
	}

	urlObj.Path = "/" + lo.RandomString(5, lo.LowerCaseLettersCharset)
	return connStr, func() {
		_ = mongodbContainer.Terminate(ctx)
	}, err
}
