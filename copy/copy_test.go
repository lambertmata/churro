package copy

import (
	"reflect"
	"testing"
	"time"
)

type DomainUser struct {
	ID        string
	Name      string
	Enabled   bool
	CreatedAt *time.Time
	UpdatedAt *time.Time
	Tags      []string
	Role      Role
	Emails    []*string
}

type DBUser struct {
	ID        string
	Name      string
	Enabled   bool
	CreatedAt *time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
	Tags      []string
	Role      Role
	Emails    []*string
}

type Role struct {
	ID   string
	Name string
}

func CompareUser(t *testing.T, dbUser DBUser, domainUser DomainUser) {
	// Verify that each field has been copied correctly
	if domainUser.ID != dbUser.ID {
		t.Errorf("expected ID %v, got %v", dbUser.ID, domainUser.ID)
	}
	if domainUser.Name != dbUser.Name {
		t.Errorf("expected Name %v, got %v", dbUser.Name, domainUser.Name)
	}
	if domainUser.Enabled != dbUser.Enabled {
		t.Errorf("expected Enabled %v, got %v", dbUser.Enabled, domainUser.Enabled)
	}
	if !reflect.DeepEqual(domainUser.Tags, dbUser.Tags) {
		t.Errorf("expected Tags %v, got %v", dbUser.Tags, domainUser.Tags)
	}
	if *domainUser.CreatedAt != *dbUser.CreatedAt {
		t.Errorf("expected CreatedAt %v, got %v", dbUser.CreatedAt, domainUser.CreatedAt)
	}
	if *domainUser.UpdatedAt != *dbUser.UpdatedAt {
		t.Errorf("expected UpdatedAt %v, got %v", dbUser.UpdatedAt, domainUser.UpdatedAt)
	}
	if domainUser.Role != dbUser.Role {
		t.Errorf("expected Role %v, got %v", dbUser.Role, domainUser.Role)
	}
	if len(domainUser.Emails) != len(dbUser.Emails) {
		t.Errorf("expected Emails %v, got %v", len(dbUser.Emails), len(dbUser.Emails))
	}
	for i := range domainUser.Emails {
		if dbUser.Emails[i] == nil && domainUser.Emails[i] != nil {
			t.Errorf("expected Emails to be nil %v, got %v", dbUser.Emails[i], dbUser.Emails[i])
		} else if dbUser.Emails[i] != nil && domainUser.Emails[i] != nil && *domainUser.Emails[i] != *dbUser.Emails[i] {
			t.Errorf("expected Emails to be %v, got %v", *dbUser.Emails[i], *domainUser.Emails[i])
		}
	}
}

func TestAsSimple(t *testing.T) {

	createdAt := time.Now()
	updatedAt := time.Now()

	dbUser := DBUser{
		ID:        "1",
		Name:      "carlo",
		Enabled:   true,
		CreatedAt: &createdAt,
		UpdatedAt: &updatedAt,
		DeletedAt: nil,
		Tags:      []string{"support"},
		Role:      Role{ID: "1", Name: "Support"},
		Emails:    []*string{nil},
	}

	var domainUser DomainUser

	As(dbUser, &domainUser)

	CompareUser(t, dbUser, domainUser)
}

func TestCopySlice(t *testing.T) {

	createdAt := time.Now()
	updatedAt := time.Now()

	str := "example@email.com"

	dbUsers := []DBUser{
		{
			ID:        "1",
			Name:      "carlo",
			Enabled:   true,
			CreatedAt: &createdAt,
			UpdatedAt: &updatedAt,
			DeletedAt: nil,
			Tags:      []string{"support"},
			Role:      Role{ID: "1", Name: "Support"},
			Emails:    []*string{nil, nil, &str},
		},
	}

	var domainUsers []DomainUser

	As(dbUsers, &domainUsers)

	if len(domainUsers) != len(dbUsers) {
		t.Errorf("expected %d users, got %d", len(dbUsers), len(domainUsers))
	}

	for i := 0; i < len(dbUsers); i++ {
		CompareUser(t, dbUsers[i], domainUsers[i])
	}

}
