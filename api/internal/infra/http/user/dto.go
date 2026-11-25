package userhttp

import "github.com/Lmare/lightning-playground/api/internal/business/user/domain"

type userDTO struct {
	ID     string `json:"id"`
	Nom    string `json:"nom"`
	Prenom string `json:"prenom"`
	Age    int    `json:"age"`
	Email  string `json:"email"`
}

func toUserDTOs(users []domain.UserModel) []userDTO {
	dtos := make([]userDTO, 0, len(users))
	for _, u := range users {
		dtos = append(dtos, userDTO{ID: u.ID, Nom: u.Nom, Prenom: u.Prenom, Age: u.Age, Email: u.Email})
	}
	return dtos
}
