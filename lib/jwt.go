package lib

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lefalya/commonuser/definition"
)

type JWTHandler struct {
	jwtSecret        string
	jwtTokenIssuer   string
	jwtTokenLifeSpan int
}

func (jh *JWTHandler) ParseJWT(jwtToken string, expectedStruct interface{ jwt.Claims }) (interface{ jwt.Claims }, error) {
	claimedToken, err := jwt.ParseWithClaims(jwtToken, expectedStruct, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jh.jwtSecret), nil
	})
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	if claims, ok := claimedToken.Claims.(interface{ jwt.Claims }); ok && claimedToken.Valid {
		return claims, nil
	} else {
		return nil, definition.Unauthorized
	}
}

func (jh *JWTHandler) ParseAccessToken(jwtToken string) (*UserClaims, error) {
	userClaims, err := jh.ParseJWT(jwtToken, &UserClaims{})
	if err != nil {
		return nil, err
	}

	return userClaims.(*UserClaims), nil
}

func (jh *JWTHandler) ParseRefreshToken(jwtToken string) (*RefreshTokenClaims, error) {
	refreshClaims, err := jh.ParseJWT(jwtToken, &RefreshTokenClaims{})
	if err != nil {
		return nil, err
	}

	return refreshClaims.(*RefreshTokenClaims), nil
}

func NewJWTHandler(jwtSecret string, jwtTokenIssuer string, jwtTokenLifeSpan int) *JWTHandler {
	return &JWTHandler{
		jwtSecret:        jwtSecret,
		jwtTokenIssuer:   jwtTokenIssuer,
		jwtTokenLifeSpan: jwtTokenLifeSpan,
	}
}
