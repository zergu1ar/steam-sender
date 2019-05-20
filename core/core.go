package core

import (
	"github.com/zergu1ar/steam"
	"time"
)

type (
	Config struct {
		Log struct {
			Way         string
			Destination string
		}
		Accounts []*Account
		Main     struct {
			Partner uint32
			Token   string
		}
		MainAccount struct {
			Username     string
			Password     string
			SharedSecret string
		}
	}
	Account struct {
		Username       string
		Password       string
		SharedSecret   string
		IdentitySecret string
	}
	Session struct {
		Session        *steam.Session
		TimeDiff       time.Duration
		WebAPIKey      string
		SecretIdentity string
	}
)

func (c *Config) GetLogWay() string {
	return c.Log.Way
}
func (c *Config) GetLogDestination() string {
	return c.Log.Destination
}
func (c *Config) GetConnString() string {
	return ""
}

func (c *Config) GetLogApplicationName() string {
	return ""
}
