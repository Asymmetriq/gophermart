package model

import validation "github.com/go-ozzo/ozzo-validation/v4"

func (u *User) Validate() error {
	return validation.ValidateStruct(u,
		validation.Field(&u.Login),
		validation.Field(&u.Password, validation.Required),
	)
}
