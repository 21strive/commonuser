package jwt_impl

import "github.com/golang-jwt/jwt/v5"

type UserClaims struct {
	UUID      string `json:"uuid"` // user uuid
	RandId    string `json:"randId"`
	Name      string `json:"name"`
	Username  string `json:"username,omitempty"`
	Email     string `json:"email,omitempty"`
	Avatar    string `json:"avatar,omitempty"`
	Verified  bool   `json:"verified"`
	SessionID string `json:"sessionid"`
	jwt.RegisteredClaims
}
