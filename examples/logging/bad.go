package logging

import (
	"fmt"
	"log"
)

type UserService struct{}

// Using fmt.Println instead of a structured logger.
func (s *UserService) CreateUser(name string) error {
	fmt.Println("creating user:", name)
	fmt.Printf("user name: %s\n", name)
	return nil
}

// Using stdlib log instead of golog.
func (s *UserService) DeleteUser(id int) error {
	log.Printf("deleting user %d", id)
	log.Println("user deleted")
	return nil
}

// log.Fatal is also banned.
func StartServer(addr string) {
	log.Fatal("failed to start server on", addr)
}
