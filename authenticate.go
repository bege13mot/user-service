package main

import (
	"log"
	"os"
	"time"

	pb "github.com/bege13mot/user-service/proto/auth"
	"github.com/dgrijalva/jwt-go"
)

// Define a secure key string used
// as a salt when hashing our tokens.
// Please make your own way more secure than this,
// use a randomly generated md5 hash or something.
var (
	key        []byte
	defaultKey = "mySuperSecretKey!123"
)

func init() {
	usersKey := os.Getenv("SECRET_KEY")

	if usersKey == "" {
		log.Println("Use default Secret Key")
		key = []byte(defaultKey)
	} else {
		key = []byte(usersKey)
	}
}

// CustomClaims is our custom metadata, which will be hashed
// and sent as the second segment in our JWT
type customClaims struct {
	User *pb.User
	jwt.StandardClaims
}

type authable interface {
	decode(token string) (*customClaims, error)
	encode(user *pb.User) (string, error)
}

type tokenService struct {
	repo repository
}

// Decode a token string into a token object
func (srv *tokenService) decode(tokenString string) (*customClaims, error) {

	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &customClaims{}, func(token *jwt.Token) (interface{}, error) {
		return key, nil
	})

	if err != nil {
		return nil, err
	}

	// Validate the token and return the custom claims
	if claims, ok := token.Claims.(*customClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, err
}

// Encode a claim into a JWT
func (srv *tokenService) encode(user *pb.User) (string, error) {

	expireToken := time.Now().Add(time.Hour * 72).Unix()

	// Create the Claims
	claims := customClaims{
		user,
		jwt.StandardClaims{
			ExpiresAt: expireToken,
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token and return
	return token.SignedString(key)
}
