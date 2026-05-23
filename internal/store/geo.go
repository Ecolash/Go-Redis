package store

import "math"

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

func geoPos(score uint64) (lon, lat float64) {
	const (
		minLat = -85.05112878
		maxLat = 85.05112878
		minLon = -180.0
		maxLon = 180.0
		scale  = float64(uint64(1) << 26)
	)
	compact := func(x uint64) uint32 {
		x &= 0x5555555555555555
		x = (x | (x >> 1)) & 0x3333333333333333
		x = (x | (x >> 2)) & 0x0F0F0F0F0F0F0F0F
		x = (x | (x >> 4)) & 0x00FF00FF00FF00FF
		x = (x | (x >> 8)) & 0x0000FFFF0000FFFF
		x = (x | (x >> 16)) & 0x00000000FFFFFFFF
		return uint32(x)
	}
	latNorm := compact(score)
	lonNorm := compact(score >> 1)
	latStep := (maxLat - minLat) / scale
	lonStep := (maxLon - minLon) / scale
	lat = minLat + (float64(latNorm) + 0.5) * latStep
	lon = minLon + (float64(lonNorm) + 0.5) * lonStep
	return
}

// GeoPosResult holds the decoded coordinates for a geo member.
type GeoPosResult struct {
	Lon float64
	Lat float64
}

// GeoPos returns the decoded coordinates for each member.
// Entries for missing keys or members are nil.
func (s *Store) GeoPos(key string, members []string) []*GeoPosResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	results := make([]*GeoPosResult, len(members))
	e, ok := s.data[key]
	if !ok || e.kind != kindZSet {
		return results
	}
	for i, m := range members {
		score, exists := e.zsetVal.scores[m]
		if !exists {
			continue
		}
		lon, lat := geoPos(uint64(score))
		results[i] = &GeoPosResult{Lon: lon, Lat: lat}
	}
	return results
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6372.797560856
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	lat1 = lat1 * math.Pi / 180
	lat2 = lat2 * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Asin(math.Sqrt(a))
	return r * c * 1000
}

// GeoDist returns the distance in meters between two members, or -1 if either is missing.
func (s *Store) GeoDist(key, member1, member2 string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	if !ok || e.kind != kindZSet {
		return 0, false
	}
	score1, ok1 := e.zsetVal.scores[member1]
	score2, ok2 := e.zsetVal.scores[member2]
	if !ok1 || !ok2 {
		return 0, false
	}
	lon1, lat1 := geoPos(uint64(score1))
	lon2, lat2 := geoPos(uint64(score2))
	return haversine(lat1, lon1, lat2, lon2), true
}

// GeoSearch returns members within radiusMeters of (lon, lat).
func (s *Store) GeoSearch(key string, lon, lat, radiusMeters float64) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	if !ok || e.kind != kindZSet {
		return nil
	}
	var results []string
	for member, score := range e.zsetVal.scores {
		mLon, mLat := geoPos(uint64(score))
		if haversine(lat, lon, mLat, mLon) <= radiusMeters {
			results = append(results, member)
		}
	}
	return results
}

func (s *Store) GeoAdd(key string, members []GeoMember) int {
	zms := make([]ZSetMember, len(members))
	for i, m := range members {
		zms[i] = ZSetMember{Score: float64(geoScore(m.Lon, m.Lat)), Member: m.Member}
	}
	return s.ZAdd(key, zms)
}
