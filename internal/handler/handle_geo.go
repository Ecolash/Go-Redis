package handler

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/errs"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
	"github.com/codecrafters-io/redis-starter-go/internal/store"
)

const (
	minLon = -180.0
	maxLon = 180.0
	minLat = -85.05112878
	maxLat = 85.05112878
)

func (h *Handler) handleGeoPos(parts []string) string {
	// GEOPOS key member [member ...]
	if len(parts) < 3 {
		return errs.WrongArgs
	}
	members := parts[2:]
	results := h.store.GeoPos(parts[1], members)
	var sb strings.Builder
	fmt.Fprintf(&sb, "*%d\r\n", len(results))
	for _, r := range results {
		if r == nil {
			sb.WriteString(nullArray)
		} else {
			lonStr := strconv.FormatFloat(r.Lon, 'g', 17, 64)
			latStr := strconv.FormatFloat(r.Lat, 'g', 17, 64)
			sb.WriteString("*2\r\n")
			sb.WriteString(resp.BulkString(lonStr))
			sb.WriteString(resp.BulkString(latStr))
		}
	}
	return sb.String()
}

func (h *Handler) handleGeoAdd(parts []string) string {
	// GEOADD key longitude latitude member [longitude latitude member ...]
	if len(parts) < 5 || (len(parts)-2)%3 != 0 {
		return errs.WrongArgs
	}
	key := parts[1]
	members := make([]store.GeoMember, 0, (len(parts)-2)/3)
	for i := 2; i < len(parts); i += 3 {
		lon, err := strconv.ParseFloat(parts[i], 64)
		if err != nil || math.IsNaN(lon) {
			return resp.Error("ERR value is not a valid float")
		}
		lat, err := strconv.ParseFloat(parts[i+1], 64)
		if err != nil || math.IsNaN(lat) {
			return resp.Error("ERR value is not a valid float")
		}
		if lon < minLon || lon > maxLon || lat < minLat || lat > maxLat {
			return resp.Error(fmt.Sprintf("ERR invalid longitude,latitude pair %g,%g", lon, lat))
		}
		members = append(members, store.GeoMember{Lon: lon, Lat: lat, Member: parts[i+2]})
	}
	n := h.store.GeoAdd(key, members)
	return resp.Integer(n)
}
