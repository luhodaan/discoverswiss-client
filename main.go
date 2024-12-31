package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Accommodation struct {
	Source     string `default:"discoverSwiss"`
	Active     bool   `default:"true"`
	Shortname  string

	Meta struct {
		Id string `json:"Id"`
	} `json:"_Meta"`

	AccoDetail struct {
		Language AccoDetailLanguage `json:"de"`
	} `json:"AccoDetail"`

	GpsInfo []struct {
		Gpstype   string  `json:"Gpstype"`
		Latitude  float64 `json:"Latitude"`
		Longitude float64 `json:"Longitude"`
	} `json:"GpsInfo"`

	// AccoCategory struct {
	// 	Id string `json:"Id"`
	// } `json:"AccoCategory"`

	AccoType struct {
		Id string `json:"Id"`
	} `json:"AccoType"`

	AccoOverview struct {
		TotalRooms   int    `json:"TotalRooms"`
		SingleRooms  int    `json:"SingleRooms"`
		DoubleRooms  int    `json:"DoubleRooms"`
		CheckInFrom  string `json:"CheckInFrom"`
		CheckInTo    string `json:"CheckInTo"`
		CheckOutFrom string `json:"CheckOutFrom"`
		CheckOutTo   string `json:"CheckOutTo"`
		MaxPersons   int    `json:"MaxPersons"`
	} `json:"AccoOverview"`
}

type AccoDetailLanguage struct {
	Name        string `json:"Name"`
	Street      string `json:"Street"`
	Zip         string `json:"Zip"`
	City        string `json:"City"`
	CountryCode string `json:"CountryCode"`
	Email       string `json:"Email"`
	Phone       string `json:"Phone"`
}

