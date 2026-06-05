package utils

import "math"

// HaversineDistance returns the distance in metres between two lat/lng points.
func HaversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371000 // Earth radius in metres

	lat1R := lat1 * math.Pi / 180
	lat2R := lat2 * math.Pi / 180
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1R)*math.Cos(lat2R)*math.Sin(dLng/2)*math.Sin(dLng/2)

	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func IsWithinGeofence(userLat, userLng, companyLat, companyLng, radiusMetres float64) bool {
	return HaversineDistance(userLat, userLng, companyLat, companyLng) <= radiusMetres
}
