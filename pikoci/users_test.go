package pikoci_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/pikoci/pikoci/user"
	"github.com/xescugc/pikoci/pikoci/utils"
	"go.uber.org/mock/gomock"
)

func TestCreateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	s.Users.EXPECT().Create(ctx, gomock.Any()).Return(uint32(1), nil)

	u, err := s.S.CreateUser(ctx, user.User{Username: "admin", Password: "secret"}, false)
	require.NoError(t, err)
	require.NotNil(t, u)
	assert.Equal(t, uint32(1), u.ID)
	assert.Equal(t, "admin", u.Username)
	// Password should be hashed
	assert.NotEqual(t, "secret", u.Password)
	assert.True(t, utils.CheckPasswordHash("secret", u.Password))
}

func TestCreateUser_Hashed(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	hash, _ := utils.HashPassword("secret")

	s.Users.EXPECT().Create(ctx, gomock.Any()).Return(uint32(2), nil)

	u, err := s.S.CreateUser(ctx, user.User{Username: "admin", Password: hash}, true)
	require.NoError(t, err)
	assert.Equal(t, hash, u.Password)
}

func TestCreateUser_InvalidUsername(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	_, err := s.S.CreateUser(ctx, user.User{Username: "INVALID USER", Password: "secret"}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid Username format")
}

func TestCreateUser_EmptyPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	_, err := s.S.CreateUser(ctx, user.User{Username: "admin", Password: ""}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid empty Password")
}

func TestGetUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	expected := &user.WithMemberships{
		User: user.User{ID: 1, Username: "admin"},
	}
	s.Users.EXPECT().FindWithMemberships(ctx, "admin").Return(expected, nil)

	u, err := s.S.GetUser(ctx, "admin")
	require.NoError(t, err)
	assert.Equal(t, expected, u)
}

func TestGetUser_InvalidUsername(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	_, err := s.S.GetUser(ctx, "INVALID USER")
	require.Error(t, err)
}

func TestListUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	expected := []*user.User{{ID: 1, Username: "admin"}, {ID: 2, Username: "pepito"}}
	s.Users.EXPECT().Filter(ctx).Return(expected, nil)

	us, err := s.S.ListUsers(ctx)
	require.NoError(t, err)
	assert.Equal(t, expected, us)
}

func TestUserLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	hash, _ := utils.HashPassword("secret")
	um := &user.WithMemberships{
		User: user.User{ID: 1, Username: "admin", Password: hash},
	}
	s.Users.EXPECT().FindWithMemberships(ctx, "admin").Return(um, nil)

	u, jwt, err := s.S.UserLogin(ctx, "admin", "secret")
	require.NoError(t, err)
	require.NotNil(t, u)
	assert.NotEmpty(t, jwt)
	assert.Equal(t, "admin", u.Username)
}

func TestUserLogin_WrongPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	s := newService(ctrl)
	ctx := context.TODO()

	hash, _ := utils.HashPassword("secret")
	um := &user.WithMemberships{
		User: user.User{ID: 1, Username: "admin", Password: hash},
	}
	s.Users.EXPECT().FindWithMemberships(ctx, "admin").Return(um, nil)

	_, _, err := s.S.UserLogin(ctx, "admin", "wrong")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wrong")
}
