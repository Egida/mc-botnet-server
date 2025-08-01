package model

type SignUp struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type SignIn struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
}
