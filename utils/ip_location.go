package utils

import (
	"github.com/imroc/req"
	"github.com/ontio/ontology/common/log"
)

type GeoLocation struct {
	Lat     float32 `json:"lat"`
	Lon     float32 `json:"lon"`
	Country string  `json:"country"`
}

func GetIpLocation(ip string) *GeoLocation {
	r, err := req.Get("http://ip-api.com/json/" + ip)
	if err != nil {
		log.Error("fetch ip location from remote error " + err.Error())
		return nil
	}
	var location GeoLocation
	_ = r.ToJSON(&location)
	return &location
}
