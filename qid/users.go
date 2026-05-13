package qid

import (
	"context"
	"fmt"

	"github.com/golang-jwt/jwt/v5"

	"github.com/xescugc/qid/qid/user"
	"github.com/xescugc/qid/qid/utils"
)

func (q *Qid) UserLogin(ctx context.Context, un, pass string) (*user.WithMemberships, string, error) {
	if !utils.ValidateCanonical(un) {
		return nil, "", fmt.Errorf("invalid Username format %q", un)
	}
	um, err := q.Users.FindWithMemberships(ctx, un)
	if err != nil {
		return nil, "", fmt.Errorf("failed to Find User: %w", err)
	}

	ok := utils.CheckPasswordHash(pass, um.Password)
	if !ok {
		return nil, "", fmt.Errorf("Username or Password is wrong")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": um,
	})
	tokenString, err := token.SignedString(q.JWTSecret)
	if err != nil {
		return nil, "", fmt.Errorf("failed to Find User: %w", err)
	}

	return um, tokenString, nil
}

func (q *Qid) RefreshToken(ctx context.Context, un string) (*user.WithMemberships, string, error) {
	if !utils.ValidateCanonical(un) {
		return nil, "", fmt.Errorf("invalid Username format %q", un)
	}
	um, err := q.Users.FindWithMemberships(ctx, un)
	if err != nil {
		return nil, "", fmt.Errorf("failed to Find User: %w", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": um,
	})
	tokenString, err := token.SignedString(q.JWTSecret)
	if err != nil {
		return nil, "", fmt.Errorf("failed to sign token: %w", err)
	}

	return um, tokenString, nil
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
