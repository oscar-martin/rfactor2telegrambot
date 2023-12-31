package pubsub

import (
	"github.com/oscar-martin/rfactor2telegrambot/pkg/model"
	"github.com/oscar-martin/rfactor2telegrambot/pkg/resources"
)

const (
	PubSubSessionInfoPreffix         = "sessionInfo-"
	PubSubDriversSessionPreffix      = "driversSession-"
	PubSubStintDataPreffix           = "stintData-"
	PubSubThumbnailPreffix           = "thumbnail_"
	PubSubSessionStartedPreffix      = "sessionStarted_"
	PubSubFirstDriverEnteredPreffix  = "firstDriverEntered_"
	PubSubSessionStoppedPreffix      = "sessionStopped_"
	PubSubSelectedSessionDataPreffix = "selectedSessionData_"
	PubSubCarsPositionPreffix        = "carsPosition_"
)

var (
	LiveSessionInfoDataPubSub = NewPubSub[model.LiveSessionInfoData]()
	LiveStandingDataPubSub    = NewPubSub[model.LiveStandingData]()
	LiveStandingHistoryPubSub = NewPubSub[model.LiveStandingHistoryData]()
	TrackThumbnailPubSub      = NewPubSub[resources.Resource]()
	SessionStartedPubSub      = NewPubSub[model.ServerStarted]()
	SessionStoppedPubSub      = NewPubSub[string]()
	FirstDriverEnteredPubSub  = NewPubSub[model.ServerStarted]()
	SelectedSessionDataPubSub = NewPubSub[model.SelectedSessionData]()
	CarsPositionPubSub        = NewPubSub[[]model.CarPosition]()
)
