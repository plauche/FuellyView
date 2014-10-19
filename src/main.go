package FuellyView

import (
	"appengine"
	"appengine/datastore"
	"appengine/taskqueue"
	"appengine/urlfetch"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func init() {
	http.HandleFunc("/refresh", getData)
	http.HandleFunc("/parseCar", parseCar)
	//http.HandleFunc("/", queryDb)
	//http.HandleFunc("/makes/{make:[A-Za-z]+}.json", serveMakeJson)
	http.HandleFunc("/filters.json", serveFiltersJson)
	http.HandleFunc("/cars.json", getCarData)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
}

type CarInfo struct {
	Make  string
	Model string
	Year  int
	Mpg   float64
	Url   string
}

func ModelScrape(client *http.Client, Make string, Model string, Url string) CarInfo {

	res, err := client.Get(Url)
	if err != nil {
		log.Fatal(err)
	}

	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		log.Fatal(err)
	}

	var car CarInfo

	fmt.Printf("Parsing %s\n", Url)
	doc.Find("ul.model-year-summary").Each(func(i int, s *goquery.Selection) {

		car.Make = Make
		car.Model = Model
		s.Find(".summary-avg-data").Each(func(y int, m *goquery.Selection) {
			car.Mpg, _ = strconv.ParseFloat(m.Text(), 32)
		})
		s.Find(".summary-year").Each(func(y int, m *goquery.Selection) {
			car.Year, _ = strconv.Atoi(m.Text())
		})
		car.Url = Url
	})

	return car
}

func getCarData(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	carMake := r.URL.Query()["make"]
	mpg := r.URL.Query()["mpg"]
	year := r.URL.Query()["year"]

	var orderBy string
	if len(mpg) == 1 && mpg[0] == "bottom" {
		orderBy = "Mpg"
	} else {
		orderBy = "-Mpg"
	}

	//fmt.Fprintf(w, "%+v", r.URL.Query()["make"])
	type CarDisplay struct {
		Url     string
		Display string
	}

	var dataDisplay []CarDisplay
	q := datastore.NewQuery("CarInfo").
		Order(orderBy).
		Limit(10)

	if len(carMake) == 1 && len(carMake[0]) > 1 {
		q = q.Filter("Make =", carMake[0])
	}

	if len(year) == 1 && len(year[0]) == 4 {
		carYear, _ := strconv.Atoi(year[0])
		q = q.Filter("Year =", carYear)
	}

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
		var info CarDisplay
		info.Url = fmt.Sprintf("%s/%d", car2.Url, car2.Year)
		info.Display = fmt.Sprintf("%d %s %s (%-3.1f)",
			car2.Year,
			strings.Title(strings.Replace(car2.Make, "_", " ", -1)),
			strings.Title(strings.Replace(car2.Model, "_", " ", -1)),
			car2.Mpg)
		dataDisplay = append(dataDisplay, info)
	}

	dataJson, _ := json.Marshal(dataDisplay)

	fmt.Fprintf(w, "%s", string(dataJson))
}

func queryDb(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	type CarDisplay struct {
		Url     string
		Display string
	}
	type DataDisplay struct {
		Cars  []CarDisplay
		Makes []string
	}

	var dataDisplay DataDisplay

	q := datastore.NewQuery("CarInfo").
		Order("-Mpg").
		Limit(10)
	//		Project("Make").
	//		Distinct()

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
		var info CarDisplay
		info.Url = car2.Url
		info.Display = fmt.Sprintf("%d %s %s (%-3.1f)", car2.Year, strings.Title(car2.Make), strings.Title(car2.Model), car2.Mpg)
		dataDisplay.Cars = append(dataDisplay.Cars, info)
	}

	q = datastore.NewQuery("CarInfo").
		Project("Make").
		Distinct()

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

		dataDisplay.Makes = append(dataDisplay.Makes, car2.Make)
	}

	fmt.Fprintf(w, "Db generated")
}

func parseCar(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	client := urlfetch.Client(c)
	url := r.FormValue("url")
	fmt.Fprintf(w, url)

	urlParts := strings.Split(url, "/")
	fmt.Fprintf(w, urlParts[0])

	var car CarInfo = ModelScrape(client, urlParts[len(urlParts)-2], urlParts[len(urlParts)-1], url)

	if len(car.Make) > 0 && len(car.Model) > 0 {
		// Now lets check to see if the car exists first...
		q := datastore.NewQuery("CarInfo").
			Filter("Make =", car.Make).
			Filter("Model =", car.Model).
			Filter("Year =", car.Year)
		t := q.Run(c)

		var existingCar CarInfo

		_, err := t.Next(&existingCar)
		if err == datastore.Done {
			// Car does not exist so store it away
			_, err := datastore.Put(c, datastore.NewIncompleteKey(c, "CarInfo", nil), &car)
			if err != nil {
				return
			}
		} else {
			// Car already exists so do nothing...
			fmt.Fprintf(w, "Pre-existing data %+v\n", existingCar)
		}

	}
}

func getData(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	client := urlfetch.Client(c)
	res, err := client.Get("http://www.fuelly.com/car/")
	if err != nil {
		log.Fatal(err)
	}

	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		log.Fatal(err)
	}

	count := 0
	doc.Find(".models-list").Each(func(i int, s *goquery.Selection) {
		s.Find("a").Each(func(y int, m *goquery.Selection) {
			_, exists := m.Attr("href")
			if exists == true {
				count = count + 1
				//fmt.Fprintf(w, "%s\n", "%d:%s\n", count, url)
			}
		})
	})

	fmt.Fprintf(w, "Found %d\n", count)

	//car := make(chan CarInfo, 200)
	//sem := make(chan int, 20)
	//countChan := make(chan int, count)

	doc.Find(".models-list").Each(func(i int, s *goquery.Selection) {
		s.Find("a").Each(func(y int, m *goquery.Selection) {
			modelUrl, exists := m.Attr("href")
			if exists == true {

				//fmt.Fprintf(w, "%s\n", urlParts)

				t := taskqueue.NewPOSTTask("/parseCar", map[string][]string{"url": {modelUrl}})
				if _, err := taskqueue.Add(c, t, ""); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				//go ModelScrape(client, sem, car, countChan, urlParts[len(urlParts)-2], m.Text(), modelUrl)
			}
		})
	})

	fmt.Fprintf(w, "%s\n", "DB create")

	fmt.Fprintf(w, "%s\n", "DB Done")
}
