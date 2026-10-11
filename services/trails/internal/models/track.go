package models

import "math"

const earthRadiusKm = 6371.0

// gainThreshold ignores elevation jitter below this many meters, as GPS
// tracks are noisy.
const gainThreshold = 3.0

// ComputeTrackStats measures a track; nil for an empty one.
func ComputeTrackStats(track []TrackPoint) *TrackStats {
	if len(track) < 2 {
		return nil
	}
	stats := &TrackStats{}
	var distance, gain float64
	var minEle, maxEle, ref *float64
	for i, p := range track {
		if i > 0 {
			distance += haversine(*track[i-1][0], *track[i-1][1], *p[0], *p[1])
		}
		if e := p[2]; e != nil {
			if minEle == nil || *e < *minEle {
				minEle = e
			}
			if maxEle == nil || *e > *maxEle {
				maxEle = e
			}
			switch {
			case ref == nil:
				ref = e
			case *e-*ref >= gainThreshold:
				gain += *e - *ref
				ref = e
			case *e < *ref:
				ref = e
			}
		}
	}
	stats.DistanceKm = math.Round(distance*10) / 10
	stats.ElevationGain = int(math.Round(gain))
	if minEle != nil {
		lo, hi := int(math.Round(*minEle)), int(math.Round(*maxEle))
		stats.MinElevation, stats.MaxElevation = &lo, &hi
	}
	return stats
}

func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat, dLng := toRad(lat2-lat1), toRad(lng2-lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusKm * math.Asin(math.Sqrt(a))
}
