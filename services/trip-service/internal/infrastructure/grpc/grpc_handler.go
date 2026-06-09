package grpc

import (
	"context"
	"log"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/services/trip-service/internal/infrastructure/events"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type gRPCHandler struct {
	pb.UnimplementedTripServiceServer
	service   domain.TripService
	publisher *events.TripEventPublisher
}

func NewGRPCHandler(server *grpc.Server, service domain.TripService, publisher *events.TripEventPublisher) *gRPCHandler {
	handler := &gRPCHandler{
		service:   service,
		publisher: publisher,
	}

	pb.RegisterTripServiceServer(server, handler)
	return handler
}

func (h *gRPCHandler) PreviewTrip(ctx context.Context, req *pb.PreviewTripRequest) (*pb.PreviewTripResponse, error) {
	// 1. Guard Check: Ensure coordinates aren't arriving as nil payloads
	if req.GetStartLocation() == nil || req.GetEndLocation() == nil {
		return nil, status.Error(codes.InvalidArgument, "startLocation and endLocation parameters are completely required")
	}

	pickup := req.GetStartLocation()
	destination := req.GetEndLocation()

	pickupCoord := &types.Coordinate{
		Latitude:  pickup.Latitude,
		Longitude: pickup.Longitude,
	}

	destinationCoord := &types.Coordinate{
		Latitude:  destination.Latitude,
		Longitude: destination.Longitude,
	}

	userID := req.GetUserId()

	log.Printf("User id %v", userID)

	// 2. Call Domain Service Layer
	route, err := h.service.GetRoute(ctx, pickupCoord, destinationCoord)
	if err != nil {
		log.Printf("[gRPC Core Error] GetRoute failed execution: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get routes from domain: %v", err)
	}

	// 3. Guard Check: Ensure the entity object populated successfully
	if route == nil {
		log.Println("[gRPC Core Error] GetRoute returned a nil route model without an explicit error context")
		return nil, status.Error(codes.NotFound, "calculated route vector was empty")
	}

	estimatedFares := h.service.EstimatePackagesPriceWithRoute(route)

	fares, err := h.service.GenerateTripFares(ctx, estimatedFares, userID, route)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate ride fares: %v", err)
	}

	// 4. Return matching your strict .proto contract fields
	return &pb.PreviewTripResponse{
		TripID:    "generated_placeholder_uuid_123", // Added missing string tripID = 1 field
		Route:     route.ToProto(),
		RideFares: domain.ToRideFaresProto(fares), // Ensured casing aligns with compiled outputs
	}, nil
}

func (h *gRPCHandler) CreateTrip(ctx context.Context, req *pb.CreateTripRequest) (*pb.CreateTripResponse, error) {
	fareID := req.GetRideFareID()
	userID := req.GetUserID()

	rideFare, err := h.service.GetAndValidateFare(ctx, fareID, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to validate the fare: %v", err)
	}

	trip, err := h.service.CreateTrip(ctx, rideFare)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create the trip: %v", err)
	}

	if err := h.publisher.PublishTripCreated(ctx, trip); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to publish the trip created event: %v", err)
	}

	return &pb.CreateTripResponse{
		TripID: trip.ID.Hex(),
	}, nil
}
