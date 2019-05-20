package confirm

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/zergu1ar/gaarx"
	"github.com/zergu1ar/steam"
	"log"
	"net/http"
	"steam-confirm/core"
	"time"
)

type (
	confirmService struct {
		log  func() *logrus.Entry
		app  *gaarx.App
		ctx  context.Context
		done chan bool
	}
)

func Create(ctx context.Context) *confirmService {
	ms := confirmService{
		ctx:  ctx,
		done: make(chan bool, 1),
	}
	return &ms
}

func (cs *confirmService) Start(app *gaarx.App) error {
	cs.app = app
	cs.log = func() *logrus.Entry {
		return app.GetLog().WithField("service", "CS")
	}
	er := cs.app.Storage().Range("sessions", func(key, val interface{}) bool {
		session := val.(*core.Session)
		cs.handleInventory(session)
		time.Sleep(time.Second * 3)
		return true
	})
	if er != nil {
		cs.log().Warning(er)
	}
	cs.handleReceiveTrades()
	cs.app.Stop()
	return nil
}

func (cs *confirmService) GetName() string {
	return "CreateOffers service"
}

func (cs *confirmService) Stop() {
	cs.log().Info("shutdown")
}

func (cs *confirmService) handleInventory(session *core.Session) {
	sid := session.Session.GetSteamID()
	var item2Send []*steam.EconItem
	apps, err := session.Session.GetInventoryAppStats(sid)
	if err != nil {
		cs.log().Error(err)
		return
	}
	for _, v := range apps {
		for _, ctx := range v.Contexts {
			invItem, err := session.Session.GetInventory(sid, v.AppID, ctx.ID, true)
			if err != nil {
				cs.log().Error(err)
			}

			for _, item := range invItem {
				sendItem := steam.EconItem{
					AssetID:   item.AssetID,
					AppID:     item.AppID,
					ContextID: item.ContextID,
					Amount:    uint16(item.Amount),
				}
				item2Send = append(item2Send, &sendItem)
				log.Printf("Item: %s = %d\n", item.Desc.MarketHashName, item.AssetID)
			}
		}
	}
	if len(item2Send) > 0 {
		offer := steam.TradeOffer{
			SendItems: item2Send,
		}
		var SID steam.SteamID
		SID.ParseDefaults(cs.app.Config().(*core.Config).Main.Partner)
		err := session.Session.SendTradeOffer(&offer, SID, cs.app.Config().(*core.Config).Main.Token)
		if err != nil {
			cs.log().Error(err)
			return
		}
		if offer.ID > 0 {
			cs.log().WithField("offerID", offer.ID).Info("Offer sent")
			cs.actionConfirm(session)
		}
	}
}

func (cs *confirmService) actionConfirm(session *core.Session) {
	confirmations, err := session.Session.GetConfirmations(
		session.SecretIdentity,
		time.Now().Add(session.TimeDiff).Unix(),
	)
	if err != nil {
		cs.log().Error(err)
	}
	time.Sleep(time.Second * 3)

	for i := range confirmations {
		c := confirmations[i]
		err = session.Session.AnswerConfirmation(
			c,
			session.SecretIdentity,
			"allow",
			time.Now().Add(session.TimeDiff).Unix(),
		)
		if err != nil {
			cs.log().WithField("tradeID", c.OfferID).Error(err)
			continue
		}
		cs.log().WithField("tradeID", c.OfferID).Info("Confirm action")
		time.Sleep(time.Second)
	}
}

func (cs *confirmService) handleReceiveTrades() {
	timeTip, err := steam.GetTimeTip()
	if err != nil {
		cs.log().Error(err)
	}
	client := http.Client{
		//CheckRedirect: func(req *http.Request, via []*http.Request) error {
		//	return http.ErrUseLastResponse
		//},
	}
	steamSession := steam.NewSession(&client, "")
	err = steamSession.Login(
		cs.app.Config().(*core.Config).MainAccount.Username,
		cs.app.Config().(*core.Config).MainAccount.Password,
		cs.app.Config().(*core.Config).MainAccount.SharedSecret,
		time.Duration(timeTip.Time-time.Now().Unix()),
	)
	if err != nil {
		cs.log().Error(cs.app.Config().(*core.Config).MainAccount.Username, err)
		panic(err)
	}
	if _, err := steamSession.GetWebAPIKey(); err != nil {
		cs.log().Error(err)
		return
	}

	trades, err := steamSession.GetTradeOffers(
		steam.TradeFilterRecvOffers,
		time.Now(),
	)
	if err != nil {
		cs.log().Error(err)
	}
	time.Sleep(time.Second * 3)
	for _, offer := range trades.ReceivedOffers {
		var sid steam.SteamID
		sid.ParseDefaults(offer.Partner)
		if offer.State != steam.TradeStateActive || offer.ConfirmationMethod == steam.TradeConfirmationMobileApp {
			continue
		}
		if len(offer.SendItems) == 0 {
			cs.log().WithField("offerID", offer.ID).Info("gift")
			err := offer.Accept(steamSession)
			if err != nil {
				cs.log().WithField("offerID", offer.ID).Warning(err)
			}
			continue
		}
		cs.log().WithField("offerID", offer.ID).Warning("Fail to accept")
	}
}