type DiscoverSwissResponse struct {
	Count         int               `json:"count"`
	HasNextPage   bool              `json:"hasNextPage"`
	NextPageToken string            `json:"nextPageToken"`
	Data          []LodgingBusiness `json:"data"`
}
type LodgingBusiness struct {
	Name string `json:"name"`

	Address struct {
		AddressCountry  string `json:"addressCountry"`
		AddressLocality string `json:"addressLocality"`
		PostalCode      string `json:"postalCode"`
		StreetAddress   string `json:"streetAddress"`
		Email           string `json:"email"`
		Telephone       string `json:"telephone"`
	} `json:"address"`

	Geo struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"geo"`

	NumberOfRooms []struct {
		PropertyID string `json:"propertyId"`
		Value      string `json:"value"`
	} `json:"numberOfRooms"`

	StarRating StarRating `json:"starRating"`

	NumberOfBeds int `json:"numberOfBeds"`

	Identifier string `json:"identifier"`

	CheckinTime      string `json:"checkinTime"`
	CheckinTimeTo    string `json:"checkinTimeTo"`
	CheckoutTimeFrom string `json:"checkoutTimeFrom"`
	CheckoutTime     string `json:"checkoutTime"`
}

type StarRating struct {
	RatingValue    float64 `json:"ratingValue"`
	AdditionalType string  `json:"additionalType"`
	Name           string  `json:"name"`
}

var env struct {
	HTTP_URL    string
	HTTP_METHOD string `default:"GET"`

	SUBSCRIPTION_KEY string
}

const ENV_HEADER_PREFIX = "HTTP_HEADER_"

func FailOnError(err error, msg string) {
	if err != nil {
		slog.Error(msg, "err", err)
		panic(err)
	}
}

func lodgingRequest(url *url.URL, httpHeaders http.Header, httpMethod string) (string, error) {
	headers := httpHeaders
	u := url
	req, err := http.NewRequest(httpMethod, u.String(), http.NoBody)
	if err != nil {
		return "", fmt.Errorf("could not create http request: %w", err)
	}

	req.Header = headers

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error during http request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http request returned non-Ok status: %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading DiscoverSwissResponse body: %w", err)
	}

	return string(body), nil
}

func customHeaders() http.Header {
	headers := http.Header{}
	for _, env := range os.Environ() {
		for i := 1; i < len(env); i++ {
			if env[i] == '=' {
				envk := env[:i]
				if strings.HasPrefix(envk, ENV_HEADER_PREFIX) {
					envv := env[i+1:]
					headerName, headerValue, found := strings.Cut(envv, ":")
					if !found {
						slog.Error("invalid header config", "env", envk, "val", envv)
						panic("invalid header config")
					}
					headers[headerName] = []string{strings.TrimSpace(headerValue)}
				}
				break
			}
		}
	}
	return headers
}

// // Helper function to map star rating name to AccoCategory Id
// func mapStarRatingToAccoCategoryId(name string) string {
// 	// Check if the name contains a number
// 	if len(name) > 0 && (name[0] >= '1' && name[0] <= '5') {
// 		return fmt.Sprintf("%dstars", name[0]-'0')
// 	}
// 	// Return the original name if it doesn't contain stars
// 	return name
// }

func mapAdditionalTypeToAccoTypeId(additionalType string) string {
	if strings.EqualFold(additionalType, "Hotel") {
		return "HotelPension"
	}
	return additionalType
}
func mapLodgingBusinessToAccommodation(lb LodgingBusiness) Accommodation {
	acco := Accommodation{
		Source:    "discoverSwiss",
		Active:    true,
		Shortname: lb.Name,
	}

	acco.Meta.Id = lb.Identifier

	acco.GpsInfo = []struct {
		Gpstype   string  `json:"Gpstype"`
		Latitude  float64 `json:"Latitude"`
		Longitude float64 `json:"Longitude"`
	}{
		{
			Gpstype:   "position",
			Latitude:  lb.Geo.Latitude,
			Longitude: lb.Geo.Longitude,
		},
	}

	acco.AccoDetail.Language = AccoDetailLanguage{
		Name:        lb.Name,
		Street:      lb.Address.StreetAddress,
		Zip:         lb.Address.PostalCode,
		City:        lb.Address.AddressLocality,
		CountryCode: lb.Address.AddressCountry,
		Email:       lb.Address.Email,
		Phone:       lb.Address.Telephone,
	}

	var totalRooms, singleRooms, doubleRooms int
	for _, room := range lb.NumberOfRooms {
		value := 0
		fmt.Sscanf(room.Value, "%d", &value)

		switch room.PropertyID {
		case "total":
			totalRooms = value
		case "single":
			singleRooms = value
		case "double":
			doubleRooms = value
		}
	}

	acco.AccoOverview = struct {
		TotalRooms   int    `json:"TotalRooms"`
		SingleRooms  int    `json:"SingleRooms"`
		DoubleRooms  int    `json:"DoubleRooms"`
		CheckInFrom  string `json:"CheckInFrom"`
		CheckInTo    string `json:"CheckInTo"`
		CheckOutFrom string `json:"CheckOutFrom"`
		CheckOutTo   string `json:"CheckOutTo"`
		MaxPersons   int    `json:"MaxPersons"`
	}{
		TotalRooms:   totalRooms,
		SingleRooms:  singleRooms,
		DoubleRooms:  doubleRooms,
		CheckInFrom:  lb.CheckinTime,
		CheckInTo:    lb.CheckinTimeTo,
		CheckOutFrom: lb.CheckoutTimeFrom,
		CheckOutTo:   lb.CheckoutTime,
		MaxPersons:   lb.NumberOfBeds,
	}

	// acco.AccoCategory = struct {
	// 	Id   string `json:"Id"`
	// 	Self string `json:"Self"`
	// }{
	// 	Id: mapStarRatingToAccoCategoryId(lodging.StarRating.Name),
	// 	Self: fmt.Sprintf("https://api.tourism.testingmachine.eu/v1/AccommodationTypes/%s",
	// 		mapStarRatingToAccoCategoryId(lodging.StarRating.Name)),
	// }

	acco.AccoType = struct {
		Id string `json:"Id"`
	}{
		Id: mapAdditionalTypeToAccoTypeId(lb.StarRating.AdditionalType),
	}

	return acco
}

func main() {
	err := godotenv.Load()
	if err != nil {
		slog.Error("Error loading .env file", "err", err)
	}

	envconfig.MustProcess("", &env)

	headers := customHeaders()
	url, err := url.Parse(env.HTTP_URL)
	if err != nil {
		slog.Error("failed parsing url", "err", err)
		return
	}

	body, err := lodgingRequest(url, headers, "GET")
	if err != nil {
		slog.Error("failed making request", "err", err)
		return
	}

	var DiscoverSwissResponse DiscoverSwissResponse
	err = json.Unmarshal([]byte(body), &DiscoverSwissResponse)
	if err != nil {
		slog.Error("failed unmarshalling DiscoverSwissResponse object", "err", err)
		return
	}

	//fmt.Printf("%+v\n", DiscoverSwissResponse.Data[0])

	accomodation := mapLodgingBusinessToAccommodation(DiscoverSwissResponse.Data[0])

	//fmt.Printf("%+v\n", accomodation)

	jsonData, _ := json.MarshalIndent(accomodation, "", "    ")
	fmt.Println(string(jsonData))

}
