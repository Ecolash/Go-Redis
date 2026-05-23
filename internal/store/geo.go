package store

type GeoMember struct {
	Lon    float64
	Lat    float64
	Member string
}

// Source: https://github.com/codecrafters-io/redis-geocoding-algorithm
func geoScore(lon, lat float64) uint64 {
	const (
		minLat = -85.05112878
		maxLat = 85.05112878
		minLon = -180.0
		maxLon = 180.0
		scale  = float64(uint64(1) << 26)
	)

	latNorm := uint32(scale * (lat - minLat) / (maxLat - minLat))
	lonNorm := uint32(scale * (lon - minLon) / (maxLon - minLon))

	spread := func(v uint32) uint64 {
		x := uint64(v)

		x = (x | (x << 16)) & 0x0000FFFF0000FFFF
		x = (x | (x << 8)) & 0x00FF00FF00FF00FF
		x = (x | (x << 4)) & 0x0F0F0F0F0F0F0F0F
		x = (x | (x << 2)) & 0x3333333333333333
		x = (x | (x << 1)) & 0x5555555555555555

		return x
	}

	return spread(latNorm) | (spread(lonNorm) << 1)
}

func (s *Store) GeoAdd(key string, members []GeoMember) int {
	zms := make([]ZSetMember, len(members))
	for i, m := range members {
		score := geoScore(m.Lon, m.Lat)
		zms[i] = ZSetMember{Score: score, Member: m.Member}
	}
	return s.ZAdd(key, zms)
}
