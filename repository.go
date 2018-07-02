package main

import (
	pb "github.com/bege13mot/user-service/proto/auth"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

type myUser struct {
	*pb.User
}

func (model *myUser) BeforeCreate(scope *gorm.Scope) error {
	uuid, _ := uuid.NewV4()
	return scope.SetColumn("Id", uuid.String())
}

type repository interface {
	getAll() ([]*pb.User, error)
	get(id string) (*pb.User, error)
	create(user *pb.User) error
	getByEmail(email string) (*pb.User, error)
	ping() error
}

type userRepository struct {
	db *gorm.DB
}

func (repo *userRepository) getAll() ([]*pb.User, error) {
	var users []*pb.User
	if err := repo.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (repo *userRepository) get(id string) (*pb.User, error) {
	var user *pb.User
	user.Id = id
	if err := repo.db.First(&user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (repo *userRepository) getByEmail(email string) (*pb.User, error) {
	user := &pb.User{}
	if err := repo.db.Where("email = ?", email).
		First(&user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (repo *userRepository) create(user *pb.User) error {
	uuid, _ := uuid.NewV4()
	user.Id = uuid.String()

	if err := repo.db.Create(user).Error; err != nil {
		return err
	}
	return nil
}

func (repo *userRepository) ping() error {
	if err := repo.db.DB().Ping(); err != nil {
		return err
	}
	return nil
}
