package domain

import (
	"go.mongodb.org/mongo-driver/bson/primitive"

	pb "ride-sharing/shared/proto/trip"

	tripTypes "ride-sharing/services/trip-service/pkg/types"
)

type RideFareModel struct {
	ID                primitive.ObjectID         `bson:"_id,omitempty"`
	UserID            string                     `bson:"userID"`
	PackageSlug       string                     `bson:"packageSlug"`
	TotalPriceInCents float64                    `bson:"totalPriceInCents"`
	Route             *tripTypes.OsrmApiResponse `bson:"route"`
}

func (r *RideFareModel) ToProto() *pb.RideFare {
	return &pb.RideFare{
		Id:                r.ID.Hex(),
		UserID:            r.UserID,
		PackageSlug:       r.PackageSlug,
		TotalPriceInCents: r.TotalPriceInCents,
	}
}

func ToRideFaresProto(fares []*RideFareModel) []*pb.RideFare {
	var protoFares []*pb.RideFare

	for _, f := range fares {
		protoFares = append(protoFares, f.ToProto())
	}

	return protoFares
}
