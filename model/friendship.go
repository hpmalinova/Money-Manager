package model

type Friendship struct {
	UserOne    int    `json:"userOne" validate:"required, numeric, gte=0"`
	UserTwo    int    `json:"userTwo" validate:"required, numeric, gte=0"`
	Status     string `json:"status"`
	ActionUser int    `json:"actionUser" validate:"required, numeric, gte=0"`
}

type Friends struct {
	Usernames []string
}

type GetFriends struct {
	Friends        Friends
	PendingFriends Friends
}
