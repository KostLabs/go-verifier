package interface_definition

// UserRepository interface defined in the same package as its implementation.
// It should live in the consumer package instead.
type UserRepository interface {
	FindByID(id int) (*User, error)
	Save(u *User) error
	Delete(id int) error
}

type User struct {
	ID   int
	Name string
}

// postgresUserRepository implements UserRepository in the same package —
// the interface should be defined where it's consumed, not here.
type postgresUserRepository struct {
	dsn string
}

func (r *postgresUserRepository) FindByID(id int) (*User, error) {
	return &User{ID: id}, nil
}

func (r *postgresUserRepository) Save(u *User) error {
	return nil
}

func (r *postgresUserRepository) Delete(id int) error {
	return nil
}
