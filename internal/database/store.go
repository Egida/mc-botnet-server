package database

import (
	"context"
	"github.com/mc-botnet/mc-botnet-server/internal/database/models"
	"github.com/mc-botnet/mc-botnet-server/internal/model"
	"github.com/stephenafamo/bob"
)

type Store interface {
	CreateUser(ctx context.Context, u *model.User) (int, error)
	GetUser(ctx context.Context, id int) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
}

type SQLStore struct {
	db bob.DB
}

func NewSQLStore(db bob.DB) *SQLStore {
	return &SQLStore{db}
}

func (s *SQLStore) CreateUser(ctx context.Context, u *model.User) (int, error) {
	id, err := models.Users.Insert(&models.UserSetter{
		Username: &u.Username,
		Password: &u.Password,
	}).Exec(ctx, s.db)
	return int(id), err
}

func (s *SQLStore) GetUser(ctx context.Context, id int) (*model.User, error) {
	u, err := models.FindUser(ctx, s.db, int32(id))
	if err != nil {
		return nil, err
	}
	return &model.User{
		ID:       int(u.ID),
		Username: u.Username,
		Password: u.Password,
	}, nil
}

func (s *SQLStore) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	u, err := models.Users.Query(models.SelectWhere.Users.Username.EQ(username)).One(ctx, s.db)
	if err != nil {
		return nil, err
	}
	return &model.User{
		ID:       int(u.ID),
		Username: u.Username,
		Password: u.Password,
	}, nil
}
