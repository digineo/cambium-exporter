package exporter

import (
	"strconv"
	"time"
)

type rebootResponse struct {
	Unixtime int    `json:"uTs"`
	Reason   string `json:"code"`
}

type radioResponse struct {
	ID           int    `json:"id"`
	Band         string `json:"band"` // enum: { "2.4GHz", "5GHz" }
	ChannelWidth string `json:"chWidth"`
	Channel      string `json:"channel"`
	MAC          string `json:"mac"`
	Power        int    `json:"pow"`
	Quality      int    `json:"rfqlt"` // percentage points
	RxAvg        int    `json:"rxAvg"` // in kbps
	TxAvg        int    `json:"txAvg"` // in kbps
}

// DeviceAPIResponse holds the data returned from the controller's
// device endpoint.
type DeviceAPIResponse struct {
	Model    string `json:"model"`
	MAC      string `json:"mac"`
	Serial   string `json:"sn"`
	SiteName string `json:"tid"`

	Config struct {
		Name string `json:"name"`
	} `json:"cfg"`

	System struct {
		Online   bool             `json:"online"`
		UpTime   int64            `json:"upTime"`
		DownTime int64            `json:"dnTime"`
		Reboots  []rebootResponse `json:"lastRbt"`
	} `json:"sys"`

	Management struct {
		FirmwareVersion string `json:"actSw"`
	} `json:"mgmt"`

	Radios []radioResponse `json:"radios"`
}

type Radio struct {
	Band         Band
	Channel      int // channel number
	ChannelWidth int // in MHz
	Power        int
	Quality      int // 0..100
	Rx           int // average rate in kbps
	Tx           int // average rate in kbps
}

func (radio radioResponse) Normalize() (r Radio) {
	r = Radio{
		Band:         BandUnknown,
		Channel:      -1,
		ChannelWidth: -1,
		Rx:           radio.RxAvg,
		Tx:           radio.TxAvg,
		Quality:      radio.Quality,
		Power:        radio.Power,
	}

	switch radio.Band {
	case "2.4GHz":
		r.Band = BandBGN
	case "5GHz":
		r.Band = BandAC
	}

	if ch, err := strconv.Atoi(radio.Channel); err == nil {
		r.Channel = ch
	}
	if chw, err := strconv.Atoi(radio.ChannelWidth); err == nil {
		r.ChannelWidth = chw
	}

	return
}

type Band string

const (
	BandAC      = Band("5")
	BandBGN     = Band("2.4")
	BandUnknown = Band("unknown")
)

// Device holds basic metrics for a single WiFi AP.
// It is constructed from a DeviceAPIResponse.
type Device struct {
	Model           string
	MAC             string
	Serial          string
	SiteName        string
	Hostname        string
	FirmwareVersion string
	Uptime          *time.Time
	Downtime        *time.Time
	LastRebootAt    *time.Time
	RebootReason    string
	Radios          []Radio
}

func msToTime(ms int64) time.Time {
	ns := time.Duration(ms) * time.Millisecond

	return time.Unix(0, int64(ns))
}

func (api *DeviceAPIResponse) Normalize() *Device {
	dev := &Device{
		Model:           api.Model,
		MAC:             api.MAC,
		Serial:          api.Serial,
		SiteName:        api.SiteName,
		Hostname:        api.Config.Name,
		FirmwareVersion: api.Management.FirmwareVersion,
	}

	if api.System.Online {
		t := msToTime(api.System.UpTime)
		dev.Uptime = &t
	} else {
		t := msToTime(api.System.DownTime)
		dev.Downtime = &t
	}

	if len(api.System.Reboots) > 0 {
		lastReboot := api.System.Reboots[0]
		for _, r := range api.System.Reboots {
			if r.Unixtime > lastReboot.Unixtime {
				lastReboot = r
			}
		}
		t := time.Unix(int64(lastReboot.Unixtime), 0)
		dev.LastRebootAt = &t
		dev.RebootReason = lastReboot.Reason
	}

	for _, radio := range api.Radios {
		dev.Radios = append(dev.Radios, radio.Normalize())
	}

	return dev
}

type APGroupAPIResponse struct {
	Name             string `json:"name"`
	DevicesCount     int    `json:"deviceCount"`
	DevicesOffline   int    `json:"offlineCount"`
	DevicesOutOfSync int    `json:"outOfSyncCount"`
	ClientCount      int    `json:"clientCount"`
	ClientCount24H   int    `json:"clientCount24h"`
}
