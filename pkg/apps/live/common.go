package live

import "github.com/nicksnyder/go-i18n/v2/i18n"

const (
	symbolTimes    = "⏱"
	symbolSectors  = "🔂"
	symbolCompound = "🛞"
	symbolLaps     = "🏁"
	symbolTeam     = "🏎️"
	symbolDriver   = "👐"
	symbolUpdate   = "🔄"
	symbolDiff     = "⏲️"
	symbolOptimum  = "🚀"
	symbolPhoto    = "📸"
)

func getInlineKeyboardTimes(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.time",
			Other: "Time",
		},
	})

	return msg
}

func getInlineKeyboardBestLap(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.bestLap",
			Other: "Best Lap",
		},
	})
	return msg
}

func getInlineKeyboardBestLapSectors(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.sectorsBL",
			Other: "Sectors BL.",
		},
	})
	return msg
}

func getInlineKeyboardLastLap(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.lastLap",
			Other: "Last Lap",
		},
	})
	return msg
}

func getInlineKeyboardLastLapSectors(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.sectorsLL",
			Other: "Sectors LL.",
		},
	})
	return msg
}

func getInlineKeyboardOptimumLap(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.optimal",
			Other: "Optimal",
		},
	})
	return msg
}

func getInlineKeyboardOptimumLapSectors(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.sectorsO",
			Other: "Sectors O.",
		},
	})
	return msg
}

func getInlineKeyboardSectors(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.sectors",
			Other: "Sectors",
		},
	})
	return msg
}

func getInlineKeyboardLiveMap(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.map",
			Other: "Map 🗺️",
		},
	})
	return msg
}

func getInlineKeyboardStatus(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.status",
			Other: "Status 🏎️",
		},
	})
	return msg
}

func getInlineKeyboardInfo(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.info",
			Other: "Info 👐",
		},
	})
	return msg
}

func getInlineKeyboardDiff(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.gap",
			Other: "Gap ⏳",
		},
	})
	return msg
}

func getInlineKeyboardCompound(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.tyres",
			Other: "Tyres",
		},
	})
	return msg
}

func getInlineKeyboardLaps(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.laps",
			Other: "Laps",
		},
	})
	return msg
}

func getInlineKeyboardTeam(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.cars",
			Other: "Cars",
		},
	})
	return msg
}

func getInlineKeyboardDriver(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.drivers",
			Other: "Drivers",
		},
	})
	return msg
}

func getInlineKeyboardUpdate(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.update",
			Other: "Update",
		},
	})
	return msg
}

func getInlineKeyboardMaxSpeed(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.topSpeed",
			Other: "Top Speed",
		},
	})
	return msg
}

func getInlineKeyboardCar(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.car",
			Other: "Car",
		},
	})
	return msg
}

func getLapHeader(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.headerLap",
			Other: "LAP",
		},
	})
	return msg
}

func getTopSpeedHeader(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.headerTopSpeed",
			Other: "Top Speed",
		},
	})
	return msg
}

func getDriverHeader(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.headerDriver",
			Other: "DRI",
		},
	})
	return msg
}

func getNameHeader(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.headerName",
			Other: "Name",
		},
	})
	return msg
}

func getSectorsHeader(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.headerSectors",
			Other: "Sectors",
		},
	})
	return msg
}

func getLastHeader(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.headerLast",
			Other: "Last",
		},
	})
	return msg
}

func getBestHeader(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.headerBest",
			Other: "Best",
		},
	})
	return msg
}

func getOptimalHeader(loc *i18n.Localizer) string {
	msg := loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "apps.headerOptimal",
			Other: "Optimal",
		},
	})
	return msg
}
