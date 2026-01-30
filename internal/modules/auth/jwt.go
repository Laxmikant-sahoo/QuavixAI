package auth

import jwt "github.com/golang-jwt/jwt/v5"

type JWTService interface {
	Generate(uid, role string) (string, error)
}

type jwtService struct {
	secret []byte
}

func NewJWT(secret string) JWTService {
	return &jwtService{secret: []byte(secret)}
}

func (j *jwtService) Generate(uid, role string) (string, error) {
	claims := jwt.MapClaims{
		"uid":  uid,
		"role": role,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(j.secret)
}
