package exporter

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	client  *Client
	apGroup string          // WiFi AP Group name
	ctx     context.Context // HTTP request context
}

type PortalCollector struct {
	client *Client
	portal string          // Guest access portal name
	ctx    context.Context // HTTP request context
}

var (
	_ prometheus.Collector = (*Collector)(nil)
	_ prometheus.Collector = (*PortalCollector)(nil)
)

const namespace = "cambium_maestro"

var (
	ctrlUp = prometheus.NewDesc(namespace+"up", "indicator whether cloud controller is reachable", nil, nil)

	groupLabels           = []string{"name"} // used in groupDesc
	groupDevicesCount     = groupDesc("devices_count", "number of adopted devices")
	groupDevicesOffline   = groupDesc("devices_offline_count", "number of offline devices")
	groupDevicesOutOfSync = groupDesc("devices_out_of_sync_count", "number of devices with old configuration")
	groupClientCount      = groupDesc("client_count", "number of currently connected clients")
	groupClientCount24H   = groupDesc("client_count_24h", "number of clients seen in the past 24 hours")

	apLabels   = []string{"apgroup", "mac"} // used in apDesc
	apUp       = apDesc("up", "details for AP", "model", "hostname", "serial", "site", "firmware")
	apUptime   = apDesc("uptime", "number of uptime seconds")
	apDowntime = apDesc("downtime", "number of downtime seconds")
	apReboot   = apDesc("reboot", "number of seconds since last reboot", "reason")

	radioLabels       = []string{"apgroup", "ap", "band"} // used in radioDesc
	radioChannel      = radioDesc("channel", "WiFi channel number")
	radioChannelWidth = radioDesc("channel_width", "WiFi channel width in MHz")
	radioPower        = radioDesc("power", "RF transmit power")
	radioQuality      = radioDesc("quality", "RF quality measurement in percentage points")
	radioXfer         = radioDesc("transfer_rate", "current traffic rate in bps", "direction")

	portalSessions   = prometheus.NewDesc(namespace+"_sessions_count", "number of active sessions", []string{"name"}, nil)
	portalAPSessions = prometheus.NewDesc(namespace+"_ap_sessions_count", "number of active sessions", []string{"portal", "mac"}, nil)
)

func (*Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- ctrlUp

	ch <- groupDevicesCount
	ch <- groupDevicesOffline
	ch <- groupDevicesOutOfSync
	ch <- groupClientCount
	ch <- groupClientCount24H

	ch <- apUp
	ch <- apUptime
	ch <- apDowntime
	ch <- apReboot

	ch <- radioChannel
	ch <- radioChannelWidth
	ch <- radioPower
	ch <- radioQuality
	ch <- radioXfer
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.client.log.Debugf("collecting metrics for %s", c.apGroup)

	metric := func(desc *prometheus.Desc, v float64, labels ...string) {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, labels...)
	}
	intMetric := func(desc *prometheus.Desc, v int, labels ...string) {
		metric(desc, float64(v), labels...)
	}
	now := time.Now()
	timeMetric := func(desc *prometheus.Desc, v *time.Time, labels ...string) {
		if v != nil {
			metric(desc, now.Sub(*v).Seconds(), labels...)
		}
	}

	group, err := c.client.fetchAPGroupData(c.ctx, c.apGroup)
	if err != nil {
		c.client.log.Errorf("fetching AP group data for %s failed with %v", c.apGroup, err)
		metric(ctrlUp, 0)
		return
	}

	devices, err := c.client.fetchDevices(c.ctx, c.apGroup)
	if err != nil {
		c.client.log.Errorf("fetching device data for %s failed with %v", c.apGroup, err)
		metric(ctrlUp, 0)
		return
	}

	metric(ctrlUp, 1)

	name := group.Name
	intMetric(groupDevicesCount, group.DevicesCount, name)
	intMetric(groupDevicesOffline, group.DevicesOffline, name)
	intMetric(groupDevicesOutOfSync, group.DevicesOutOfSync, name)
	intMetric(groupClientCount, group.ClientCount, name)
	intMetric(groupClientCount24H, group.ClientCount24H, name)

	for _, dev := range devices {
		mac := dev.MAC
		metric(apUp, 1, name, mac, dev.Model, dev.Hostname, dev.Serial, dev.SiteName, dev.FirmwareVersion)

		timeMetric(apUptime, dev.Uptime, name, mac)
		timeMetric(apDowntime, dev.Downtime, name, mac)
		timeMetric(apReboot, dev.LastRebootAt, name, mac, dev.RebootReason)

		for _, r := range dev.Radios {
			band := string(r.Band)
			intMetric(radioChannel, r.Channel, name, mac, band)
			intMetric(radioChannelWidth, r.ChannelWidth, name, mac, band)
			intMetric(radioPower, r.Power, name, mac, band)
			intMetric(radioQuality, r.Quality, name, mac, band)

			// controller reports kBit/s, we export Bit/s
			intMetric(radioXfer, r.Tx*1000, name, mac, band, "out")
			intMetric(radioXfer, r.Rx*1000, name, mac, band, "in")
		}
	}
}

func (p *PortalCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- ctrlUp
	ch <- portalSessions
	ch <- portalAPSessions
}

func (c *PortalCollector) Collect(ch chan<- prometheus.Metric) {
	c.client.log.Debugf("collecting metrics for portal %s", c.portal)

	metric := func(desc *prometheus.Desc, v float64, labels ...string) {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, labels...)
	}

	sessions, total, err := c.client.fetchPortalSessions(c.ctx, c.portal)
	if err != nil {
		c.client.log.Errorf("fetching portal data for %s failed with %v", c.portal, err)
		metric(ctrlUp, 0)
		return
	}

	metric(ctrlUp, 1)
	metric(portalSessions, float64(total), c.portal)

	for _, s := range sessions {
		metric(portalAPSessions, float64(s.Sessions), c.portal, s.DeviceMAC)
	}
}

func groupDesc(name, help string, extraLabel ...string) *prometheus.Desc {
	fqdn := prometheus.BuildFQName(namespace, "ap_group", name)
	return prometheus.NewDesc(fqdn, help, groupLabels, nil)
}

func apDesc(name, help string, extraLabel ...string) *prometheus.Desc {
	fqdn := prometheus.BuildFQName(namespace, "ap", name)
	return prometheus.NewDesc(fqdn, help, append(apLabels, extraLabel...), nil)
}

func radioDesc(name, help string, extraLabel ...string) *prometheus.Desc {
	fqdn := prometheus.BuildFQName(namespace, "ap_radio", name)
	return prometheus.NewDesc(fqdn, help, append(radioLabels, extraLabel...), nil)
}
