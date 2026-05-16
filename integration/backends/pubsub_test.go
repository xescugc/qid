package backends_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gocloud.dev/pubsub"
)

var testCounter int

func uniqueTopicURL(system string) string {
	testCounter++
	name := fmt.Sprintf("qid_test_%d", testCounter)
	u := fmt.Sprintf("%s://%s", system, name)
	switch system {
	case "nats":
		u += "?natsv2"
	}
	return u
}

func uniqueSubURL(system string) string {
	name := fmt.Sprintf("qid_test_%d", testCounter)
	u := fmt.Sprintf("%s://%s", system, name)
	switch system {
	case "nats":
		u += "?queue=" + name + "&natsv2"
	case "kafka":
		u = fmt.Sprintf("kafka://%s-group?topic=%s", name, name)
	}
	return u
}

func TestPubSubBackends(t *testing.T) {
	for _, system := range pubsubSystems() {
		system := system
		t.Run(system, func(t *testing.T) {
			t.Run("ConnectAndShutdown", func(t *testing.T) {
				ctx := context.Background()
				topicURL := uniqueTopicURL(system)
				subURL := uniqueSubURL(system)

				topic, err := pubsub.OpenTopic(ctx, topicURL)
				require.NoError(t, err)

				sub, err := pubsub.OpenSubscription(ctx, subURL)
				require.NoError(t, err)

				// For rabbit/kafka: send a message to ensure the exchange/topic is created
				err = topic.Send(ctx, &pubsub.Message{Body: []byte("init")})
				require.NoError(t, err)

				rctx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()
				msg, err := sub.Receive(rctx)
				require.NoError(t, err)
				msg.Ack()

				err = topic.Shutdown(ctx)
				require.NoError(t, err)

				err = sub.Shutdown(ctx)
				require.NoError(t, err)
			})

			t.Run("SendAndReceive", func(t *testing.T) {
				ctx := context.Background()
				topicURL := uniqueTopicURL(system)
				subURL := uniqueSubURL(system)

				topic, err := pubsub.OpenTopic(ctx, topicURL)
				require.NoError(t, err)
				defer topic.Shutdown(ctx)

				sub, err := pubsub.OpenSubscription(ctx, subURL)
				require.NoError(t, err)
				defer sub.Shutdown(ctx)

				body := []byte(`{"test": "send-and-receive"}`)
				err = topic.Send(ctx, &pubsub.Message{Body: body})
				require.NoError(t, err)

				rctx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()

				msg, err := sub.Receive(rctx)
				require.NoError(t, err)
				assert.Equal(t, body, msg.Body)
				msg.Ack()
			})

			t.Run("MultipleMessages", func(t *testing.T) {
				ctx := context.Background()
				topicURL := uniqueTopicURL(system)
				subURL := uniqueSubURL(system)

				topic, err := pubsub.OpenTopic(ctx, topicURL)
				require.NoError(t, err)
				defer topic.Shutdown(ctx)

				sub, err := pubsub.OpenSubscription(ctx, subURL)
				require.NoError(t, err)
				defer sub.Shutdown(ctx)

				messages := []string{"msg1", "msg2", "msg3"}
				for _, m := range messages {
					err := topic.Send(ctx, &pubsub.Message{Body: []byte(m)})
					require.NoError(t, err)
				}

				received := make(map[string]bool)
				for range messages {
					rctx, cancel := context.WithTimeout(ctx, 10*time.Second)
					msg, err := sub.Receive(rctx)
					cancel()
					require.NoError(t, err)
					received[string(msg.Body)] = true
					msg.Ack()
				}

				for _, m := range messages {
					assert.True(t, received[m], "expected to receive message %q", m)
				}
			})
		})
	}
}
