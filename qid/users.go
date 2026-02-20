package qid

import (
	"context"
	"fmt"

	"github.com/xescugc/qid/qid/user"
	"github.com/xescugc/qid/qid/utils"
)

func (q *Qid) UserLogin(ctx context.Context, un, pass string) (*user.User, error) {
	if !utils.ValidateCanonical(un) {
		return nil, fmt.Errorf("invalid Username format %q", un)
	}
	u, err := q.Users.Find(ctx, un)
	if err != nil {
		return nil, fmt.Errorf("failed to Find User: %w", err)
	}

	ok := utils.CheckPasswordHash(pass, u.Password)
	if !ok {
		return nil, fmt.Errorf("Username or Password is wrong")
	}

	return u, nil
}

func (q *Qid) GetUser(ctx context.Context, un string) (*user.WithMemberships, error) {
	if !utils.ValidateCanonical(un) {
		return nil, fmt.Errorf("invalid Username format %q", un)
	}

	um, err := q.Users.FindWithMemberships(ctx, un)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return um, nil
}

func (q *Qid) CreateUser(ctx context.Context, u user.User, isHash bool) (*user.User, error) {
	if !utils.ValidateCanonical(u.Username) {
		return nil, fmt.Errorf("invalid Username format %q", u.Username)
	} else if u.Password == "" {
		return nil, fmt.Errorf("invalid empty Password")
	}

	if !isHash {
		hash, err := utils.HashPassword(u.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash Passowrd: %w", err)
		}
		u.Password = hash
	}

	id, err := q.Users.Create(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("failed to Create User: %w", err)
	}
	u.ID = id

	return &u, nil
}

func (q *Qid) ListUsers(ctx context.Context) ([]*user.User, error) {
	us, err := q.Users.Filter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to Find User: %w", err)
	}

	return us, nil
}
