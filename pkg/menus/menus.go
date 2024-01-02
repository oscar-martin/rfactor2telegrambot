package menus

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type Menuer interface {
	Menu() tgbotapi.ReplyKeyboardMarkup
}

type ApplicationMenu struct {
	Name       string
	From       string
	prevMenuer Menuer
	loc        *i18n.Localizer
}

func NewApplicationMenu(name, from string, prevMenuer Menuer, loc *i18n.Localizer) ApplicationMenu {
	return ApplicationMenu{
		Name:       name,
		From:       from,
		prevMenuer: prevMenuer,
		loc:        loc,
	}
}

func (am *ApplicationMenu) ButtonBackTo() string {
	buttonBackTo := am.loc.MustLocalize(&i18n.LocalizeConfig{
		// MessageID: "menus.backTo",
		DefaultMessage: &i18n.Message{
			ID:    "menus.backTo",
			Other: "Back to",
		},
	})

	return buttonBackTo + " " + am.From
}

func (am *ApplicationMenu) PrevMenu() tgbotapi.ReplyKeyboardMarkup {
	return am.prevMenuer.Menu()
}
