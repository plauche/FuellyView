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

func serveYearJson(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	carMake := r.URL.Query()["make"]

	q := datastore.NewQuery("CarInfo")

	if len(carMake) == 1 && len(carMake[0]) > 1 {
		q = q.Filter("Make =", carMake[0])
	}

	q = q.Project("Year").
		Order("-Year").
		Distinct()

	var years []string
	for t := q.Run(c); ; {
		var car2 CarInfo
		_, err := t.Next(&car2)
		if err == datastore.Done {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		year := fmt.Sprintf("%d", car2.Year)
		years = append(years, year)
	}

	dataJson, _ := json.Marshal(years)
	fmt.Fprintf(w, "%s", string(dataJson))
}

func serveMakeJson(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	year := r.URL.Query()["year"]

	q := datastore.NewQuery("CarInfo")

	if len(year) == 1 && len(year[0]) == 4 {
		carYear, _ := strconv.Atoi(year[0])
		q = q.Filter("Year =", carYear)
	}

	q = q.Project("Make").
		Distinct()

	type MakeData struct {
		Display string
		Value   string
	}

	var makes []MakeData
	for t := q.Run(c); ; {
		var car2 CarInfo
		_, err := t.Next(&car2)
		if err == datastore.Done {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		var newMake MakeData
		newMake.Value = car2.Make
		newMake.Display = strings.Replace(car2.Make, "_", " ", -1)
		newMake.Display = strings.Title(newMake.Display)

		makes = append(makes, newMake)
	}

	dataJson, _ := json.Marshal(makes)

	fmt.Fprintf(w, "%s", string(dataJson))
}
