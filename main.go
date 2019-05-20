package main

import (
	"context"
	"flag"
	"github.com/sirupsen/logrus"
	"github.com/zergu1ar/gaarx"
	"github.com/zergu1ar/logrus-filename"
	"github.com/zergu1ar/steam"
	"net/http"
	"os"
	"os/signal"
	cs "steam-confirm/confirm"
	"steam-confirm/core"
	"syscall"
	"time"
)

var (
	configFile   = flag.String("config", "config.toml", "Project config")
	configSource = flag.String("configType", "toml", "Project config type")
)

const ExitCode = 0

func main() {
	flag.Parse()
	var stop = make(chan os.Signal)
	ctx, finish := context.WithCancel(context.Background())
	var application = &gaarx.App{
		Finish: finish,
	}
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		time.Sleep(time.Second)
		finish()
	}()
	log := logrus.New()
	log.Formatter = &logrus.TextFormatter{DisableColors: true}
	filenameHook := filename.NewHook()
	filenameHook.Field = "line"
	log.AddHook(filenameHook)
	application.Start(
		gaarx.WithConfigFile(*configSource, *configFile, &core.Config{}),
		gaarx.WithContext(ctx),
		gaarx.WithStorage("sessions"),
		gaarx.WithLogger(log),
		gaarx.WithMethods(gaarx.Method{
			Name: "GetSessions",
			Func: func(app *gaarx.App) error {
				for _, account := range app.Config().(*core.Config).Accounts {
					timeTip, err := steam.GetTimeTip()
					if err != nil {
						app.GetLog().Fatal(err)
					}
					client := http.Client{
						//CheckRedirect: func(req *http.Request, via []*http.Request) error {
						//	return http.ErrUseLastResponse
						//},
					}
					steamSession := steam.NewSession(&client, "")
					err = steamSession.Login(
						account.Username,
						account.Password,
						account.SharedSecret,
						time.Duration(timeTip.Time-time.Now().Unix()),
					)
					if err != nil {
						app.GetLog().Error(account.Username, err)
						panic(err)
					}
					var apiKey string
					apiKey, err = steamSession.GetWebAPIKey()
					if err != nil {
						app.GetLog().Error(account.Username, err)
					}
					sess := core.Session{
						Session:        steamSession,
						TimeDiff:       time.Duration(timeTip.Time),
						SecretIdentity: account.IdentitySecret,
						WebAPIKey:      apiKey,
					}
					_ = app.Storage().Set("sessions", account.Username, &sess)
					time.Sleep(time.Second * 1)
				}
				return nil
			},
		}),
		gaarx.WithServices(cs.Create(ctx)),
	)
	_ = application.CallMethod("GetSessions")
	application.Work()
	<-ctx.Done()
	os.Exit(ExitCode)
}
