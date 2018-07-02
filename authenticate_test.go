package main

import (
	"testing"

	pb "github.com/testProject/user-service/proto/auth"
)

func TestCases(t *testing.T) {

	srv := tokenService{}

	user := pb.User{
		Name: "testUser",
	}

	validToken, err := srv.encode(&user)
	if err != nil {
		t.Errorf("Encode error, %v", err)
	}

	//Decode checking
	_, err = srv.decode(validToken)
	if err != nil {
		t.Errorf("Decode error, %v", err)
	}

	_, err = srv.decode("123")
	if err == nil {
		t.Error("Invalid token shouldn't be allowed")
	}
}
