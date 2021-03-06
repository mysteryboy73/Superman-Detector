package builders

import (
	"errors"

	"github.com/mysteryboy73/Superman-Detector/models"
	"github.com/mysteryboy73/Superman-Detector/repository"
	"github.com/umahmood/haversine"
)

// ResponseBuilder interface for building response
type ResponseBuilder interface {
	Build(loginRequest models.LoginRequest) (response models.TravelResponse, err error)
}

// LoginResponseBuilder retrieves necessary data for response
type LoginResponseBuilder struct {
	geoRepository repository.DataRepo
}

// NewLoginResponseBuilder returns a LoginResponseBuilder with repository dependency
func NewLoginResponseBuilder() *LoginResponseBuilder {
	loginResponseBuilder := LoginResponseBuilder{geoRepository: repository.NewGeoRepository()}
	return &loginResponseBuilder
}

// Build constructs a TravelResponse
func (lrb *LoginResponseBuilder) Build(request models.LoginRequest) (models.TravelResponse, error) {
	var response models.TravelResponse

	// Save current login attempt to the database
	saved := lrb.geoRepository.SaveLogin(request)

	if saved != true {
		return models.TravelResponse{}, errors.New("login could not be saved")
	}

	// Retrieve the current login attempt location information
	currentLocation, err := lrb.geoRepository.GetLocation(request.IPAddress)
	if err != nil {
		return models.TravelResponse{}, errors.New("could not retrieve location")
	}

	// Get the details of the previous and subsequent login attempts
	previousLogin, futureLogin := lrb.geoRepository.GetPreviousAndFutureIPAdress(request.Username, request.IPAddress, request.UnixTimestamp) // Should also probably return an error

	// Get previous and future location information
	previousLocation, _ := lrb.geoRepository.GetLocation(previousLogin.IPAddress)
	futureLocation, _ := lrb.geoRepository.GetLocation(futureLogin.IPAddress)

	// Tack on ip addresses and timestamp to resposne
	previousLocation.IP = previousLogin.IPAddress
	previousLocation.TimeStamp = previousLogin.UnixTimestamp
	futureLocation.IP = futureLogin.IPAddress
	futureLocation.TimeStamp = futureLogin.UnixTimestamp

	// Tack on speed
	previousLocation.Speed = calculateGeoSuspiciousSpeed(previousLogin, request, previousLocation, currentLocation)
	futureLocation.Speed = calculateGeoSuspiciousSpeed(request, futureLogin, currentLocation, futureLocation)

	// Tack on geo suspicious
	response.TravelFromCurrentGeoSuspicious = previousLocation.Speed > 500
	response.TravelToCurrentGeoSuspicious = futureLocation.Speed > 500

	response.CurrentLocation = currentLocation
	response.PreviousLocation = &previousLocation
	response.FutureLocation = &futureLocation

	return response, nil
}

func calculateGeoSuspiciousSpeed(fromLogin models.LoginRequest, toLogin models.LoginRequest, fromLocation models.GeoLocation, toLocation models.GeoLocation) int {
	// Get time differences
	timeTraveled := calculateTimeDifferenceInHours(fromLogin.UnixTimestamp, toLogin.UnixTimestamp)

	// Distance between latitudes and longitudes
	location1 := haversine.Coord{Lat: fromLocation.Lat, Lon: fromLocation.Lon}
	location2 := haversine.Coord{Lat: toLocation.Lat, Lon: toLocation.Lon}
	distanceMi, _ := haversine.Distance(location1, location2)

	// Find speed travled
	speed := distanceMi / timeTraveled

	return int(speed)
}

func calculateTimeDifferenceInHours(startTime int, endTime int) float64 {
	timeDifferenceSecond := endTime - startTime
	timeDifferenceHours := timeDifferenceSecond / 3600
	return float64(timeDifferenceHours)
}
