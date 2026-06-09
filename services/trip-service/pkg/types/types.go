package types

import (
	pb "ride-sharing/shared/proto/trip"
)

type OsrmApiResponse struct {
	Routes []struct {
		Distance float64 `json:"distance"`
		Duration float64 `json:"duration"`
		Geometry struct {
			Coordinate [][]float64 `json:"coordinates"`
		} `json:"geometry"`
	} `json:"routes"`
}

func (o *OsrmApiResponse) ToProto() *pb.Route {
	// 1. Guard Check: Defend against empty route slices from OSRM
	if o == nil || len(o.Routes) == 0 {
		return &pb.Route{
			Geometry: []*pb.Geometry{},
			Distance: 0,
			Duration: 0,
		}
	}

	// Now it is 100% safe to read index [0]
	route := o.Routes[0]
	geometry := route.Geometry.Coordinate
	coordinates := make([]*pb.Coordinate, len(geometry))

	for i, coord := range geometry {
		// 2. Fix Layout: OSRM returns [Longitude, Latitude].
		// coord[1] is Latitude, coord[0] is Longitude.
		coordinates[i] = &pb.Coordinate{
			Latitude:  coord[1],
			Longitude: coord[0],
		}
	}

	return &pb.Route{
		Geometry: []*pb.Geometry{
			{
				Coordinates: coordinates,
			},
		},
		Distance: route.Distance,
		Duration: route.Duration,
	}
}

type PricingConfig struct {
	PricePerUnitOfDistance float64
	PricingPerMinute       float64
}

func DefaultPricingConfig() *PricingConfig {
	return &PricingConfig{
		PricePerUnitOfDistance: 1.5,
		PricingPerMinute:       0.25,
	}

}
