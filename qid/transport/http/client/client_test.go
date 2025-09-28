package client_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/transport/http/client"
)

func Test_IsQID_Service(t *testing.T) {
	assert.Implements(t, (*qid.Service)(nil), new(client.Client))
}
