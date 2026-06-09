package main

import (
	"log"
	math "math/rand/v2"
	pb "ride-sharing/shared/proto/driver"
	"ride-sharing/shared/util"
	"sync"

	"github.com/mmcloughlin/geohash"
)

type driverInMap struct {
	Driver *pb.Driver
	// Index int
	// TODO: route
}

type Service struct {
	drivers []*driverInMap
	mu      sync.RWMutex
}

func NewService() *Service {
	return &Service{
		drivers: make([]*driverInMap, 0),
	}
}

func (s *Service) FindAvailableDrivers(packageType string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matchingDrivers []string

	log.Printf("[MATCH CHECK] Total tracked drivers in memory: %d. Looking for slug: '%s'", len(s.drivers), packageType)

	for i, driver := range s.drivers {
		if driver == nil || driver.Driver == nil {
			log.Printf("[MATCH CHECK] Index %d has a nil driver block container", i)
			continue
		}

		// 🌟 THIS LINE SHOWS YOU THE RAW COMPARISON TRUTH:
		log.Printf("[MATCH CHECK] Comparing Driver ID '%s': Stored Slug='%s' vs Requested Slug='%s'",
			driver.Driver.Id, driver.Driver.PackageSlug, packageType)

		if driver.Driver.PackageSlug == packageType {
			matchingDrivers = append(matchingDrivers, driver.Driver.Id)
		}
	}

	return matchingDrivers
}

func (s *Service) RegisterDriver(driverId string, packageSlug string) (*pb.Driver, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	randomIndex := math.IntN(len(PredefinedRoutes))
	randomRoute := PredefinedRoutes[randomIndex]

	randomPlate := GenerateRandomPlate()
	randomAvatar := util.GetRandomAvatar(randomIndex)

	// we can ignore this property for now, but it must be sent to the frontend.
	geohash := geohash.Encode(randomRoute[0][0], randomRoute[0][1])

	driver := &pb.Driver{
		Id:             driverId,
		Geohash:        geohash,
		Location:       &pb.Location{Latitude: randomRoute[0][0], Longitude: randomRoute[0][1]},
		Name:           "Lando Norris",
		PackageSlug:    packageSlug,
		ProfilePicture: randomAvatar,
		CarPlate:       randomPlate,
	}

	s.drivers = append(s.drivers, &driverInMap{
		Driver: driver,
	})

	log.Printf("After adding driver map %v", s.drivers)

	return driver, nil
}

func (s *Service) UnregisterDriver(driverId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, driver := range s.drivers {
		if driver.Driver.Id == driverId {
			s.drivers = append(s.drivers[:i], s.drivers[i+1:]...)
		}
	}
}
