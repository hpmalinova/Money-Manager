package model

type Friendship struct {
	UserOne    int    `json:"userOne" validate:"required, numeric, gte=0"`
	UserTwo    int    `json:"userTwo" validate:"required, numeric, gte=0"`
	Status     string `json:"status"` //omitempty? todo
	ActionUser int    `json:"actionUser" validate:"required, numeric, gte=0"`
}

type AddFriend struct {
	FriendName   string `json:"friendName"`
	ActionUserID int    `json:"actionUserID"`
}
