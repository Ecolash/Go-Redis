package store

type GeoMember struct {
	Lon    float64
	Lat    float64
	Member string
}

func (s *Store) GeoAdd(key string, members []GeoMember) int {
	zms := make([]ZSetMember, len(members))
	for i, m := range members {
		zms[i] = ZSetMember{Score: 0, Member: m.Member}
	}
	return s.ZAdd(key, zms)
}
