package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"github.com/charmbracelet/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/knadh/koanf/v2"
	"github.com/mc-botnet/mc-botnet-server/internal/database"
	"github.com/mc-botnet/mc-botnet-server/internal/logger"
	"github.com/mc-botnet/mc-botnet-server/internal/model"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strconv"
	"strings"
)

var ErrUnauthorized = errors.New("auth: unauthorized")

// UserID panics if r doesn't have the user ID. Use only in [Service.Middleware]-protected handlers.
func UserID(r *http.Request) int {
	return r.Context().Value("userID").(int)
}

type Service struct {
	l     *log.Logger
	store database.Store

	secret []byte
}

func NewService(conf *koanf.Koanf, store database.Store) (*Service, error) {
	b64 := conf.Bool("jwt.base64")

	var secret []byte
	if b64 {
		var err error
		secret, err = base64.StdEncoding.DecodeString(conf.MustString("jwt.secret"))
		if err != nil {
			return nil, err
		}
	} else {
		secret = []byte(conf.MustString("jwt.secret"))
	}

	return &Service{
		l:      logger.NewLogger("auth", log.InfoLevel),
		store:  store,
		secret: secret,
	}, nil
}

func (s *Service) SignUp(ctx context.Context, req *model.SignUp) (string, error) {
	exists, err := s.store.UserExistsByUsername(ctx, req.Username)
	if err != nil {
		return "", err
	}
	if exists {
		return "", database.ErrConflict
	}

	hashed, err := hash(req.Password)
	if err != nil {
		return "", err
	}

	id, err := s.store.CreateUser(ctx, &model.User{
		Username: req.Username,
		Password: hashed,
	})
	if err != nil {
		return "", err
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject: strconv.Itoa(id),
	})
	return t.SignedString(s.secret)
}

func (s *Service) SignIn(ctx context.Context, req *model.SignIn) (string, error) {
	u, err := s.store.FindUserByUsername(ctx, req.Username)
	if err != nil {
		return "", err
	}
	if !verify(req.Password, u.Password) {
		return "", ErrUnauthorized
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject: strconv.Itoa(u.ID),
	})
	return token.SignedString(s.secret)
}

func (s *Service) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if tokenString == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(
			tokenString,
			func(token *jwt.Token) (interface{}, error) {
				return s.secret, nil
			},
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		subString, err := token.Claims.GetSubject()
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		sub, err := strconv.Atoi(subString)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), "userID", sub))
		next(w, r)
	}
}

func hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func verify(password, hash string) bool {
	b, err := base64.StdEncoding.DecodeString(hash)
	if err != nil {
		return false
	}
	return bcrypt.CompareHashAndPassword(b, []byte(password)) == nil
}
