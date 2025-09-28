package asynq_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xescugc/qid/qid/queue"
	"github.com/xescugc/qid/qid/queue/providers/asynq"
)

func Test_IsQID_Service(t *testing.T) {
	assert.Implements(t, (*queue.Queue)(nil), new(asynq.Asynq))
}
