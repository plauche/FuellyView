// servers.go
package FuellyView

import (
	"appengine"
	"appengine/datastore"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type MakeData struct {
	Display string
	Value   string
}

type FilterData struct {
	Makes []MakeData
	Years []string
}

type CarDisplay struct {
	Url     string
	Display string
}

func serveFiltersJson(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	var filterData FilterData
	yearQuery := ""
	makeQuery := ""
	makeData := r.URL.Query()["make"]
	yearData := r.URL.Query()["year"]

	if len(makeData) == 1 && len(makeData[0]) > 1 {
		makeQuery = makeData[0]
	}

	if len(yearData) == 1 && len(yearData[0]) == 4 {
		yearQuery = yearData[0]
	}

	filterData.Years = getYears(c, makeQuery)
	filterData.Makes = getMakes(c, yearQuery)

	dataJson, err := json.Marshal(filterData)
	if err == nil {
		fmt.Fprintf(w, "%s", string(dataJson))
	}
}

func getMakes(c appengine.Context, yearData string) []MakeData {

	q := datastore.NewQuery("CarInfo")

	if len(yearData) == 4 {
		carYear, _ := strconv.Atoi(yearData)
		q = q.Filter("Year =", carYear)
	}

	q = q.Project("Make").
		Distinct()

	var makes []MakeData
	for t := q.Run(c); ; {
		var car2 CarInfo
		_, err := t.Next(&car2)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return makes
		}

		var newMake MakeData
		newMake.Value = car2.Make
		newMake.Display = strings.Replace(car2.Make, "_", " ", -1)
		newMake.Display = strings.Title(newMake.Display)

		makes = append(makes, newMake)
	}

	return makes
}

func getYears(c appengine.Context, makeData string) []string {
	var years []string

	q := datastore.NewQuery("CarInfo")

	if len(makeData) > 0 {
		q = q.Filter("Make =", makeData)
	}

	q = q.Project("Year").
		Order("-Year").
		Distinct()

	for t := q.Run(c); ; {
		var car2 CarInfo
		_, err := t.Next(&car2)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return years
		}

		year := fmt.Sprintf("%d", car2.Year)
		years = append(years, year)
	}

	return years
}

func serveCarsJson(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	makeQuery := ""
	yearQuery := 0
	carMake := r.URL.Query()["make"]
	year := r.URL.Query()["year"]

	if len(carMake) == 1 && len(carMake[0]) > 1 {
		makeQuery = carMake[0]
	}

	if len(year) == 1 && len(year[0]) == 4 {
		yearQuery, _ = strconv.Atoi(year[0])
	}

	type CarData struct {
		TopCars    []CarDisplay
		BottomCars []CarDisplay
	}

	var dataDisplay CarData

	dataDisplay.BottomCars = getCars(c, makeQuery, yearQuery, "Mpg")
	dataDisplay.TopCars = getCars(c, makeQuery, yearQuery, "-Mpg")

	dataJson, _ := json.Marshal(dataDisplay)

	fmt.Fprintf(w, "%s", string(dataJson))
}

func getCars(c appengine.Context, searchMake string, searchYear int, orderBy string) []CarDisplay {
	var dataDisplay []CarDisplay
	q := datastore.NewQuery("CarInfo").
		Order(orderBy).
		Limit(10)

	if len(searchMake) > 0 {
		q = q.Filter("Make =", searchMake)
	}

	if searchYear > 0 {
		q = q.Filter("Year = ", searchYear)
	}

	for t := q.Run(c); ; {
		var car2 CarInfo
		_, err := t.Next(&car2)
		if err == datastore.Done {
			break
		}
		if err != nil {
			break
		}
		var info CarDisplay
		info.Url = fmt.Sprintf("%s/%d", car2.Url, car2.Year)
		info.Display = fmt.Sprintf("%d %s %s (%-3.1f)",
			car2.Year,
			strings.Title(strings.Replace(car2.Make, "_", " ", -1)),
			strings.Title(strings.Replace(car2.Model, "_", " ", -1)),
			car2.Mpg)
		dataDisplay = append(dataDisplay, info)
	}

	return dataDisplay
}
