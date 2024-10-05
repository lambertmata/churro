package churro

import (
	"github.com/lambertmata/churro/utils"
	"testing"
)

func TestCopy(t *testing.T) {

	type Subscription struct {
		Started bool
	}

	type User struct {
		ID           string       `json:"id"`
		Name         string       `json:"name"`
		CreatedAt    string       `json:"created_at"`
		Subscription Subscription `json:"subscription"`
	}

	type UserResponse struct {
		ID           string       `json:"id"`
		Name         string       `json:"name"`
		Subscription Subscription `json:"subscription"`
	}

	user := User{
		ID:   "1",
		Name: "John Doe",
		Subscription: Subscription{
			Started: true,
		},
	}

	userRes := utils.As(user, UserResponse{})

	if user.ID != userRes.ID {
		t.Errorf(`expected user res %s, got %s`, user.ID, userRes.ID)
	}
	if user.Name != userRes.Name {
		t.Errorf(`expected user res %s, got %s`, user.Name, userRes.Name)
	}
	if user.Subscription.Started != userRes.Subscription.Started {
		t.Errorf(`expected user res %t, got %t`, user.Subscription.Started, userRes.Subscription.Started)
	}

}
