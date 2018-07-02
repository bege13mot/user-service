package main

import (
	"errors"
	"fmt"
	"log"

	pb "github.com/bege13mot/user-service/proto/auth"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"
)

type service struct {
	repo         repository
	tokenService authable
}

func (srv *service) Get(ctx context.Context, req *pb.User) (*pb.Response, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("Not valid request")
	}
	user, err := srv.repo.get(req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.Response{User: user}, nil
}

func (srv *service) GetAll(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	users, err := srv.repo.getAll()
	if err != nil {
		return nil, err
	}
	return &pb.Response{Users: users}, nil
}

func (srv *service) Auth(ctx context.Context, req *pb.User) (*pb.Token, error) {
	log.Println("Auth with email: ", req.Email)
	user, err := srv.repo.getByEmail(req.Email)
	log.Println("Auth as: ", user)
	if err != nil {
		return nil, err
	}
	// Compares our given password against the hashed password
	// stored in the database
	if passErr := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, passErr
	}
	token, err := srv.tokenService.encode(user)
	if err != nil {
		return nil, err
	}
	return &pb.Token{Token: token}, nil
}

func (srv *service) Create(ctx context.Context, req *pb.User) (*pb.Response, error) {
	log.Println("Create user: ", req.Name, req.Email)

	// Generates a hashed version of our password
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	req.Password = string(hashedPass)
	if err := srv.repo.create(req); err != nil {
		return nil, err
	}
	return &pb.Response{User: req}, nil
}

func (srv *service) ValidateToken(ctx context.Context, req *pb.Token) (*pb.Token, error) {
	log.Println("Validate token: ", req.Token)
	// Decode token
	claims, err := srv.tokenService.decode(req.Token)
	if err != nil {
		log.Println("Not Valid token: ", err)
		return nil, err
	}
	if claims.User.Id == "" {
		return nil, errors.New("invalid user")
	}
	return &pb.Token{Valid: true}, nil
}

// Check implements Consul health checking
func (srv *service) Check(ctx context.Context, in *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	if err := srv.repo.ping(); err != nil {
		return &pb.HealthCheckResponse{
			Status: pb.HealthCheckResponse_NOT_SERVING,
		}, err
	}

	return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_SERVING}, nil
}
