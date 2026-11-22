package userhttp

import "github.com/Lmare/lightning-playground/backend/internal/business/user/domain"

type personView struct {
	Index  int
	Person domain.UserModel
}

func viewObject(persons []domain.UserModel) []personView {

	var viewData []personView
	for i, p := range persons {
		viewData = append(viewData, personView{
			Index:  i + 1,
			Person: p,
		})
	}
	return viewData

}
