package model

type Group struct {
	ID             int    `json:"id" validate:"numeric,gte=0"`
	Name           string `json:"name" validate:"required,min=1,max=32"`
	ParticipantIDs []int  `json:"participantIDs"` //todo check if okay
}

type CreateGroup struct {
	Name         string   `json:"name"`
	Participants []string `json:"participants"`
}
